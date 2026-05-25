package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CaptureRecord struct {
	Timestamp   time.Time           `json:"timestamp"`
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	RemoteAddr  string              `json:"remote_addr"`
	ContentType string              `json:"content_type,omitempty"`
	Headers     map[string][]string `json:"headers,omitempty"`
	RawBody     string              `json:"raw_body"`
}

type captureSink struct {
	path      string
	recordFn  func(CaptureRecord) error
	logOutput io.Writer
}

func (s *captureSink) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != s.path {
		http.NotFound(w, r)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	record := CaptureRecord{
		Timestamp:   time.Now().UTC(),
		Method:      r.Method,
		Path:        r.URL.Path,
		RemoteAddr:  r.RemoteAddr,
		ContentType: strings.TrimSpace(r.Header.Get("Content-Type")),
		Headers:     map[string][]string(r.Header),
		RawBody:     string(body),
	}
	if s.recordFn != nil {
		if err := s.recordFn(record); err != nil {
			http.Error(w, "record failed", http.StatusInternalServerError)
			return
		}
	}
	if s.logOutput != nil {
		fmt.Fprintf(
			s.logOutput,
			"captured at=%s method=%s path=%s remote=%s content_type=%s body=%s\n",
			record.Timestamp.Format(time.RFC3339Nano),
			record.Method,
			record.Path,
			record.RemoteAddr,
			record.ContentType,
			record.RawBody,
		)
	}
	w.WriteHeader(http.StatusAccepted)
}

type jsonlRecorder struct {
	mu     sync.Mutex
	file   *os.File
	encode *json.Encoder
}

func newJSONLRecorder(path string) (*jsonlRecorder, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create output directory: %w", err)
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open output file: %w", err)
	}
	return &jsonlRecorder{
		file:   file,
		encode: json.NewEncoder(file),
	}, nil
}

func (r *jsonlRecorder) Record(record CaptureRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.encode.Encode(record)
}

func (r *jsonlRecorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return nil
	}
	return r.file.Close()
}

func main() {
	bind := flag.String("bind", "127.0.0.1:18080", "bind address")
	path := flag.String("path", "/capture", "capture path")
	out := flag.String("out", "./data/g2s-capture-sink.jsonl", "JSONL output path")
	flag.Parse()

	recorder, err := newJSONLRecorder(strings.TrimSpace(*out))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create recorder: %v\n", err)
		os.Exit(1)
	}
	defer recorder.Close()

	sink := &captureSink{
		path:      normalizeCapturePath(*path),
		recordFn:  recorder.Record,
		logOutput: os.Stdout,
	}

	fmt.Printf("capture sink listening bind=%s path=%s out=%s\n", strings.TrimSpace(*bind), sink.path, strings.TrimSpace(*out))
	if err := http.ListenAndServe(strings.TrimSpace(*bind), sink); err != nil {
		fmt.Fprintf(os.Stderr, "capture sink failed: %v\n", err)
		os.Exit(1)
	}
}

func normalizeCapturePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/capture"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}
