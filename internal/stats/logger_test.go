package stats

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/llmapimux/llmapimux"
)

// helper to create a temp dir and clean it up after test
func makeTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gocc-stats-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// readLines reads all non-empty lines from a file.
func readLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}

// parseJSONLRecords reads all JSONL lines and parses them as map[string]any for inspection.
func parseJSONLRecords(t *testing.T, path string) []map[string]any {
	t.Helper()
	lines := readLines(t, path)
	var records []map[string]any
	for _, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("parse JSONL line %q: %v", line, err)
		}
		records = append(records, m)
	}
	return records
}

func TestNewSessionLogger_DebugDumpWritesRequestBody(t *testing.T) {
	dir := makeTempDir(t)
	l, err := NewSessionLogger(dir, "pid", "profile", "v0.1.0", true)
	if err != nil {
		t.Fatalf("NewSessionLogger(debug): %v", err)
	}
	ctx := context.Background()
	temp := 0.7
	l.OnRequestStart(ctx, llmapimux.RequestStartEvent{
		RequestID:        "req-dbg-1",
		Time:             time.Now(),
		InboundProtocol:  llmapimux.ProtocolAnthropic,
		OutboundProtocol: llmapimux.ProtocolOpenAIResponses,
		Streaming:        true,
		IRRequest: &llmapimux.Request{
			Model:       "gpt-5",
			Temperature: &temp,
			MaxTokens:   128,
			Messages: []llmapimux.Message{
				{Role: llmapimux.RoleUser, Content: []llmapimux.ContentPart{
					{Type: llmapimux.ContentTypeText, Text: &llmapimux.TextContent{Text: "hi"}},
				}},
			},
		},
	})
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	var mainJSONL, debugJSONL string
	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasSuffix(name, "-debug.jsonl"):
			debugJSONL = dir + "/" + name
		case strings.HasSuffix(name, ".jsonl"):
			mainJSONL = dir + "/" + name
		}
	}
	if mainJSONL == "" || debugJSONL == "" {
		t.Fatalf("expected both main and debug jsonl files, got main=%q debug=%q", mainJSONL, debugJSONL)
	}

	// Main jsonl must not contain request_body records.
	for _, rec := range parseJSONLRecords(t, mainJSONL) {
		if rec["type"] == "request_body" {
			t.Errorf("request_body leaked into primary jsonl: %v", rec)
		}
	}

	debugRecs := parseJSONLRecords(t, debugJSONL)
	if len(debugRecs) != 1 {
		t.Fatalf("expected 1 debug record, got %d: %v", len(debugRecs), debugRecs)
	}
	rec := debugRecs[0]
	if rec["type"] != "request_body" {
		t.Errorf("debug record type = %v, want request_body", rec["type"])
	}
	if rec["request_id"] != "req-dbg-1" {
		t.Errorf("request_id = %v, want req-dbg-1", rec["request_id"])
	}
	if rec["outbound_protocol"] != "openai_responses" {
		t.Errorf("outbound_protocol = %v, want openai_responses", rec["outbound_protocol"])
	}
	if _, ok := rec["ir_request"]; !ok {
		t.Error("ir_request missing")
	}
	outBody, ok := rec["outbound_body"].(map[string]any)
	if !ok {
		t.Fatalf("outbound_body missing or not a JSON object: %v", rec["outbound_body"])
	}
	if outBody["model"] != "gpt-5" {
		t.Errorf("outbound_body.model = %v, want gpt-5", outBody["model"])
	}
}

func TestNewSessionLogger_DebugDisabledSkipsDebugFile(t *testing.T) {
	dir := makeTempDir(t)
	l, err := NewSessionLogger(dir, "pid", "profile", "v0.1.0", false)
	if err != nil {
		t.Fatalf("NewSessionLogger: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "-debug.jsonl") {
			t.Errorf("unexpected debug sidecar when debug=false: %s", e.Name())
		}
	}
}

func TestNewSessionLogger_CreatesFiles(t *testing.T) {
	dir := makeTempDir(t)
	l, err := NewSessionLogger(dir, "prof-id-1", "my-profile", "v0.1.0", false)
	if err != nil {
		t.Fatalf("NewSessionLogger: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	var jsonlFound, logFound bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			jsonlFound = true
		}
		if strings.HasSuffix(e.Name(), ".log") {
			logFound = true
		}
	}
	if !jsonlFound {
		t.Error("expected .jsonl file to be created")
	}
	if !logFound {
		t.Error("expected .log file to be created")
	}
}

func TestNewSessionLogger_SessionStartInJSONL(t *testing.T) {
	dir := makeTempDir(t)
	l, err := NewSessionLogger(dir, "pid-abc", "test-profile", "v0.2.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(dir)
	var jsonlPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			jsonlPath = dir + "/" + e.Name()
		}
	}

	records := parseJSONLRecords(t, jsonlPath)
	// At least session_start and session_end.
	if len(records) < 2 {
		t.Fatalf("expected at least 2 JSONL records, got %d", len(records))
	}
	if records[0]["type"] != "session_start" {
		t.Errorf("first record type = %q, want session_start", records[0]["type"])
	}
	if records[0]["profile_id"] != "pid-abc" {
		t.Errorf("profile_id = %q, want pid-abc", records[0]["profile_id"])
	}
	if records[0]["profile_name"] != "test-profile" {
		t.Errorf("profile_name = %q, want test-profile", records[0]["profile_name"])
	}
	if records[0]["gocc_version"] != "v0.2.0" {
		t.Errorf("gocc_version = %q, want v0.2.0", records[0]["gocc_version"])
	}
	if records[len(records)-1]["type"] != "session_end" {
		t.Errorf("last record type = %q, want session_end", records[len(records)-1]["type"])
	}
}

func TestNewSessionLogger_TextHeader(t *testing.T) {
	dir := makeTempDir(t)
	l, err := NewSessionLogger(dir, "pid-abc", "my-profile", "v0.1.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(dir)
	var logPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			logPath = dir + "/" + e.Name()
		}
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)

	for _, want := range []string{
		"gocc session",
		"profile=my-profile",
		"session end",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("text log missing %q", want)
		}
	}
}

// buildTestRequest creates a minimal llmapimux.Request for testing.
func buildTestRequest(model string, msgs int, tools int, systemPrompt string) *llmapimux.Request {
	req := &llmapimux.Request{
		Model: model,
	}
	for i := 0; i < msgs; i++ {
		req.Messages = append(req.Messages, llmapimux.Message{
			Role: "user",
			Content: []llmapimux.ContentPart{
				{Type: "text", Text: &llmapimux.TextContent{Text: fmt.Sprintf("msg%d", i)}},
			},
		})
	}
	for i := 0; i < tools; i++ {
		req.Tools = append(req.Tools, llmapimux.Tool{Name: fmt.Sprintf("tool%d", i)})
	}
	if systemPrompt != "" {
		req.SystemPrompt = []llmapimux.ContentPart{
			{Type: "text", Text: &llmapimux.TextContent{Text: systemPrompt}},
		}
	}
	return req
}

func TestFullLifecycle_Success(t *testing.T) {
	dir := makeTempDir(t)
	l, err := NewSessionLogger(dir, "prof-id", "my-profile", "v0.1.0", false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reqID := "7f3a1b2c-abcd-4567-8901-abcdef012345"
	now := time.Now()

	// OnRequestStart
	l.OnRequestStart(ctx, llmapimux.RequestStartEvent{
		RequestID:        reqID,
		Time:             now,
		InboundProtocol:  llmapimux.ProtocolAnthropic,
		OutboundProtocol: llmapimux.ProtocolOpenAIChat,
		Streaming:        true,
		IRRequest:        buildTestRequest("s-gpt-4o", 3, 2, "be helpful"),
	})

	// OnFirstByte
	firstByteTime := now.Add(500 * time.Millisecond)
	l.OnFirstByte(ctx, llmapimux.FirstByteEvent{
		RequestID: reqID,
		Time:      firstByteTime,
		TTFB:      500 * time.Millisecond,
	})

	// OnStreamChunk
	l.OnStreamChunk(ctx, llmapimux.StreamChunkEvent{
		RequestID:       reqID,
		Time:            now.Add(600 * time.Millisecond),
		SequenceNum:     1,
		ElapsedTime:     600 * time.Millisecond,
		InterChunkDelay: 100 * time.Millisecond,
		IREvent: &llmapimux.StreamEvent{
			Type: llmapimux.StreamEventDelta,
		},
	})

	// OnComplete
	l.OnComplete(ctx, llmapimux.CompleteEvent{
		RequestID:        reqID,
		Time:             now.Add(2 * time.Second),
		Status:           llmapimux.CompletionStatusSuccess,
		TTFB:             500 * time.Millisecond,
		TotalLatency:     2 * time.Second,
		Usage:            llmapimux.Usage{InputTokens: 1024, OutputTokens: 256, CacheReadTokens: 512, ThinkingTokens: 106},
		OutputThroughput: 91.4,
		StopReason:       "end_turn",
		ActualModel:      "gpt-4o-2024-08-06",
		AttemptNum:       1,
	})

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	// Find files.
	entries, _ := os.ReadDir(dir)
	var jsonlPath, logPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			jsonlPath = dir + "/" + e.Name()
		}
		if strings.HasSuffix(e.Name(), ".log") {
			logPath = dir + "/" + e.Name()
		}
	}

	// Verify JSONL.
	records := parseJSONLRecords(t, jsonlPath)
	types := make([]string, 0, len(records))
	for _, r := range records {
		types = append(types, r["type"].(string))
	}
	// Expected order: session_start, request_start, first_byte, stream_chunk, complete, session_end
	want := []string{"session_start", "request_start", "first_byte", "stream_chunk", "complete", "session_end"}
	if len(types) != len(want) {
		t.Fatalf("JSONL record types = %v, want %v", types, want)
	}
	for i, w := range want {
		if types[i] != w {
			t.Errorf("record[%d] type = %q, want %q", i, types[i], w)
		}
	}

	// Verify request_start record fields.
	rsRec := records[1]
	if rsRec["inbound_protocol"] != "anthropic" {
		t.Errorf("inbound_protocol = %v, want anthropic", rsRec["inbound_protocol"])
	}
	if rsRec["model_level"] != "sonnet" {
		t.Errorf("model_level = %v, want sonnet", rsRec["model_level"])
	}
	if rsRec["request_model"] != "gpt-4o" {
		t.Errorf("request_model = %v, want gpt-4o", rsRec["request_model"])
	}
	if rsRec["message_count"] != float64(3) {
		t.Errorf("message_count = %v, want 3", rsRec["message_count"])
	}
	if rsRec["has_tools"] != true {
		t.Errorf("has_tools = %v, want true", rsRec["has_tools"])
	}
	if rsRec["has_system"] != true {
		t.Errorf("has_system = %v, want true", rsRec["has_system"])
	}
	if rsRec["streaming"] != true {
		t.Errorf("streaming = %v, want true", rsRec["streaming"])
	}

	// Verify complete record.
	compRec := records[4]
	if compRec["status"] != "success" {
		t.Errorf("complete status = %v, want success", compRec["status"])
	}
	if compRec["actual_model"] != "gpt-4o-2024-08-06" {
		t.Errorf("actual_model = %v, want gpt-4o-2024-08-06", compRec["actual_model"])
	}

	// Verify session_end counters.
	seRec := records[5]
	if seRec["total_requests"] != float64(1) {
		t.Errorf("total_requests = %v, want 1", seRec["total_requests"])
	}
	if seRec["total_errors"] != float64(0) {
		t.Errorf("total_errors = %v, want 0", seRec["total_errors"])
	}
	if seRec["total_input_tokens"] != float64(1024) {
		t.Errorf("total_input_tokens = %v, want 1024", seRec["total_input_tokens"])
	}
	if seRec["total_output_tokens"] != float64(256) {
		t.Errorf("total_output_tokens = %v, want 256", seRec["total_output_tokens"])
	}

	// Verify text log.
	textData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(textData)
	for _, want := range []string{
		"START",
		"FIRST_BYTE",
		"FINISH OK",
		"session end",
		"requests=1",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("text log missing %q\nfull text:\n%s", want, text)
		}
	}
	// stream_chunk should NOT appear in text log.
	if strings.Contains(text, "stream_chunk") || strings.Contains(text, "CHUNK") {
		t.Errorf("text log should not contain chunk entries")
	}
}

func TestFullLifecycle_Error(t *testing.T) {
	dir := makeTempDir(t)
	l, err := NewSessionLogger(dir, "prof-id", "my-profile", "v0.1.0", false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reqID := "error-req-id-1234"
	now := time.Now()

	l.OnRequestStart(ctx, llmapimux.RequestStartEvent{
		RequestID:        reqID,
		Time:             now,
		InboundProtocol:  llmapimux.ProtocolAnthropic,
		OutboundProtocol: llmapimux.ProtocolOpenAIChat,
		Streaming:        false,
		IRRequest:        buildTestRequest("Haiku(gpt-3.5-turbo)", 1, 0, ""),
	})

	l.OnComplete(ctx, llmapimux.CompleteEvent{
		RequestID:    reqID,
		Time:         now.Add(800 * time.Millisecond),
		Status:       llmapimux.CompletionStatusError,
		Error:        errors.New("internal server error"),
		TotalLatency: 800 * time.Millisecond,
		AttemptNum:   1,
	})

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(dir)
	var jsonlPath, logPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			jsonlPath = dir + "/" + e.Name()
		}
		if strings.HasSuffix(e.Name(), ".log") {
			logPath = dir + "/" + e.Name()
		}
	}

	records := parseJSONLRecords(t, jsonlPath)
	seRec := records[len(records)-1]
	if seRec["total_errors"] != float64(1) {
		t.Errorf("total_errors = %v, want 1", seRec["total_errors"])
	}
	if seRec["total_requests"] != float64(1) {
		t.Errorf("total_requests = %v, want 1", seRec["total_requests"])
	}

	textData, _ := os.ReadFile(logPath)
	text := string(textData)
	if !strings.Contains(text, "FINISH ERROR") {
		t.Errorf("text log missing FINISH ERROR\n%s", text)
	}
	if !strings.Contains(text, `"internal server error"`) {
		t.Errorf("text log missing error message\n%s", text)
	}
	if !strings.Contains(text, "errors=1") {
		t.Errorf("text log missing errors=1\n%s", text)
	}
}

func TestFullLifecycle_Canceled(t *testing.T) {
	dir := makeTempDir(t)
	l, err := NewSessionLogger(dir, "prof-id", "my-profile", "v0.1.0", false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reqID := "cancel-req-id-5678"
	now := time.Now()

	l.OnRequestStart(ctx, llmapimux.RequestStartEvent{
		RequestID:        reqID,
		Time:             now,
		InboundProtocol:  llmapimux.ProtocolAnthropic,
		OutboundProtocol: llmapimux.ProtocolGemini,
		Streaming:        true,
		IRRequest:        buildTestRequest("Opus(gemini-2.0-pro)", 2, 0, ""),
	})

	l.OnFirstByte(ctx, llmapimux.FirstByteEvent{
		RequestID: reqID,
		Time:      now.Add(300 * time.Millisecond),
		TTFB:      300 * time.Millisecond,
	})

	l.OnComplete(ctx, llmapimux.CompleteEvent{
		RequestID:    reqID,
		Time:         now.Add(1600 * time.Millisecond),
		Status:       llmapimux.CompletionStatusCanceled,
		Error:        errors.New("context canceled"),
		TotalLatency: 1600 * time.Millisecond,
		Usage:        llmapimux.Usage{InputTokens: 512, OutputTokens: 30},
		AttemptNum:   1,
	})

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(dir)
	var jsonlPath, logPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			jsonlPath = dir + "/" + e.Name()
		}
		if strings.HasSuffix(e.Name(), ".log") {
			logPath = dir + "/" + e.Name()
		}
	}

	records := parseJSONLRecords(t, jsonlPath)
	seRec := records[len(records)-1]
	if seRec["total_canceled"] != float64(1) {
		t.Errorf("total_canceled = %v, want 1", seRec["total_canceled"])
	}

	textData, _ := os.ReadFile(logPath)
	text := string(textData)
	if !strings.Contains(text, "FINISH CANCELED") {
		t.Errorf("text log missing FINISH CANCELED\n%s", text)
	}
	if !strings.Contains(text, "canceled=1") {
		t.Errorf("text log missing canceled=1\n%s", text)
	}
}

func TestFullLifecycle_WithAttemptError(t *testing.T) {
	dir := makeTempDir(t)
	l, err := NewSessionLogger(dir, "prof-id", "my-profile", "v0.1.0", false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reqID := "fallback-req-id-abcd"
	now := time.Now()

	l.OnRequestStart(ctx, llmapimux.RequestStartEvent{
		RequestID:        reqID,
		Time:             now,
		InboundProtocol:  llmapimux.ProtocolAnthropic,
		OutboundProtocol: llmapimux.ProtocolOpenAIChat,
		Streaming:        true,
		IRRequest:        buildTestRequest("s-gpt-4o", 1, 0, ""),
	})

	// Attempt 1 fails.
	l.OnAttemptError(ctx, llmapimux.AttemptErrorEvent{
		RequestID:  reqID,
		AttemptNum: 1,
		Target: llmapimux.RouteResult{
			Protocol: llmapimux.ProtocolOpenAIChat,
			BaseURL:  "https://api.openai.com/v1",
			Model:    "gpt-4o",
		},
		SendErr: llmapimux.SendError{
			StatusCode:  429,
			IsTimeout:   false,
			IsConnError: false,
			Err:         errors.New("rate limit exceeded"),
		},
	})

	// Fallback succeeds.
	l.OnFirstByte(ctx, llmapimux.FirstByteEvent{
		RequestID: reqID,
		Time:      now.Add(1200 * time.Millisecond),
		TTFB:      1200 * time.Millisecond,
	})

	l.OnComplete(ctx, llmapimux.CompleteEvent{
		RequestID:        reqID,
		Time:             now.Add(4 * time.Second),
		Status:           llmapimux.CompletionStatusSuccess,
		TotalLatency:     4 * time.Second,
		Usage:            llmapimux.Usage{InputTokens: 800, OutputTokens: 200},
		OutputThroughput: 50.0,
		ActualModel:      "gemini-2.0-flash",
		AttemptNum:       2,
	})

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(dir)
	var jsonlPath, logPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			jsonlPath = dir + "/" + e.Name()
		}
		if strings.HasSuffix(e.Name(), ".log") {
			logPath = dir + "/" + e.Name()
		}
	}

	// Verify JSONL has attempt_error record.
	records := parseJSONLRecords(t, jsonlPath)
	types := make([]string, 0)
	for _, r := range records {
		types = append(types, r["type"].(string))
	}
	if !contains(types, "attempt_error") {
		t.Errorf("JSONL missing attempt_error record, got: %v", types)
	}

	// Find attempt_error record.
	var aeRec map[string]any
	for _, r := range records {
		if r["type"] == "attempt_error" {
			aeRec = r
			break
		}
	}
	if aeRec["target_protocol"] != "openai_chat" {
		t.Errorf("target_protocol = %v, want openai_chat", aeRec["target_protocol"])
	}
	if aeRec["status_code"] != float64(429) {
		t.Errorf("status_code = %v, want 429", aeRec["status_code"])
	}
	if aeRec["error"] != "rate limit exceeded" {
		t.Errorf("error = %v, want rate limit exceeded", aeRec["error"])
	}

	// Verify text log has FAIL and attempt info.
	textData, _ := os.ReadFile(logPath)
	text := string(textData)
	if !strings.Contains(text, "FAIL#1") {
		t.Errorf("text log missing FAIL#1\n%s", text)
	}
	if !strings.Contains(text, "FIRST_BYTE") || !strings.Contains(text, "attempt#2") {
		t.Errorf("text log should show attempt#2 on FIRST_BYTE\n%s", text)
	}
	if !strings.Contains(text, "attempt=2") {
		t.Errorf("text log missing attempt=2 on FINISH\n%s", text)
	}
}

func TestNewSessionLogger_InvalidDir(t *testing.T) {
	// Create a directory where the text file would be, but make it a file (so creating the log fails).
	dir := makeTempDir(t)

	// We can't easily make a file creation fail predictably without mocking os.Create.
	// Instead, test that if logsDir is a file (not a dir), NewSessionLogger returns an error.
	filePath := dir + "/notadir"
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Attempt to use filePath as logsDir — MkdirAll will fail since it's a file.
	// Actually MkdirAll on an existing file in the path should fail.
	// We test that no JSONL file remains if text file can't be created.
	// Instead, let's test a scenario where we can verify cleanup:
	// Use a subdir that we can control permissions on.
	restrictedDir := dir + "/restricted"
	if err := os.Mkdir(restrictedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file with the name that the .log file would have to block its creation.
	// We can't easily predict the filename (it includes timestamp + UUID).
	// So instead, test with a non-existent parent directory in logsDir to trigger MkdirAll failure.
	// Use an existing file as a component of the path.
	badDir := filePath + "/subdir"
	_, err := NewSessionLogger(badDir, "pid", "profile", "v0.1", false)
	if err == nil {
		t.Error("expected error when logsDir path is under a file, got nil")
	}
	// Verify no files leaked into the bad path (impossible since it's under a file).
}

func TestNewSessionLogger_CreatesLogsDir(t *testing.T) {
	dir := makeTempDir(t)
	subdir := dir + "/nested/logs/dir"

	l, err := NewSessionLogger(subdir, "pid", "profile", "v0.1.0", false)
	if err != nil {
		t.Fatalf("NewSessionLogger with nested dir: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(subdir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Error("expected log files to be created in nested dir")
	}
}

func TestMultipleRequests_CountersAccumulate(t *testing.T) {
	dir := makeTempDir(t)
	l, err := NewSessionLogger(dir, "prof-id", "my-profile", "v0.1.0", false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	now := time.Now()

	// 3 requests: 1 success, 1 error, 1 canceled.
	for i, status := range []llmapimux.CompletionStatus{
		llmapimux.CompletionStatusSuccess,
		llmapimux.CompletionStatusError,
		llmapimux.CompletionStatusCanceled,
	} {
		reqID := fmt.Sprintf("req-%d-abcdef123456", i)
		l.OnRequestStart(ctx, llmapimux.RequestStartEvent{
			RequestID:        reqID,
			Time:             now,
			InboundProtocol:  llmapimux.ProtocolAnthropic,
			OutboundProtocol: llmapimux.ProtocolOpenAIChat,
			Streaming:        false,
			IRRequest:        buildTestRequest("s-gpt-4o", 1, 0, ""),
		})

		var reqErr error
		if status == llmapimux.CompletionStatusError {
			reqErr = errors.New("some error")
		} else if status == llmapimux.CompletionStatusCanceled {
			reqErr = errors.New("context canceled")
		}

		l.OnComplete(ctx, llmapimux.CompleteEvent{
			RequestID:    reqID,
			Time:         now.Add(time.Second),
			Status:       status,
			Error:        reqErr,
			TotalLatency: time.Second,
			Usage:        llmapimux.Usage{InputTokens: 100, OutputTokens: 50},
			AttemptNum:   1,
		})
	}

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(dir)
	var jsonlPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			jsonlPath = dir + "/" + e.Name()
		}
	}

	records := parseJSONLRecords(t, jsonlPath)
	seRec := records[len(records)-1]
	if seRec["total_requests"] != float64(3) {
		t.Errorf("total_requests = %v, want 3", seRec["total_requests"])
	}
	if seRec["total_errors"] != float64(1) {
		t.Errorf("total_errors = %v, want 1", seRec["total_errors"])
	}
	if seRec["total_canceled"] != float64(1) {
		t.Errorf("total_canceled = %v, want 1", seRec["total_canceled"])
	}
	if seRec["total_input_tokens"] != float64(300) {
		t.Errorf("total_input_tokens = %v, want 300", seRec["total_input_tokens"])
	}
	if seRec["total_output_tokens"] != float64(150) {
		t.Errorf("total_output_tokens = %v, want 150", seRec["total_output_tokens"])
	}
}

func TestReqShort(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"7f3a1b2c-abcd-4567", "7f3a"},
		{"abcd", "abcd"},
		{"ab", "ab"},
		{"", ""},
	}
	for _, tt := range tests {
		got := reqShort(tt.id)
		if got != tt.want {
			t.Errorf("reqShort(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

// contains checks if a string slice contains a value.
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
