package stats

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/llmapimux/llmapimux"
	"github.com/cliswitch/gocc/internal/config"
)

// SessionLogger implements llmapimux.StatsReporter and writes log records in
// both JSONL and human-readable text formats for a single gocc session.
type SessionLogger struct {
	sessionID   string
	startTime   time.Time
	profileID   string
	profileName string
	goccVersion string

	jsonlFile *os.File
	textFile  *os.File

	mu              sync.Mutex
	totalRequests   int
	totalErrors     int
	totalCanceled   int
	totalInput      int
	totalOutput     int
	totalActiveTime time.Duration // sum of each request's TotalLatency (excludes idle time)

	requests map[string]*requestState

	writeErrOnce sync.Once
	closeOnce    sync.Once
}

// NewSessionLogger creates a new SessionLogger, generating a UUID session ID,
// creating two log files (JSONL and text) in logsDir, and writing the session start records.
// If logsDir does not exist, it is created. If text file creation fails after JSONL file was
// created, the JSONL file is cleaned up and an error is returned.
func NewSessionLogger(logsDir, profileID, profileName, goccVersion string) (*SessionLogger, error) {
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, fmt.Errorf("stats: create logs dir: %w", err)
	}

	sessionID := uuid.New().String()
	now := time.Now()
	ts := now.Format("2006-01-02T15-04-05")
	baseName := ts + "_" + sessionID

	jsonlPath := filepath.Join(logsDir, baseName+".jsonl")
	textPath := filepath.Join(logsDir, baseName+".log")

	jsonlFile, err := os.Create(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("stats: create jsonl file: %w", err)
	}

	textFile, err := os.Create(textPath)
	if err != nil {
		jsonlFile.Close()
		os.Remove(jsonlPath)
		return nil, fmt.Errorf("stats: create text log file: %w", err)
	}

	l := &SessionLogger{
		sessionID:   sessionID,
		startTime:   now,
		profileID:   profileID,
		profileName: profileName,
		goccVersion: goccVersion,
		jsonlFile:   jsonlFile,
		textFile:    textFile,
		requests:    make(map[string]*requestState),
	}

	pid := os.Getpid()

	// Write session start records.
	rec := SessionStartRecord{
		Type:        "session_start",
		Time:        now,
		SessionID:   sessionID,
		ProfileID:   profileID,
		ProfileName: profileName,
		GoccVersion: goccVersion,
		PID:         pid,
	}
	if err := writeJSONL(jsonlFile, rec); err != nil {
		l.handleWriteError(err)
	}
	writeTextSessionHeader(textFile, sessionID, now, profileName, pid)

	return l, nil
}

// handleWriteError is called on the first write error, printing a warning to stderr.
func (l *SessionLogger) handleWriteError(err error) {
	l.writeErrOnce.Do(func() {
		fmt.Fprintf(os.Stderr, "gocc: stats log write error: %v\n", err)
	})
}

// OnRequestStart implements llmapimux.StatsReporter.
func (l *SessionLogger) OnRequestStart(ctx context.Context, e llmapimux.RequestStartEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var modelLevel, requestModel string
	if e.IRRequest != nil {
		lvl, mdl, ok := config.ParseAnnotatedModel(e.IRRequest.Model)
		if ok {
			modelLevel = lvl
			requestModel = mdl
		} else {
			// Not annotated — use model as-is for requestModel.
			modelLevel = ""
			requestModel = e.IRRequest.Model
		}
	}

	hasTools := false
	hasSystem := false
	toolCount := 0
	messageCount := 0
	if e.IRRequest != nil {
		hasTools = len(e.IRRequest.Tools) > 0
		hasSystem = len(e.IRRequest.SystemPrompt) > 0
		toolCount = len(e.IRRequest.Tools)
		messageCount = len(e.IRRequest.Messages)
	}

	rs := &requestState{
		startTime:    e.Time,
		hasFirstByte: false,
		attemptNum:   0,
		modelLevel:   modelLevel,
		requestModel: requestModel,
		streaming:    e.Streaming,
		messageCount: messageCount,
		toolCount:    toolCount,
	}
	l.requests[e.RequestID] = rs

	// Write JSONL record.
	rec := RequestStartRecord{
		Type:             "request_start",
		Time:             e.Time,
		RequestID:        e.RequestID,
		InboundProtocol:  string(e.InboundProtocol),
		OutboundProtocol: string(e.OutboundProtocol),
		Streaming:        e.Streaming,
		ModelLevel:       modelLevel,
		RequestModel:     requestModel,
		MessageCount:     messageCount,
		HasTools:         hasTools,
		HasSystem:        hasSystem,
	}
	if err := writeJSONL(l.jsonlFile, rec); err != nil {
		l.handleWriteError(err)
	}

	// Write text log.
	reqIDShort := reqShort(e.RequestID)
	writeTextRequestStart(l.textFile, reqIDShort, e.Time, l.profileName, rs)
}

// OnFirstByte implements llmapimux.StatsReporter.
func (l *SessionLogger) OnFirstByte(ctx context.Context, e llmapimux.FirstByteEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	rs, ok := l.requests[e.RequestID]
	if !ok {
		return
	}
	rs.hasFirstByte = true

	// Write JSONL record.
	rec := FirstByteRecord{
		Type:      "first_byte",
		Time:      e.Time,
		RequestID: e.RequestID,
		TTFBMS:    e.TTFB.Milliseconds(),
	}
	if err := writeJSONL(l.jsonlFile, rec); err != nil {
		l.handleWriteError(err)
	}

	// Write text log.
	writeTextFirstByte(l.textFile, e.Time, e.TTFB, rs.attemptNum)
}

// OnStreamChunk implements llmapimux.StatsReporter. Only JSONL is written — text log skips chunks.
func (l *SessionLogger) OnStreamChunk(ctx context.Context, e llmapimux.StreamChunkEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	eventType := ""
	hasUsage := false
	if e.IREvent != nil {
		eventType = string(e.IREvent.Type)
		hasUsage = e.IREvent.Usage != nil
	}

	rec := StreamChunkRecord{
		Type:         "stream_chunk",
		Time:         e.Time,
		RequestID:    e.RequestID,
		Seq:          e.SequenceNum,
		ElapsedMS:    e.ElapsedTime.Milliseconds(),
		InterChunkMS: e.InterChunkDelay.Milliseconds(),
		EventType:    eventType,
		HasUsage:     hasUsage,
	}
	if err := writeJSONL(l.jsonlFile, rec); err != nil {
		l.handleWriteError(err)
	}
}

// OnAttemptError implements llmapimux.StatsReporter.
func (l *SessionLogger) OnAttemptError(ctx context.Context, e llmapimux.AttemptErrorEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	rs, ok := l.requests[e.RequestID]
	if !ok {
		return
	}

	elapsed := now.Sub(rs.startTime)
	rs.attemptNum = e.AttemptNum

	errMsg := ""
	if e.SendErr.Err != nil {
		errMsg = e.SendErr.Err.Error()
	}

	// Write JSONL record.
	rec := AttemptErrorRecord{
		Type:           "attempt_error",
		Time:           now,
		RequestID:      e.RequestID,
		AttemptNum:     e.AttemptNum,
		TargetProtocol: string(e.Target.Protocol),
		TargetBaseURL:  e.Target.BaseURL,
		TargetModel:    e.Target.Model,
		StatusCode:     e.SendErr.StatusCode,
		IsTimeout:      e.SendErr.IsTimeout,
		IsConnError:    e.SendErr.IsConnError,
		Error:          errMsg,
	}
	if err := writeJSONL(l.jsonlFile, rec); err != nil {
		l.handleWriteError(err)
	}

	// Write text log.
	writeTextAttemptError(l.textFile, now, e.AttemptNum, string(e.Target.Protocol),
		e.Target.BaseURL, e.Target.Model, e.SendErr.StatusCode, elapsed, errMsg)
}

// OnComplete implements llmapimux.StatsReporter.
func (l *SessionLogger) OnComplete(ctx context.Context, e llmapimux.CompleteEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Map status to text.
	var textStatus string
	switch e.Status {
	case llmapimux.CompletionStatusSuccess:
		textStatus = "OK"
	case llmapimux.CompletionStatusError:
		textStatus = "ERROR"
	case llmapimux.CompletionStatusCanceled:
		textStatus = "CANCELED"
	default:
		textStatus = string(e.Status)
	}

	// Accumulate counters.
	l.totalRequests++
	switch e.Status {
	case llmapimux.CompletionStatusError:
		l.totalErrors++
	case llmapimux.CompletionStatusCanceled:
		l.totalCanceled++
	}
	l.totalInput += e.Usage.InputTokens
	l.totalOutput += e.Usage.OutputTokens
	l.totalActiveTime += e.TotalLatency

	errMsg := ""
	if e.Error != nil {
		errMsg = e.Error.Error()
	}

	stopReason := string(e.StopReason)

	// Write JSONL record.
	rec := CompleteRecord{
		Type:                "complete",
		Time:                e.Time,
		RequestID:           e.RequestID,
		Status:              string(e.Status),
		Error:               errMsg,
		TTFBMS:              e.TTFB.Milliseconds(),
		TotalMS:             e.TotalLatency.Milliseconds(),
		InputTokens:         int64(e.Usage.InputTokens),
		OutputTokens:        int64(e.Usage.OutputTokens),
		CacheReadTokens:     int64(e.Usage.CacheReadTokens),
		CacheCreationTokens: int64(e.Usage.CacheCreationTokens),
		ThinkingTokens:      int64(e.Usage.ThinkingTokens),
		OutputTPS:           e.OutputThroughput,
		StopReason:          stopReason,
		ActualModel:         e.ActualModel,
		AttemptNum:          e.AttemptNum,
	}
	if err := writeJSONL(l.jsonlFile, rec); err != nil {
		l.handleWriteError(err)
	}

	// Write text log.
	writeTextComplete(l.textFile, e.Time, textStatus, e.TotalLatency,
		e.Usage.InputTokens, e.Usage.OutputTokens, e.Usage.CacheReadTokens, e.Usage.ThinkingTokens,
		e.OutputThroughput, e.ActualModel, e.AttemptNum, errMsg)
	writeTextRequestEnd(l.textFile)

	// Clean up requestState.
	delete(l.requests, e.RequestID)
}

// Close writes the session_end footer and closes both log files.
// It is safe to call Close multiple times; only the first call takes effect.
func (l *SessionLogger) Close() error {
	var closeErr error
	l.closeOnce.Do(func() {
		l.mu.Lock()
		defer l.mu.Unlock()

		// Warn about orphaned requests.
		if len(l.requests) > 0 {
			fmt.Fprintf(os.Stderr, "gocc: stats: %d orphaned request(s) at session close\n", len(l.requests))
		}

		now := time.Now()
		dur := now.Sub(l.startTime)

		// Write JSONL session_end.
		rec := SessionEndRecord{
			Type:              "session_end",
			Time:              now,
			SessionID:         l.sessionID,
			DurationMS:        dur.Milliseconds(),
			TotalRequests:     int64(l.totalRequests),
			TotalErrors:       int64(l.totalErrors),
			TotalCanceled:     int64(l.totalCanceled),
			TotalInputTokens:  int64(l.totalInput),
			TotalOutputTokens: int64(l.totalOutput),
		}
		if err := writeJSONL(l.jsonlFile, rec); err != nil {
			l.handleWriteError(err)
		}

		// Write text session footer.
		writeTextSessionFooter(l.textFile, dur, l.totalRequests, l.totalErrors, l.totalCanceled, l.totalInput, l.totalOutput, l.totalActiveTime)

		var firstErr error
		if err := l.jsonlFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := l.textFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		closeErr = firstErr
	})
	return closeErr
}

// reqShort returns the first 4 characters of a request ID for display.
func reqShort(id string) string {
	if len(id) <= 4 {
		return id
	}
	return id[:4]
}
