package watcher_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/01x/codeindex/internal/config"
	"github.com/01x/codeindex/internal/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeConfig(languages []string, ignore []string) config.Config {
	cfg := config.DefaultConfig()
	cfg.Languages = languages
	cfg.Ignore = ignore
	return cfg
}

// TestWatcher_CallsCallbackOnWrite verifies that modifying a file triggers the callback.
func TestWatcher_CallsCallbackOnWrite(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\n"), 0644))

	cfg := makeConfig([]string{"go"}, []string{})

	var mu sync.Mutex
	var received []string

	w, err := watcher.New(dir, cfg, func(path string) {
		mu.Lock()
		received = append(received, path)
		mu.Unlock()
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Start(ctx)
	}()

	// Give the watcher time to start.
	time.Sleep(100 * time.Millisecond)

	// Write to the file.
	require.NoError(t, os.WriteFile(filePath, []byte("package main\n// changed\n"), 0644))

	// Wait for callback (up to 2 seconds).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(received)
		mu.Unlock()
		if count > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	<-errCh

	mu.Lock()
	defer mu.Unlock()
	assert.NotEmpty(t, received, "expected callback to be called on file write")
	assert.Equal(t, filePath, received[0])
}

// TestWatcher_SkipsNonLanguageFiles verifies that files with non-matching extensions are ignored.
func TestWatcher_SkipsNonLanguageFiles(t *testing.T) {
	dir := t.TempDir()

	// Only watching Go files, create a .ts file.
	tsFile := filepath.Join(dir, "index.ts")
	require.NoError(t, os.WriteFile(tsFile, []byte("export {}\n"), 0644))

	cfg := makeConfig([]string{"go"}, []string{})

	var mu sync.Mutex
	var received []string

	w, err := watcher.New(dir, cfg, func(path string) {
		mu.Lock()
		received = append(received, path)
		mu.Unlock()
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Write to the .ts file — should NOT trigger callback.
	require.NoError(t, os.WriteFile(tsFile, []byte("export { foo }\n"), 0644))

	// Wait briefly to confirm no callback fires.
	time.Sleep(500 * time.Millisecond)

	cancel()
	<-errCh

	mu.Lock()
	defer mu.Unlock()
	assert.Empty(t, received, "expected no callback for non-language file")
}

// TestWatcher_SkipsIgnoredPaths verifies that files under ignored directories are skipped.
func TestWatcher_SkipsIgnoredPaths(t *testing.T) {
	dir := t.TempDir()

	// Create an ignored subdirectory.
	ignoredDir := filepath.Join(dir, "vendor")
	require.NoError(t, os.MkdirAll(ignoredDir, 0755))
	ignoredFile := filepath.Join(ignoredDir, "lib.go")
	require.NoError(t, os.WriteFile(ignoredFile, []byte("package lib\n"), 0644))

	cfg := makeConfig([]string{"go"}, []string{"vendor"})

	var mu sync.Mutex
	var received []string

	w, err := watcher.New(dir, cfg, func(path string) {
		mu.Lock()
		received = append(received, path)
		mu.Unlock()
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Write to the file in the ignored dir.
	require.NoError(t, os.WriteFile(ignoredFile, []byte("package lib\n// changed\n"), 0644))

	// Wait briefly to confirm no callback fires.
	time.Sleep(500 * time.Millisecond)

	cancel()
	<-errCh

	mu.Lock()
	defer mu.Unlock()
	assert.Empty(t, received, "expected no callback for file in ignored directory")
}

// TestWatcher_DebouncesBatchedWrites verifies rapid writes produce a single callback.
func TestWatcher_DebouncesBatchedWrites(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\n"), 0644))

	cfg := makeConfig([]string{"go"}, []string{})

	var mu sync.Mutex
	var received []string

	w, err := watcher.New(dir, cfg, func(path string) {
		mu.Lock()
		received = append(received, path)
		mu.Unlock()
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Write to the file 5 times rapidly.
	for i := 0; i < 5; i++ {
		require.NoError(t, os.WriteFile(filePath, []byte("package main\n// v"+string(rune('0'+i))+"\n"), 0644))
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce window to expire + buffer.
	time.Sleep(500 * time.Millisecond)

	cancel()
	<-errCh

	mu.Lock()
	defer mu.Unlock()
	// Should fire once or a small number of times, not 5 times.
	assert.NotEmpty(t, received, "expected at least one callback")
	assert.LessOrEqual(t, len(received), 3, "expected debounce to batch rapid writes, got %d callbacks", len(received))
}

// TestWatcher_StopsOnContextCancel verifies Start returns when context is cancelled.
func TestWatcher_StopsOnContextCancel(t *testing.T) {
	dir := t.TempDir()
	cfg := makeConfig([]string{"go"}, []string{})

	w, err := watcher.New(dir, cfg, func(path string) {})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		// context.Canceled is acceptable; nil is also fine.
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop after context cancellation")
	}
}

// TestWatcher_HandlesNewSubdirectory verifies files created in new subdirs are watched.
func TestWatcher_HandlesNewSubdirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := makeConfig([]string{"go"}, []string{})

	var mu sync.Mutex
	var received []string

	w, err := watcher.New(dir, cfg, func(path string) {
		mu.Lock()
		received = append(received, path)
		mu.Unlock()
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create a new subdirectory and file after watcher starts.
	subDir := filepath.Join(dir, "pkg")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	time.Sleep(150 * time.Millisecond) // let watcher add the new dir

	newFile := filepath.Join(subDir, "util.go")
	require.NoError(t, os.WriteFile(newFile, []byte("package pkg\n"), 0644))

	// Wait for callback.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(received)
		mu.Unlock()
		if count > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	<-errCh

	mu.Lock()
	defer mu.Unlock()
	assert.NotEmpty(t, received, "expected callback for file in new subdirectory")
}
