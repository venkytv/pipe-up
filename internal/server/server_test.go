package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/venkytv/tts-cached/internal/cache"
	"github.com/venkytv/tts-cached/internal/config"
)

func TestHandleTTSSynthAndCache(t *testing.T) {
	dir := t.TempDir()
	fp := &fakePiper{}
	player := &fakePlayer{ch: make(chan string, 1)}

	cfg := config.Config{
		VoiceID:  "default",
		CacheDir: dir,
	}
	mgr := cache.NewManager(dir, 1024*1024, logDiscard)
	srv := New(cfg, mgr, fp, player, logDiscard)

	body := bytes.NewBufferString(`{"text":"  hello   world "}`)
	req := httptest.NewRequest(http.MethodPost, "/tts", body)
	rec := httptest.NewRecorder()

	srv.handleTTS(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ttsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != "cache_miss" {
		t.Fatalf("expected cache_miss, got %s", resp.Status)
	}

	key := cache.BuildKey("default", "hello world")
	wantFile := filepath.Join(dir, key+".wav")
	if _, err := os.Stat(wantFile); err != nil {
		t.Fatalf("expected wav file created, err: %v", err)
	}
	if fp.calls != 1 {
		t.Fatalf("expected piper synth once, got %d", fp.calls)
	}

	select {
	case path := <-player.ch:
		if path != wantFile {
			t.Fatalf("playback path mismatch, got %s", path)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("playback not triggered")
	}
}

func TestHandleTTSCacheHit(t *testing.T) {
	dir := t.TempDir()
	fp := &fakePiper{}
	player := &fakePlayer{ch: make(chan string, 1)}

	cfg := config.Config{
		VoiceID:  "default",
		CacheDir: dir,
	}
	mgr := cache.NewManager(dir, 1024*1024, logDiscard)
	srv := New(cfg, mgr, fp, player, logDiscard)

	key := cache.BuildKey("default", "hello world")
	wav := filepath.Join(dir, key+".wav")
	if err := os.WriteFile(wav, []byte("data"), 0o644); err != nil {
		t.Fatalf("write wav: %v", err)
	}
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(wav, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/tts", bytes.NewBufferString(`{"text":"hello   world"}`))
	rec := httptest.NewRecorder()
	srv.handleTTS(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp ttsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "cache_hit" {
		t.Fatalf("expected cache_hit, got %s", resp.Status)
	}
	if fp.calls != 0 {
		t.Fatalf("piper should not be called on cache hit")
	}
	info, err := os.Stat(wav)
	if err != nil {
		t.Fatalf("stat wav: %v", err)
	}
	if !info.ModTime().After(oldTime) {
		t.Fatalf("mod time not updated on cache hit")
	}

	select {
	case <-player.ch:
	case <-time.After(1 * time.Second):
		t.Fatalf("playback not triggered on cache hit")
	}
}

type fakePiper struct {
	calls int
}

func (f *fakePiper) Synthesize(_ context.Context, _ string, outPath string) error {
	f.calls++
	return os.WriteFile(outPath, []byte("wav"), 0o644)
}

type fakePlayer struct {
	ch chan string
}

func (f *fakePlayer) PlayWav(path string) {
	f.ch <- path
}

var logDiscard = log.New(io.Discard, "", 0)
