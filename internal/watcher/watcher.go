package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/01x-in/codeindex/internal/config"
	"github.com/01x-in/codeindex/internal/indexer"
	"github.com/fsnotify/fsnotify"
)

const debounceDuration = 100 * time.Millisecond

// Watcher provides fsnotify-based file watching for auto-reindex.
// It watches a directory tree recursively, filters events by configured
// languages and ignore paths, and debounces rapid writes before calling
// the onChange callback.
type Watcher struct {
	dir      string
	cfg      config.Config
	onChange func(path string)
	exts     map[string]bool // language file extensions to watch
}

// New creates a Watcher rooted at dir. onChange is called (at most once per
// debounce window) for each changed file that matches a configured language
// and is not under an ignored directory.
func New(dir string, cfg config.Config, onChange func(path string)) (*Watcher, error) {
	exts := buildExtSet(cfg.Languages, indexer.LanguageExtensions)
	return &Watcher{
		dir:      dir,
		cfg:      cfg,
		onChange: onChange,
		exts:     exts,
	}, nil
}

// Start begins watching. It blocks until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer fw.Close()

	// Walk the directory tree and add every non-ignored directory.
	if err := w.addDirs(fw, w.dir); err != nil {
		return err
	}

	// Per-file debounce timers.
	var timerMu sync.Mutex
	timers := make(map[string]*time.Timer)

	fire := func(path string) {
		w.onChange(path)
	}

	schedule := func(path string) {
		timerMu.Lock()
		defer timerMu.Unlock()
		if t, ok := timers[path]; ok {
			t.Reset(debounceDuration)
		} else {
			timers[path] = time.AfterFunc(debounceDuration, func() {
				fire(path)
				timerMu.Lock()
				delete(timers, path)
				timerMu.Unlock()
			})
		}
	}

	for {
		select {
		case <-ctx.Done():
			// Cancel all pending timers.
			timerMu.Lock()
			for _, t := range timers {
				t.Stop()
			}
			timerMu.Unlock()
			return ctx.Err()

		case event, ok := <-fw.Events:
			if !ok {
				return nil
			}

			// When a new directory is created, add it to the watcher.
			if event.Has(fsnotify.Create) {
				info, statErr := os.Stat(event.Name)
				if statErr == nil && info.IsDir() {
					_ = w.addDirs(fw, event.Name)
					continue
				}
			}

			// Only handle write and create events on files.
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			path := event.Name
			if !w.shouldProcess(path) {
				continue
			}

			schedule(path)

		case _, ok := <-fw.Errors:
			if !ok {
				return nil
			}
			// Ignore watcher errors — they're non-fatal (e.g., permission denied).
		}
	}
}

// addDirs recursively adds all non-ignored directories under root to the watcher.
func (w *Watcher) addDirs(fw *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if !info.IsDir() {
			return nil
		}
		if w.isIgnored(path) {
			return filepath.SkipDir
		}
		return fw.Add(path)
	})
}

// shouldProcess returns true if the file at path should trigger a reindex.
func (w *Watcher) shouldProcess(path string) bool {
	if w.isIgnored(path) {
		return false
	}
	ext := filepath.Ext(path)
	return w.exts[ext]
}

// isIgnored returns true if path is under any configured ignore directory.
func (w *Watcher) isIgnored(path string) bool {
	// Make path relative to the root for comparison.
	rel, err := filepath.Rel(w.dir, path)
	if err != nil {
		return false
	}

	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts {
		for _, ignored := range w.cfg.Ignore {
			if part == ignored {
				return true
			}
		}
		// Always ignore the index storage directory.
		if part == w.cfg.IndexPath || part == ".codeindex" {
			return true
		}
	}
	return false
}

// buildExtSet builds a set of file extensions to watch for the given languages.
// extsFn is injected so callers can substitute indexer.LanguageExtensions (or a
// test double) without coupling watcher directly to the indexer package at the
// call site.
func buildExtSet(languages []string, extsFn func(string) []string) map[string]bool {
	exts := make(map[string]bool)
	for _, lang := range languages {
		for _, ext := range extsFn(lang) {
			exts[ext] = true
		}
	}
	return exts
}
