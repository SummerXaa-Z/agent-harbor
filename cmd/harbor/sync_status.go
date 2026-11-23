package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SummerXaa-Z/agent-harbor/internal/wrap"
)

type syncStatus struct {
	pending    int
	incomplete int
	synced     bool
	writeLost  bool
}

func reportSyncStatus(client *cliClient, spool string, stdout, stderr io.Writer) int {
	status, err := inspectSyncSpool(client, spool)
	if err != nil {
		if status.writeLost {
			fmt.Fprintln(stderr, "audit spool: previous write failed; records may be missing")
		}
		fmt.Fprintf(stderr, "audit spool: blocked (pending count unavailable): %v\n", err)
		return 2
	}
	state := "empty"
	code := 0
	switch {
	case status.writeLost:
		state = "degraded (previous write failed; records may be missing)"
		code = 2
	case status.pending > 0 || status.incomplete > 0:
		state = "pending"
	case status.synced:
		state = "synced"
	}
	fmt.Fprintf(stdout, "audit spool: %s; %d pending events; %d incomplete files\n", state, status.pending, status.incomplete)
	return code
}

func inspectSyncSpool(client *cliClient, spool string) (syncStatus, error) {
	var status syncStatus
	entries, err := os.ReadDir(spool)
	if os.IsNotExist(err) {
		return status, nil
	}
	if err != nil {
		return status, err
	}
	for _, entry := range entries {
		if entry.Name() == wrap.SpoolWriteFailureMarker {
			status.writeLost = true
		}
	}
	dir, err := checkedSyncCheckpointDir(spool, client)
	if err != nil {
		return status, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !isSpoolRecord(name) {
			continue
		}
		if !entry.Type().IsRegular() {
			return status, fmt.Errorf("spool %s is not a regular file", name)
		}
		if err := status.inspectFile(filepath.Join(spool, name), filepath.Join(dir, name+".json")); err != nil {
			return status, fmt.Errorf("%s: %w", name, err)
		}
	}
	return status, nil
}

func (s *syncStatus) inspectFile(path, checkpoint string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular spool")
	}
	cp, err := readSyncCheckpoint(checkpoint, file, info.Size())
	if err != nil {
		return err
	}
	s.synced = s.synced || cp.Offset > 0
	offset := cp.Offset
	reader := bufio.NewReaderSize(io.NewSectionReader(file, offset, info.Size()-offset), syncMaxLine+1)
	for {
		line, err := reader.ReadSlice('\n')
		if err == io.EOF {
			if len(line) > 0 {
				s.incomplete++
			}
			return nil
		}
		if err != nil {
			return err
		}
		if len(line) > syncMaxLine {
			return fmt.Errorf("spool line exceeds %d bytes", syncMaxLine)
		}
		if _, err := decodeSyncEvent(line, offset); err != nil {
			return err
		}
		s.pending++
		offset += int64(len(line))
	}
}
