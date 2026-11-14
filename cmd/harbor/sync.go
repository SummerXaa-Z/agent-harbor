package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SummerXaa-Z/agent-harbor/internal/wrap"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	syncBatchEvents = 100
	syncMaxLine     = 64 << 10
	syncMaxBatch    = 512 << 10
	syncMaxResponse = 64 << 10
)

type syncCheckpoint struct {
	Offset       int64  `json:"offset"`
	PrefixSHA256 string `json:"prefixSHA256"`
}
type syncRemoteError struct {
	code int
	err  error
}

func (e *syncRemoteError) Error() string { return e.err.Error() }
func (e *syncRemoteError) Unwrap() error { return e.err }

func cmdSync(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("sync", stderr)
	clientFlags := addClientFlags(flags)
	home, _ := os.UserHomeDir()
	replay := flags.Bool("replay", false, "replay retained events even if previously acknowledged")
	spool := flags.String("spool", filepath.Join(home, ".harbor", "spool"), "wrapper audit spool directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if len(flags.Args()) != 0 {
		fmt.Fprintln(stderr, "harbor sync: unexpected arguments")
		return 2
	}
	client := clientFromFlags(clientFlags)
	// Never forward an administrative credential to a redirect destination.
	client.httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	count, err := syncSpool(client, *spool, *replay)
	if err != nil {
		fmt.Fprintf(stderr, "harbor sync: %v (checkpointed %d events; spool preserved)\n", err, count)
		var remote *syncRemoteError
		if errors.As(err, &remote) {
			return remote.code
		}
		return 2
	}
	fmt.Fprintf(stdout, "synced %d events; local spool retained\n", count)
	return 0
}

// Destination-specific cursors prevent another server's ack from skipping data.
// Key rotation safely replays retained events; no plaintext key is stored.
func syncCheckpointDir(spool, endpoint, adminKey string) string {
	digest := sha256.Sum256([]byte(strings.TrimRight(endpoint, "/") + "\x00" + adminKey))
	return filepath.Join(spool, ".sync-"+hex.EncodeToString(digest[:]))
}
func syncSpool(client *cliClient, spool string, replay bool) (int, error) {
	entries, err := os.ReadDir(spool)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	checkpointDir := syncCheckpointDir(spool, client.baseURL, client.adminKey)
	if info, err := os.Lstat(checkpointDir); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return 0, fmt.Errorf("checkpoint path is not a regular directory")
		}
	} else if !os.IsNotExist(err) {
		return 0, err
	}
	total := 0
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "wrap-") || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		if !entry.Type().IsRegular() {
			return total, fmt.Errorf("spool %s is not a regular file", entry.Name())
		}
		n, err := syncSpoolFile(client, filepath.Join(spool, entry.Name()), filepath.Join(checkpointDir, entry.Name()+".json"), replay)
		total += n
		if err != nil {
			return total, fmt.Errorf("%s: %w", entry.Name(), err)
		}
	}
	return total, nil
}
func prefixDigest(file *os.File, n int64) (string, error) {
	hash := sha256.New()
	if _, err := io.CopyN(hash, io.NewSectionReader(file, 0, n), n); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func readSyncCheckpoint(path string, file *os.File, size int64) (syncCheckpoint, error) {
	var cp syncCheckpoint
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return cp, nil
	}
	if err != nil {
		return cp, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil {
		return cp, err
	}
	if len(data) > 4096 || json.Unmarshal(data, &cp) != nil || cp.Offset < 0 {
		return cp, fmt.Errorf("invalid checkpoint; restore or remove it to safely replay")
	}
	if cp.Offset > size {
		return syncCheckpoint{}, nil
	}
	digest, err := prefixDigest(file, cp.Offset)
	if err != nil {
		return cp, err
	}
	if digest != cp.PrefixSHA256 {
		return syncCheckpoint{}, nil
	}
	return cp, nil
}
func saveSyncCheckpoint(path string, cp syncCheckpoint) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".checkpoint-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = json.NewEncoder(file).Encode(cp); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
func syncSpoolFile(client *cliClient, path, checkpointPath string, replay bool) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("not a regular spool")
	}
	var cp syncCheckpoint
	if !replay {
		cp, err = readSyncCheckpoint(checkpointPath, file, info.Size())
		if err != nil {
			return 0, err
		}
	}
	// Snapshot length: concurrent append and incomplete final lines wait for
	// the next invocation. Never truncate a writer's spool.
	reader := bufio.NewReaderSize(io.NewSectionReader(file, cp.Offset, info.Size()-cp.Offset), syncMaxLine+1)
	offset := cp.Offset
	prefixHash := sha256.New()
	if _, err := io.CopyN(prefixHash, io.NewSectionReader(file, 0, cp.Offset), cp.Offset); err != nil {
		return 0, err
	}
	if cp.Offset > 0 && hex.EncodeToString(prefixHash.Sum(nil)) != cp.PrefixSHA256 {
		return 0, fmt.Errorf("spool prefix changed while opening; retry")
	}
	batch := make([]wrap.Event, 0, syncBatchEvents)
	batchBytes, total := 0, 0
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := client.ingestWrapEvents(batch); err != nil {
			var remote *syncRemoteError
			if errors.As(err, &remote) {
				return err
			}
			return &syncRemoteError{code: 4, err: err}
		}
		// Hash exactly the acknowledged bytes, not a fresh reread. External
		// rewrites invalidate resume; append remains safe and linear-time.
		digest := hex.EncodeToString(prefixHash.Sum(nil))
		if err := saveSyncCheckpoint(checkpointPath, syncCheckpoint{Offset: offset, PrefixSHA256: digest}); err != nil {
			return fmt.Errorf("save checkpoint: %w", err)
		}
		total += len(batch)
		batch = batch[:0]
		batchBytes = 0
		return nil
	}
	for {
		line, readErr := reader.ReadSlice('\n')
		if readErr == io.EOF {
			err := flush()
			return total, err
		}
		if readErr != nil {
			return total, fmt.Errorf("read spool line (maximum %d bytes): %w", syncMaxLine, readErr)
		}
		if len(line) > syncMaxLine {
			return total, fmt.Errorf("spool line exceeds %d bytes", syncMaxLine)
		}
		if batchBytes+len(line) > syncMaxBatch {
			if err := flush(); err != nil {
				return total, err
			}
		}
		var event wrap.Event
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&event); err != nil {
			return total, fmt.Errorf("invalid event at offset %d: %w", offset, err)
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return total, fmt.Errorf("trailing event data at offset %d", offset)
		}
		if event.ID == "" || event.Source != "wrap" {
			return total, fmt.Errorf("invalid wrapper event at offset %d", offset)
		}
		batch = append(batch, event)
		_, _ = prefixHash.Write(line)
		batchBytes += len(line)
		offset += int64(len(line))
		if len(batch) == syncBatchEvents {
			if err := flush(); err != nil {
				return total, err
			}
		}
	}
}
func (client *cliClient) ingestWrapEvents(events []wrap.Event) error {
	body, err := json.Marshal(struct {
		Events []wrap.Event `json:"events"`
	}{events})
	if err != nil {
		return err
	}
	if len(body) > syncMaxBatch+1024 {
		return fmt.Errorf("encoded event batch too large")
	}
	request, err := http.NewRequest(http.MethodPost, client.baseURL+"/api/v1/audit/traces/ingest", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if client.adminKey != "" {
		request.Header.Set("X-Admin-Key", client.adminKey)
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, syncMaxResponse+1))
	if err != nil {
		return err
	}
	if len(data) > syncMaxResponse {
		return fmt.Errorf("server response too large")
	}
	var envelope apiEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("decode ingest response: %w", err)
	}
	if response.StatusCode != http.StatusOK || envelope.Error != "" || envelope.Code != 0 {
		code := 4
		if response.StatusCode == http.StatusForbidden {
			code = 3
		}
		return &syncRemoteError{code: code, err: fmt.Errorf("ingest rejected (HTTP %d): %s", response.StatusCode, envelope.Error)}
	}
	var result struct {
		Accepted *int `json:"accepted"`
	}
	if err := json.Unmarshal(envelope.Data, &result); err != nil || result.Accepted == nil || *result.Accepted != len(events) {
		return fmt.Errorf("server did not acknowledge the complete batch")
	}
	return nil
}
