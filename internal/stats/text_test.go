package stats

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestFmtDuration(t *testing.T) {
	tests := []struct {
		name     string
		d        time.Duration
		expected string
	}{
		{
			name:     "zero",
			d:        0,
			expected: "0ms",
		},
		{
			name:     "milliseconds",
			d:        423 * time.Millisecond,
			expected: "423ms",
		},
		{
			name:     "just under one second",
			d:        999 * time.Millisecond,
			expected: "999ms",
		},
		{
			name:     "exactly one second",
			d:        time.Second,
			expected: "1.0s",
		},
		{
			name:     "seconds with decimal",
			d:        2800 * time.Millisecond,
			expected: "2.8s",
		},
		{
			name:     "seconds 1.1s",
			d:        1100 * time.Millisecond,
			expected: "1.1s",
		},
		{
			name:     "just under one minute",
			d:        59*time.Second + 900*time.Millisecond,
			expected: "59.9s",
		},
		{
			name:     "exactly one minute",
			d:        time.Minute,
			expected: "1m0s",
		},
		{
			name:     "one hour",
			d:        time.Hour,
			expected: "1h0m0s",
		},
		{
			name:     "one hour zero minutes",
			d:        60 * time.Minute,
			expected: "1h0m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmtDuration(tt.d)
			if got != tt.expected {
				t.Errorf("fmtDuration(%v) = %q, want %q", tt.d, got, tt.expected)
			}
		})
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{
			name:     "full URL with path",
			rawURL:   "https://api.openai.com/v1",
			expected: "api.openai.com",
		},
		{
			name:     "URL without path",
			rawURL:   "https://api.anthropic.com",
			expected: "api.anthropic.com",
		},
		{
			name:     "URL with port",
			rawURL:   "http://localhost:8080/v1",
			expected: "localhost",
		},
		{
			name:     "bare hostname fallback",
			rawURL:   "api.openai.com",
			expected: "api.openai.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHost(tt.rawURL)
			if got != tt.expected {
				t.Errorf("extractHost(%q) = %q, want %q", tt.rawURL, got, tt.expected)
			}
		})
	}
}

func TestWriteTextSessionHeader(t *testing.T) {
	f, err := os.CreateTemp("", "stats-text-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	ts := time.Date(2026, 3, 14, 15, 4, 5, 0, time.UTC)
	profileName := "my-profile"
	pid := 12345

	writeTextSessionHeader(f, sessionID, ts, profileName, pid)

	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 4096)
	n, err := f.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	got := string(buf[:n])

	wantContains := []string{
		"══════════════════════════════════════════════════════",
		"gocc session 550e8400-e29b-41d4-a716-446655440000",
		"2026-03-14 15:04:05",
		"profile=my-profile",
		"pid=12345",
	}

	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("writeTextSessionHeader output missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestWriteTextSessionFooter(t *testing.T) {
	f, err := os.CreateTemp("", "stats-text-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	// 12000 output tokens over 120s active time = 100.0 tok/s
	writeTextSessionFooter(f, time.Hour, 42, 3, 1, 50000, 12000, 120*time.Second)

	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	got := string(buf[:n])

	wantContains := []string{
		"══════════════════════════════════════════════════════",
		"session end",
		"dur=1h0m0s",
		"requests=42",
		"errors=3",
		"canceled=1",
		"in=50000",
		"out=12000",
		"avg_tok/s=100.0",
	}

	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("writeTextSessionFooter output missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestWriteTextRequestStart(t *testing.T) {
	f, err := os.CreateTemp("", "stats-text-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	ts := time.Date(2026, 3, 14, 15, 4, 5, 0, time.UTC)
	rs := &requestState{
		modelLevel:   "sonnet",
		requestModel: "gpt-4o",
		streaming:    true,
		messageCount: 5,
		toolCount:    3,
	}

	writeTextRequestStart(f, "7f3a", ts, "my-profile", rs)

	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	got := string(buf[:n])

	wantContains := []string{
		"req=7f3a",
		"15:04:05 START",
		"[my-profile]",
		"sonnet",
		"model=gpt-4o",
		"stream=true",
		"msgs=5",
		"tools=3",
	}

	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("writeTextRequestStart output missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestWriteTextFirstByte(t *testing.T) {
	f, err := os.CreateTemp("", "stats-text-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	ts := time.Date(2026, 3, 14, 15, 4, 5, 0, time.UTC)

	t.Run("no fallback", func(t *testing.T) {
		f.Truncate(0)
		f.Seek(0, 0)
		writeTextFirstByte(f, ts, 423*time.Millisecond, 0)
		f.Seek(0, 0)
		buf := make([]byte, 4096)
		n, _ := f.Read(buf)
		got := string(buf[:n])
		if !strings.Contains(got, "FIRST_BYTE ttfb=423ms") {
			t.Errorf("missing FIRST_BYTE line, got: %s", got)
		}
		if strings.Contains(got, "attempt") {
			t.Errorf("should not contain attempt when attemptNum=0, got: %s", got)
		}
	})

	t.Run("with fallback", func(t *testing.T) {
		f.Truncate(0)
		f.Seek(0, 0)
		writeTextFirstByte(f, ts, 1100*time.Millisecond, 2)
		f.Seek(0, 0)
		buf := make([]byte, 4096)
		n, _ := f.Read(buf)
		got := string(buf[:n])
		if !strings.Contains(got, "FIRST_BYTE ttfb=1.1s") {
			t.Errorf("missing FIRST_BYTE ttfb, got: %s", got)
		}
		if !strings.Contains(got, "(attempt#3)") {
			t.Errorf("missing attempt suffix, got: %s", got)
		}
	})
}

func TestWriteTextAttemptError(t *testing.T) {
	f, err := os.CreateTemp("", "stats-text-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	ts := time.Date(2026, 3, 14, 15, 5, 1, 0, time.UTC)
	writeTextAttemptError(f, ts, 1, "openai_chat", "https://api.openai.com/v1", "gpt-4o", 429, 200*time.Millisecond, "rate limit exceeded")

	f.Seek(0, 0)
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	got := string(buf[:n])

	wantContains := []string{
		"FAIL#1",
		"openai_chat",
		"api.openai.com",
		"model=gpt-4o",
		"status=429",
		"dur=200ms",
		`"rate limit exceeded"`,
	}

	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("writeTextAttemptError output missing %q\ngot:\n%s", want, got)
		}
	}
	// Should NOT contain the full URL
	if strings.Contains(got, "https://") {
		t.Errorf("should not contain full URL, got: %s", got)
	}
}

func TestWriteTextComplete(t *testing.T) {
	f, err := os.CreateTemp("", "stats-text-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	ts := time.Date(2026, 3, 14, 15, 4, 8, 0, time.UTC)

	t.Run("OK no fallback", func(t *testing.T) {
		f.Truncate(0)
		f.Seek(0, 0)
		writeTextComplete(f, ts, "OK", 2800*time.Millisecond, 1024, 256, 512, 106, 91.4, "gpt-4o-2024-08-06", 1, "")
		f.Seek(0, 0)
		buf := make([]byte, 4096)
		n, _ := f.Read(buf)
		got := string(buf[:n])

		wantContains := []string{
			"FINISH OK",
			"dur=2.8s",
			"in=1024",
			"out=256",
			"cache_r=512",
			"think=106",
			"tok/s=91.4",
			"model=gpt-4o-2024-08-06",
		}
		for _, want := range wantContains {
			if !strings.Contains(got, want) {
				t.Errorf("OK complete missing %q\ngot:\n%s", want, got)
			}
		}
		// attempt=1 should not appear (only shown when > 1)
		if strings.Contains(got, "attempt=") {
			t.Errorf("should not show attempt=1, got: %s", got)
		}
	})

	t.Run("OK with fallback", func(t *testing.T) {
		f.Truncate(0)
		f.Seek(0, 0)
		writeTextComplete(f, ts, "OK", 4800*time.Millisecond, 1024, 256, 0, 0, 53.2, "gemini-2.0-flash", 3, "")
		f.Seek(0, 0)
		buf := make([]byte, 4096)
		n, _ := f.Read(buf)
		got := string(buf[:n])

		if !strings.Contains(got, "attempt=3") {
			t.Errorf("missing attempt=3, got: %s", got)
		}
		// cache_r=0 should be omitted
		if strings.Contains(got, "cache_r=") {
			t.Errorf("cache_r=0 should be omitted, got: %s", got)
		}
	})

	t.Run("ERROR", func(t *testing.T) {
		f.Truncate(0)
		f.Seek(0, 0)
		writeTextComplete(f, ts, "ERROR", 800*time.Millisecond, 0, 0, 0, 0, 0, "", 1, "internal server error")
		f.Seek(0, 0)
		buf := make([]byte, 4096)
		n, _ := f.Read(buf)
		got := string(buf[:n])

		if !strings.Contains(got, "FINISH ERROR") {
			t.Errorf("missing FINISH ERROR, got: %s", got)
		}
		if !strings.Contains(got, `"internal server error"`) {
			t.Errorf("missing error message, got: %s", got)
		}
	})

	t.Run("CANCELED", func(t *testing.T) {
		f.Truncate(0)
		f.Seek(0, 0)
		writeTextComplete(f, ts, "CANCELED", 1600*time.Millisecond, 512, 30, 0, 0, 0, "", 1, "context canceled")
		f.Seek(0, 0)
		buf := make([]byte, 4096)
		n, _ := f.Read(buf)
		got := string(buf[:n])

		if !strings.Contains(got, "FINISH CANCELED") {
			t.Errorf("missing FINISH CANCELED, got: %s", got)
		}
		if !strings.Contains(got, `"context canceled"`) {
			t.Errorf("missing cancel message, got: %s", got)
		}
	})
}
