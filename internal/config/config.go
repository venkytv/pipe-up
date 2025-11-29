package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds environment-driven settings for the service.
type Config struct {
	PiperExec     string
	PiperModel    string
	PiperFlags    []string
	CacheDir      string
	ListenAddr    string
	PlayCmd       string
	PlayArgs      []string
	VoiceID       string
	CacheMaxBytes int64
}

const (
	defaultPiperExec     = "/usr/local/bin/piper"
	defaultCacheDir      = "/var/cache/tts-cached"
	defaultListenAddr    = "127.0.0.1:4410" // 44.1 kHz-inspired port
	defaultPlayCmd       = "/usr/bin/aplay"
	defaultVoiceID       = "default"
	defaultCacheMaxBytes = int64(536870912) // 512 MiB
)

// Load reads configuration from environment variables and ensures the cache directory exists.
func Load() (Config, error) {
	return LoadWithOverrides(Config{})
}

// LoadWithOverrides merges environment variables with explicit overrides and ensures the cache directory exists.
func LoadWithOverrides(override Config) (Config, error) {
	cfg := Config{
		PiperExec:     getEnv("PIPER_EXEC", defaultPiperExec),
		PiperModel:    strings.TrimSpace(os.Getenv("PIPER_MODEL")),
		CacheDir:      getEnv("CACHE_DIR", defaultCacheDir),
		ListenAddr:    getEnv("LISTEN_ADDR", defaultListenAddr),
		PlayCmd:       getEnv("PLAY_CMD", defaultPlayCmd),
		VoiceID:       getEnv("VOICE_ID", defaultVoiceID),
		CacheMaxBytes: defaultCacheMaxBytes,
	}

	if args := strings.TrimSpace(os.Getenv("PIPER_FLAGS")); args != "" {
		cfg.PiperFlags = strings.Fields(args)
	}

	if args := strings.TrimSpace(os.Getenv("PLAY_ARGS")); args != "" {
		cfg.PlayArgs = strings.Fields(args)
	} else {
		cfg.PlayArgs = []string{}
	}

	if maxBytesStr := strings.TrimSpace(os.Getenv("CACHE_MAX_BYTES")); maxBytesStr != "" {
		val, err := strconv.ParseInt(maxBytesStr, 10, 64)
		if err != nil || val <= 0 {
			return Config{}, errors.New("invalid CACHE_MAX_BYTES; must be positive integer")
		}
		cfg.CacheMaxBytes = val
	}

	// Apply overrides.
	if override.PiperExec != "" {
		cfg.PiperExec = override.PiperExec
	}
	if override.PiperModel != "" {
		cfg.PiperModel = override.PiperModel
	}
	if override.PiperFlags != nil {
		cfg.PiperFlags = override.PiperFlags
	}
	if override.CacheDir != "" {
		cfg.CacheDir = override.CacheDir
	}
	if override.ListenAddr != "" {
		cfg.ListenAddr = override.ListenAddr
	}
	if override.PlayCmd != "" {
		cfg.PlayCmd = override.PlayCmd
	}
	if override.PlayArgs != nil {
		cfg.PlayArgs = override.PlayArgs
	}
	if override.VoiceID != "" {
		cfg.VoiceID = override.VoiceID
	}
	if override.CacheMaxBytes > 0 {
		cfg.CacheMaxBytes = override.CacheMaxBytes
	}

	if cfg.PiperModel == "" {
		return Config{}, errors.New("PIPER_MODEL is required (flag or env)")
	}

	if err := os.MkdirAll(cfg.CacheDir, 0o755); err != nil {
		return Config{}, err
	}

	// Resolve to an absolute path for clarity in logs.
	if abs, err := filepath.Abs(cfg.CacheDir); err == nil {
		cfg.CacheDir = abs
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); strings.TrimSpace(v) != "" {
		return v
	}
	return def
}

// DefaultPiperExec exposes the default piper executable path.
func DefaultPiperExec() string { return defaultPiperExec }

// DefaultCacheDir exposes the default cache directory.
func DefaultCacheDir() string { return defaultCacheDir }

// DefaultListenAddr exposes the default listen address.
func DefaultListenAddr() string { return defaultListenAddr }

// DefaultPlayCmd exposes the default playback command.
func DefaultPlayCmd() string { return defaultPlayCmd }
