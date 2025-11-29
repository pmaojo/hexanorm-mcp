package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/analysis"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/config"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/graph"
)

// Watcher monitors the filesystem for changes and triggers incremental analysis.
// It uses fsnotify to detect file creation, modification, and deletion.
type Watcher struct {
	watcher  *fsnotify.Watcher
	analyzer *analysis.Analyzer
	graph    *graph.Graph
	config   *config.Config
}

// NewWatcher initializes a new Watcher for the specified root directory.
// It recursively adds all subdirectories to the watch list, excluding those ignored by config.
func NewWatcher(rootDir string, analyzer *analysis.Analyzer, g *graph.Graph, cfg *config.Config) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:  fw,
		analyzer: analyzer,
		graph:    g,
		config:   cfg,
	}

	// Add root recursively
	if err := w.addRecursive(rootDir); err != nil {
		fw.Close()
		return nil, err
	}

	return w, nil
}

// Start begins the event loop for monitoring file changes.
// It runs in a separate goroutine.
func (w *Watcher) Start() {
	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				w.handleEvent(event)
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			}
		}
	}()
}

// Close stops the watcher and releases resources.
func (w *Watcher) Close() error {
	return w.watcher.Close()
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	if w.shouldIgnore(event.Name) {
		return
	}

	if event.Has(fsnotify.Create) {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			w.watcher.Add(event.Name)
			w.addRecursive(event.Name)
		} else {
			w.analyzeFile(event.Name)
		}
	} else if event.Has(fsnotify.Write) {
		w.analyzeFile(event.Name)
	} else if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		// Remove from graph
		w.graph.RemoveNode(event.Name)
		// If it was a directory, fsnotify usually removes the watch automatically, but we assume file-based graph for now.
	}
}

func (w *Watcher) analyzeFile(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Failed to read file %s: %v", path, err)
		return
	}
	if err := w.analyzer.AnalyzeFile(path, content); err != nil {
		log.Printf("Failed to analyze file %s: %v", path, err)
	} else {
		log.Printf("Analyzed %s", path)
	}
}

func (w *Watcher) addRecursive(path string) error {
	return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if w.shouldIgnore(p) {
				return filepath.SkipDir
			}
			return w.watcher.Add(p)
		}
		return nil
	})
}

func (w *Watcher) shouldIgnore(path string) bool {
	base := filepath.Base(path)
	// Check config excludes
	for _, excl := range w.config.ExcludedDirs {
		if strings.Contains(path, excl) || base == excl {
			return true
		}
	}
	// Always ignore .git, .vibecoder
	if base == ".git" || base == ".vibecoder" || strings.Contains(path, "/.vibecoder/") {
		return true
	}
	return false
}
