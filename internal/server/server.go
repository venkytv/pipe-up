package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/venkytv/tts-cached/internal/cache"
	"github.com/venkytv/tts-cached/internal/config"
)

// Piper synthesizes text to an output wav file path.
type Piper interface {
	Synthesize(ctx context.Context, text, outPath string) error
}

// Player handles wav playback.
type Player interface {
	PlayWav(path string)
}

// Server bundles HTTP handlers for the TTS cache service.
type Server struct {
	cfg    config.Config
	cache  cache.Manager
	piper  Piper
	player Player
	logger *log.Logger
}

// New constructs a server with dependencies.
func New(cfg config.Config, cacheMgr cache.Manager, piper Piper, player Player, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}
	return &Server{
		cfg:    cfg,
		cache:  cacheMgr,
		piper:  piper,
		player: player,
		logger: logger,
	}
}

// Handler returns an http.Handler with registered routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/tts", s.handleTTS)
	mux.HandleFunc("/healthz", s.handleHealth)
	return mux
}

type ttsRequest struct {
	Text string `json:"text"`
}

type ttsResponse struct {
	Status string `json:"status"`
	File   string `json:"file"`
}

func (s *Server) handleTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ttsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	normalized := strings.Join(strings.Fields(req.Text), " ")
	if normalized == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	key := cache.BuildKey(s.cfg.VoiceID, normalized)
	filename := key + ".wav"
	wavPath := s.cache.PathForKey(key)

	if _, err := os.Stat(wavPath); err == nil {
		s.cache.Touch(wavPath)
		go s.player.PlayWav(wavPath)
		s.logger.Printf("INFO: /tts cache_hit key=%s file=%s", key, filename)
		s.writeJSON(w, http.StatusOK, ttsResponse{Status: "cache_hit", File: filename})
		return
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		s.logger.Printf("ERROR: stat cache file failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := s.piper.Synthesize(ctx, normalized, wavPath); err != nil {
		s.logger.Printf("ERROR: piper synth failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.cache.EnforceLimit(); err != nil {
		s.logger.Printf("ERROR: enforce cache limit failed: %v", err)
	}

	go s.player.PlayWav(wavPath)
	s.logger.Printf("INFO: /tts cache_miss key=%s file=%s", key, filepath.Base(wavPath))
	s.writeJSON(w, http.StatusOK, ttsResponse{Status: "cache_miss", File: filename})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, err := os.Stat(s.cfg.CacheDir); err != nil {
		s.logger.Printf("ERROR: cache dir check failed: %v", err)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK\n"))
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
