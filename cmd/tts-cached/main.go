package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/venkytv/tts-cached/internal/audio"
	"github.com/venkytv/tts-cached/internal/cache"
	"github.com/venkytv/tts-cached/internal/config"
	"github.com/venkytv/tts-cached/internal/piperexec"
	"github.com/venkytv/tts-cached/internal/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	env := func(key, def string) string {
		if v := os.Getenv(key); strings.TrimSpace(v) != "" {
			return v
		}
		return def
	}

	piperExec := flag.String("piper-exec", env("PIPER_EXEC", config.DefaultPiperExec()), "path to piper executable (env PIPER_EXEC)")
	piperModel := flag.String("piper-model", os.Getenv("PIPER_MODEL"), "path to piper model file (env PIPER_MODEL, required)")
	piperFlags := flag.String("piper-flags", os.Getenv("PIPER_FLAGS"), "additional piper flags (space-separated, env PIPER_FLAGS)")
	cacheDir := flag.String("cache-dir", env("CACHE_DIR", config.DefaultCacheDir()), "cache directory for wav files (env CACHE_DIR)")
	listenAddr := flag.String("listen-addr", env("LISTEN_ADDR", config.DefaultListenAddr()), "HTTP listen address (env LISTEN_ADDR)")
	playCmd := flag.String("play-cmd", env("PLAY_CMD", config.DefaultPlayCmd()), "playback command (env PLAY_CMD)")
	playArgs := flag.String("play-args", os.Getenv("PLAY_ARGS"), "playback extra args (space-separated, env PLAY_ARGS)")
	voiceID := flag.String("voice-id", env("VOICE_ID", "default"), "voice identifier used in cache key (env VOICE_ID)")
	cacheMaxBytes := flag.String("cache-max-bytes", os.Getenv("CACHE_MAX_BYTES"), "max cache size in bytes (env CACHE_MAX_BYTES, default 536870912)")

	flag.Parse()

	override := config.Config{
		PiperExec:  strings.TrimSpace(*piperExec),
		PiperModel: strings.TrimSpace(*piperModel),
		CacheDir:   strings.TrimSpace(*cacheDir),
		ListenAddr: strings.TrimSpace(*listenAddr),
		PlayCmd:    strings.TrimSpace(*playCmd),
		VoiceID:    strings.TrimSpace(*voiceID),
	}

	if strings.TrimSpace(*piperFlags) != "" {
		override.PiperFlags = strings.Fields(*piperFlags)
	}
	if strings.TrimSpace(*playArgs) != "" {
		override.PlayArgs = strings.Fields(*playArgs)
	}
	if strings.TrimSpace(*cacheMaxBytes) != "" {
		val, err := strconv.ParseInt(strings.TrimSpace(*cacheMaxBytes), 10, 64)
		if err != nil || val <= 0 {
			log.Fatalf("invalid cache-max-bytes: %v", err)
		}
		override.CacheMaxBytes = val
	}

	cfg, err := config.LoadWithOverrides(override)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("INFO: starting tts-cached with config: PIPER_EXEC=%s PIPER_MODEL=%s PIPER_FLAGS=%v CACHE_DIR=%s LISTEN_ADDR=%s PLAY_CMD=%s VOICE_ID=%s CACHE_MAX_BYTES=%d",
		cfg.PiperExec, cfg.PiperModel, cfg.PiperFlags, cfg.CacheDir, cfg.ListenAddr, cfg.PlayCmd, cfg.VoiceID, cfg.CacheMaxBytes)

	cacheMgr := cache.NewManager(cfg.CacheDir, cfg.CacheMaxBytes, log.Default())
	player := audio.NewPlayer(cfg.PlayCmd, cfg.PlayArgs, log.Default())
	piper := piperexec.New(cfg.PiperExec, cfg.PiperModel, cfg.PiperFlags, log.Default())
	srv := server.New(cfg, cacheMgr, piper, player, log.Default())

	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: srv.Handler(),
	}

	go func() {
		log.Printf("INFO: HTTP server listening on %s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("INFO: received signal %s, shutting down", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("ERROR: graceful shutdown failed: %v", err)
	}
	log.Printf("INFO: shutdown complete")
}
