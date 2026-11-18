package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func wrapAttempt(f *wrapFixture, sub string) (int, string) {
	args := []string{"wrap", sub, "--state", f.state}
	if sub == "add" {
		args = append(args, "--wrap-bin", f.wrapBin, "--policy", f.policy)
	}
	args = append(args, "claude-code", "node_repl")
	var out, errOut strings.Builder
	code := run(args, &out, &errOut)
	return code, out.String() + errOut.String()
}

func TestWrapSafetyExplicitPolicyAndPrivateCopies(t *testing.T) {
	f := setupWrapFixture(t)
	f.wrap(t, "add", "claude-code", "node_repl")
	args := stringArgs(f.entry(t, "node_repl")["args"])
	found := false
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--policy" {
			found = args[i+1] == f.policy && filepath.IsAbs(args[i+1])
			break
		}
	}
	if !found {
		t.Errorf("wrapper does not select exact absolute policy: %v", args)
	}
	for _, path := range []string{f.config, f.config + ".harbor.bak", f.state} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("%s permissions=%o, want 600", filepath.Base(path), info.Mode().Perm())
		}
	}
}

func TestWrapSafetyStateFailureDoesNotInstall(t *testing.T) {
	f := setupWrapFixture(t)
	f.state = filepath.Join(filepath.Dir(f.state), "missing", "state.json")
	before := mustReadFile(t, f.config)
	if code, output := wrapAttempt(f, "add"); code == 0 {
		t.Fatalf("state failure reported success: %s", output)
	}
	if got := mustReadFile(t, f.config); got != before {
		t.Fatal("config changed after state failure")
	}
}

func TestWrapSafetyImportStateIsNotDestroyed(t *testing.T) {
	f := setupWrapFixture(t)
	original := `{"targets":{"existing":"target-id"},"clients":{"claude-code":"agent-id"},"future":{"keep":true}}`
	if err := os.WriteFile(f.state, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	code, _ := wrapAttempt(f, "add")
	var got map[string]any
	if err := json.Unmarshal([]byte(mustReadFile(t, f.state)), &got); err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal([]byte(original), &want); err != nil {
		t.Fatal(err)
	}
	for key, value := range want {
		if !reflect.DeepEqual(got[key], value) {
			t.Errorf("state field %s lost", key)
		}
	}
	if code != 0 && mustReadFile(t, f.config) != wrapClientConfig {
		t.Fatal("rejected state changed config")
	}
}

func TestWrapSafetyRemoveRejectsEditedEntry(t *testing.T) {
	f := setupWrapFixture(t)
	f.wrap(t, "add", "claude-code", "node_repl")
	doc, err := readJSONDocument(f.config)
	if err != nil {
		t.Fatal(err)
	}
	entry := doc["mcpServers"].(map[string]any)["node_repl"].(map[string]any)
	entry["env"] = map[string]any{"TOKEN": "new-synthetic-value"}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.config, raw, 0600); err != nil {
		t.Fatal(err)
	}
	beforeState := mustReadFile(t, f.state)
	if code, output := wrapAttempt(f, "remove"); code == 0 {
		t.Fatalf("edited entry overwritten: %s", output)
	}
	if mustReadFile(t, f.config) != string(raw) {
		t.Fatal("remove changed edited config")
	}
	if mustReadFile(t, f.state) != beforeState {
		t.Fatal("remove lost recovery state on conflict")
	}
}

func TestWrapSafetyPreservesEntryMetadata(t *testing.T) {
	f := setupWrapFixture(t)
	doc, err := readJSONDocument(f.config)
	if err != nil {
		t.Fatal(err)
	}
	entry := doc["mcpServers"].(map[string]any)["node_repl"].(map[string]any)
	entry["disabled"] = true
	entry["custom"] = map[string]any{"setting": "retained"}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.config, raw, 0600); err != nil {
		t.Fatal(err)
	}
	f.wrap(t, "add", "claude-code", "node_repl")
	got := f.entry(t, "node_repl")
	for _, key := range []string{"disabled", "custom", "env"} {
		if !reflect.DeepEqual(got[key], entry[key]) {
			t.Errorf("metadata %s lost", key)
		}
	}
	f.wrap(t, "remove", "claude-code", "node_repl")
	if !reflect.DeepEqual(f.entry(t, "node_repl"), entry) {
		t.Fatal("original entry not restored")
	}
}

func TestWrapSafetyRejectsMalformedArguments(t *testing.T) {
	for _, bad := range []string{`"not-an-array"`, `["ok",42]`, `[null]`, `{}`} {
		t.Run(bad, func(t *testing.T) {
			f := setupWrapFixture(t)
			raw := strings.Replace(wrapClientConfig, `["--verbose", "--port", "1"]`, bad, 1)
			if err := os.WriteFile(f.config, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			if code, output := wrapAttempt(f, "add"); code == 0 {
				t.Fatalf("accepted malformed arguments: %s", output)
			}
			if mustReadFile(t, f.config) != raw {
				t.Fatal("invalid arguments changed config")
			}
		})
	}
}

func TestWrapSafetyRejectsPathCollisionAndUnsafeFiles(t *testing.T) {
	for _, kind := range []string{"same-file", "symlink", "hardlink", "backup-directory", "locked"} {
		t.Run(kind, func(t *testing.T) {
			f := setupWrapFixture(t)
			before := mustReadFile(t, f.config)
			switch kind {
			case "same-file":
				f.state = f.config
			case "symlink":
				if err := os.Symlink(f.config, f.state); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(f.config, f.state); err != nil {
					t.Fatal(err)
				}
			case "backup-directory":
				if err := os.Mkdir(f.config+".harbor.bak", 0700); err != nil {
					t.Fatal(err)
				}
			case "locked":
				if err := os.WriteFile(f.config+".harbor-lock", []byte("held"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if code, output := wrapAttempt(f, "add"); code == 0 {
				t.Fatalf("unsafe path accepted: %s", output)
			}
			if mustReadFile(t, f.config) != before {
				t.Fatal("config changed on refusal")
			}
			if _, err := os.Stat(f.policy); !os.IsNotExist(err) {
				t.Fatal("policy created on refusal")
			}
		})
	}
}

func TestWrapSafetyTransactionRollsBackAndReleasesLocks(t *testing.T) {
	f := setupWrapFixture(t)
	paths := []string{f.config, f.config + ".harbor.bak", f.state, f.policy}
	before := mustReadFile(t, f.config)
	err := wrapTransaction(paths, func(write wrapWriteFunc) error {
		if err := write(f.state, []byte("recovery")); err != nil {
			return err
		}
		if err := write(f.config, []byte("changed")); err != nil {
			return err
		}
		if err := write(f.policy, []byte("policy")); err != nil {
			return err
		}
		return os.ErrPermission
	})
	if err == nil {
		t.Fatal("lost transaction failure")
	}
	if mustReadFile(t, f.config) != before {
		t.Fatal("config not rolled back")
	}
	for _, path := range []string{f.state, f.policy} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("new file remains: %s", path)
		}
	}
	for _, path := range paths {
		if _, err := os.Stat(path + ".harbor-lock"); !os.IsNotExist(err) {
			t.Fatalf("lock remains: %s", path)
		}
	}
	f.wrap(t, "add", "claude-code", "node_repl")
}

func TestWrapSafetyRemoveRecoversInterruptedInstallOrRemoval(t *testing.T) {
	f := setupWrapFixture(t)
	before := mustReadFile(t, f.config)
	f.wrap(t, "add", "claude-code", "node_repl")
	if err := os.WriteFile(f.config, []byte(before), 0600); err != nil {
		t.Fatal(err)
	}
	f.wrap(t, "remove", "claude-code", "node_repl")
	state, err := loadOrCreateWrapState(f.state)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Wrapped) != 0 {
		t.Fatal("stale recovery record remains")
	}
	if mustReadFile(t, f.config) != before {
		t.Fatal("recovery changed original config")
	}
}

func TestWrapSafetyTransactionPreservesConcurrentEdit(t *testing.T) {
	f := setupWrapFixture(t)
	err := wrapTransaction([]string{f.config, f.config + ".harbor.bak", f.state, f.policy}, func(write wrapWriteFunc) error {
		if err := write(f.state, []byte("recovery evidence")); err != nil {
			return err
		}
		if err := write(f.config, []byte("new config")); err != nil {
			return err
		}
		return os.WriteFile(f.config, []byte("external edit"), 0600)
	})
	if err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("missing conflict error: %v", err)
	}
	if mustReadFile(t, f.config) != "external edit" {
		t.Fatal("overwrote concurrent edit")
	}
	if _, err := os.Stat(f.state); !os.IsNotExist(err) {
		t.Fatal("committed state despite conflict")
	}
}

func TestWrapSafetyRejectsMalformedState(t *testing.T) {
	for _, raw := range []string{`{"wrapped":{}}`, `{"version":1,"wrapped":{}} {}`, `{"version":1,"wrapped":null}`} {
		t.Run(raw, func(t *testing.T) {
			f := setupWrapFixture(t)
			if err := os.WriteFile(f.state, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			if code, _ := wrapAttempt(f, "add"); code == 0 {
				t.Fatal("accepted malformed state")
			}
			if mustReadFile(t, f.state) != raw || mustReadFile(t, f.config) != wrapClientConfig {
				t.Fatal("changed files on rejection")
			}
		})
	}
}

func TestWrapSafetyCommitFailureRestoresFilesOrKeepsRecovery(t *testing.T) {
	for _, failRollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "rollback", true: "retain-recovery"}[failRollback], func(t *testing.T) {
			f := setupWrapFixture(t)
			before := mustReadFile(t, f.config)
			err := wrapTransactionWithWriter([]string{f.config, f.config + ".harbor.bak", f.state, f.policy}, func(write wrapWriteFunc) error {
				for _, item := range []struct{ path, data string }{{f.state, "recovery"}, {f.config, "changed"}, {f.policy, "policy"}} {
					if err := write(item.path, []byte(item.data)); err != nil {
						return err
					}
				}
				return nil
			}, func(path string, data []byte) error {
				if path == f.policy || (failRollback && path == f.config && string(data) == before) {
					return os.ErrPermission
				}
				return atomicWrapWrite(path, data)
			})
			if err == nil {
				t.Fatal("lost persistence error")
			}
			if failRollback {
				if !strings.Contains(err.Error(), "rollback") {
					t.Fatal("rollback failure not reported")
				}
				if mustReadFile(t, f.state) != "recovery" {
					t.Fatal("lost recovery record")
				}
			} else {
				if mustReadFile(t, f.config) != before {
					t.Fatal("did not restore config")
				}
				if _, err := os.Stat(f.state); !os.IsNotExist(err) {
					t.Fatal("did not restore absent state")
				}
			}
		})
	}
}
