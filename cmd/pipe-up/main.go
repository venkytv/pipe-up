package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/venkytv/tts-cached/internal/cli"
)

type ttsResponse struct {
	Status string `json:"status"`
	File   string `json:"file"`
}

func main() {
	defaultServer := env("TTS_CACHED_URL", "http://127.0.0.1:4410/tts")

	var filePath string
	flag.StringVar(&filePath, "file", "", "text file to read ('-' for stdin)")
	flag.StringVar(&filePath, "f", "", "text file to read ('-' for stdin)")
	serverURL := flag.String("server", defaultServer, "tts-cached /tts endpoint URL")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] <text>\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Submit text to tts-cached.\n\n")
		fmt.Fprintln(flag.CommandLine.Output(), "Input modes:")
		fmt.Fprintln(flag.CommandLine.Output(), "  -f <path>   read text from file")
		fmt.Fprintln(flag.CommandLine.Output(), "  -f -        read text from stdin")
		fmt.Fprintln(flag.CommandLine.Output(), "  <text>      provide text as args when no -f is set")
		fmt.Fprintln(flag.CommandLine.Output())
		flag.PrintDefaults()
	}

	flag.Parse()

	text, err := cli.ResolveText(filePath, flag.Args(), os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	if err := submit(*serverURL, text, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "submit failed: %v\n", err)
		os.Exit(1)
	}
}

func submit(serverURL, text string, out io.Writer) error {
	payload := struct {
		Text string `json:"text"`
	}{Text: text}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, serverURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post to server: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed ttsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	fmt.Fprintf(out, "status=%s file=%s\n", parsed.Status, parsed.File)
	return nil
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
