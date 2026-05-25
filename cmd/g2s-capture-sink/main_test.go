package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCaptureSinkRecordsRawBody(t *testing.T) {
	records := []CaptureRecord{}
	sink := &captureSink{
		path: "/capture",
		recordFn: func(record CaptureRecord) error {
			records = append(records, record)
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/capture", strings.NewReader("<g2s>payload</g2s>"))
	req.Header.Set("Content-Type", "application/xml")
	res := httptest.NewRecorder()
	sink.ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("status=%d want %d", res.Code, http.StatusAccepted)
	}
	if len(records) != 1 {
		t.Fatalf("records=%d want 1", len(records))
	}
	if records[0].RawBody != "<g2s>payload</g2s>" {
		t.Fatalf("raw body=%q", records[0].RawBody)
	}
}

func TestCaptureSinkPathNotFound(t *testing.T) {
	sink := &captureSink{
		path: "/capture",
		recordFn: func(CaptureRecord) error {
			t.Fatal("record should not be called")
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/wrong", nil)
	res := httptest.NewRecorder()
	sink.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status=%d want %d", res.Code, http.StatusNotFound)
	}
}

func TestNormalizeCapturePath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: "/capture"},
		{in: "capture", want: "/capture"},
		{in: "/capture", want: "/capture"},
	}
	for _, tc := range cases {
		if got := normalizeCapturePath(tc.in); got != tc.want {
			t.Fatalf("normalizeCapturePath(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
