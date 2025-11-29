package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Manager manages cache paths and size enforcement.
type Manager struct {
	dir      string
	maxBytes int64
	logger   *log.Logger
}

// NewManager creates a cache manager rooted at dir with a size limit.
func NewManager(dir string, maxBytes int64, logger *log.Logger) Manager {
	if logger == nil {
		logger = log.Default()
	}
	return Manager{dir: dir, maxBytes: maxBytes, logger: logger}
}

// BuildKey returns a sha256 hex digest for the voice/text pair.
func BuildKey(voiceID, text string) string {
	sum := sha256.Sum256([]byte(voiceID + "::" + text))
	return hex.EncodeToString(sum[:])
}

// PathForKey returns the wav file path for a cache key.
func (m Manager) PathForKey(key string) string {
	return filepath.Join(m.dir, key+".wav")
}

// Touch updates mod/access time to now; best-effort.
func (m Manager) Touch(path string) {
	now := time.Now()
	_ = os.Chtimes(path, now, now)
}

// EnforceLimit deletes oldest wav files until total size is within the limit.
func (m Manager) EnforceLimit() error {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return err
	}

	type fileInfo struct {
		path string
		size int64
		mod  time.Time
	}

	var files []fileInfo
	var total int64

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".wav" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		size := info.Size()
		total += size
		files = append(files, fileInfo{
			path: filepath.Join(m.dir, e.Name()),
			size: size,
			mod:  info.ModTime(),
		})
	}

	if total <= m.maxBytes {
		return nil
	}

	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })

	for _, f := range files {
		if total <= m.maxBytes {
			break
		}
		if err := os.Remove(f.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			m.logger.Printf("ERROR: failed to remove cached file %s: %v", f.path, err)
			continue
		}
		total -= f.size
		m.logger.Printf("INFO: evicted %s (size=%d) to enforce cache limit", filepath.Base(f.path), f.size)
	}

	return nil
}
