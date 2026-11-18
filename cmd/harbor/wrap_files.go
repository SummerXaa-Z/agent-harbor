package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func atomicWrapWrite(path string, data []byte) error {
	if err := wrapRegularPath(path); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".harbor-write-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func wrapRegularPath(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing non-regular file %s", path)
	}
	if info.Sys() != nil { // Hard-linked files can expose saved credentials through another name.
		if links := wrapLinkCount(info); links > 1 {
			return fmt.Errorf("refusing hard-linked file %s", path)
		}
	}
	return nil
}

type wrapSnapshot struct {
	path   string
	data   []byte
	exists bool
}

type wrapWriteFunc func(string, []byte) error

// Locks coordinate wrap commands; byte comparisons detect intervening edits.
// Each file replacement is atomic, but crashes can require recovery via remove.
func wrapTransaction(paths []string, action func(wrapWriteFunc) error) error {
	return wrapTransactionWithWriter(paths, action, atomicWrapWrite)
}

func wrapTransactionWithWriter(paths []string, action func(wrapWriteFunc) error, persist wrapWriteFunc) error {
	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, ".harbor")
	for _, path := range paths {
		if filepath.Dir(path) == defaultDir {
			if err := os.MkdirAll(defaultDir, 0o700); err != nil {
				return err
			}
		}
	}
	snapshots := map[string]wrapSnapshot{}
	ordered := make([]string, 0, len(paths))
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
		if err != nil {
			return err
		}
		canonical := filepath.Join(parent, filepath.Base(abs))
		for _, other := range ordered {
			if other == canonical {
				return fmt.Errorf("config, backup, policy and state paths must be distinct")
			}
		}
		ordered = append(ordered, canonical)
	}
	sort.Strings(ordered)
	var locks []string
	defer func() {
		for _, path := range locks {
			_ = os.Remove(path)
		}
	}()
	for _, path := range ordered {
		if err := wrapRegularPath(path); err != nil {
			return err
		}
		lock := path + ".harbor-lock"
		file, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("lock %s: %w (after a crash, inspect recovery state before removing its lock)", path, err)
		}
		locks = append(locks, lock)
		if err := file.Close(); err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		snapshots[path] = wrapSnapshot{path: path, data: raw, exists: err == nil}
	}
	staged := map[string][]byte{}
	var writes []string
	stage := func(path string, data []byte) error {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
		if err != nil {
			return err
		}
		path = filepath.Join(parent, filepath.Base(abs))
		if _, ok := snapshots[path]; !ok {
			return fmt.Errorf("write outside transaction: %s", path)
		}
		if _, ok := staged[path]; !ok {
			writes = append(writes, path)
		}
		staged[path] = bytes.Clone(data)
		return nil
	}
	if err := action(stage); err != nil {
		return err
	}
	for _, snapshot := range snapshots {
		if !wrapUnchanged(snapshot) {
			return fmt.Errorf("file changed during wrap operation: %s; retry after closing other editors", snapshot.path)
		}
	}
	var committed []string
	for _, path := range writes {
		var err error
		if !wrapUnchanged(snapshots[path]) {
			err = fmt.Errorf("file changed during wrap operation: %s", path)
		} else {
			err = persist(path, staged[path])
		}
		if err == nil {
			committed = append(committed, path)
			continue
		}
		failures := []error{err}
		// Keep persisted recovery state if restoring any other file fails.
		recoveryAbs, _ := filepath.Abs(paths[2])
		recoveryParent, _ := filepath.EvalSymlinks(filepath.Dir(recoveryAbs))
		recovery := filepath.Join(recoveryParent, filepath.Base(recoveryAbs))
		sort.SliceStable(committed, func(i, j int) bool { return committed[i] == recovery && committed[j] != recovery })
		for i := len(committed) - 1; i >= 0; i-- {
			written := committed[i]
			if written == recovery && len(failures) > 1 {
				continue
			}
			expected := wrapSnapshot{path: written, data: staged[written], exists: true}
			var rollbackErr error
			if !wrapUnchanged(expected) {
				rollbackErr = fmt.Errorf("file changed after write")
			} else if snapshots[written].exists {
				rollbackErr = persist(written, snapshots[written].data)
			} else {
				rollbackErr = os.Remove(written)
			}
			if rollbackErr != nil {
				failures = append(failures, fmt.Errorf("rollback %s: %w; preserve recovery files and reconcile manually", written, rollbackErr))
			}
		}
		return errors.Join(failures...)
	}
	return nil
}

func wrapUnchanged(snapshot wrapSnapshot) bool {
	if err := wrapRegularPath(snapshot.path); err != nil {
		return false
	}
	current, err := os.ReadFile(snapshot.path)
	if !snapshot.exists {
		return os.IsNotExist(err)
	}
	return err == nil && bytes.Equal(current, snapshot.data)
}
