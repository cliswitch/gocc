package stats

import "time"

// SessionStartRecord is written once at the beginning of each gocc session.
type SessionStartRecord struct {
	Type        string    `json:"type"`
	Time        time.Time `json:"time"`
	SessionID   string    `json:"session_id"`
	ProfileID   string    `json:"profile_id"`
	ProfileName string    `json:"profile_name"`
	GoccVersion string    `json:"gocc_version"`
	PID         int       `json:"pid"`
}

// RequestStartRecord is written when a new proxy request begins.
type RequestStartRecord struct {
	Type             string    `json:"type"`
	Time             time.Time `json:"time"`
	RequestID        string    `json:"request_id"`
	InboundProtocol  string    `json:"inbound_protocol"`
	OutboundProtocol string    `json:"outbound_protocol"`
	Streaming        bool      `json:"streaming"`
	ModelLevel       string    `json:"model_level"`
	RequestModel     string    `json:"request_model"`
	MessageCount     int       `json:"message_count"`
	HasTools         bool      `json:"has_tools"`
	HasSystem        bool      `json:"has_system"`
}

// FirstByteRecord is written when the first response byte is received.
type FirstByteRecord struct {
	Type      string    `json:"type"`
	Time      time.Time `json:"time"`
	RequestID string    `json:"request_id"`
	TTFBMS    int64     `json:"ttfb_ms"`
}

// StreamChunkRecord is written for each streaming chunk received.
type StreamChunkRecord struct {
	Type         string    `json:"type"`
	Time         time.Time `json:"time"`
	RequestID    string    `json:"request_id"`
	Seq          int       `json:"seq"`
	ElapsedMS    int64     `json:"elapsed_ms"`
	InterChunkMS int64     `json:"inter_chunk_ms"`
	EventType    string    `json:"event_type"`
	HasUsage     bool      `json:"has_usage"`
}

// CompleteRecord is written when a request finishes (success, error, or canceled).
type CompleteRecord struct {
	Type                string    `json:"type"`
	Time                time.Time `json:"time"`
	RequestID           string    `json:"request_id"`
	Status              string    `json:"status"`
	Error               string    `json:"error,omitempty"`
	TTFBMS              int64     `json:"ttfb_ms,omitempty"`
	TotalMS             int64     `json:"total_ms"`
	InputTokens         int64     `json:"input_tokens,omitempty"`
	OutputTokens        int64     `json:"output_tokens,omitempty"`
	CacheReadTokens     int64     `json:"cache_read_tokens"`
	CacheCreationTokens int64     `json:"cache_creation_tokens"`
	ThinkingTokens      int64     `json:"thinking_tokens"`
	OutputTPS           float64   `json:"output_tps,omitempty"`
	StopReason          string    `json:"stop_reason,omitempty"`
	ActualModel         string    `json:"actual_model,omitempty"`
	AttemptNum          int       `json:"attempt_num,omitempty"`
}

// AttemptErrorRecord is written when a single attempt (within a fallback chain) fails.
type AttemptErrorRecord struct {
	Type          string    `json:"type"`
	Time          time.Time `json:"time"`
	RequestID     string    `json:"request_id"`
	AttemptNum    int       `json:"attempt_num"`
	TargetProtocol string   `json:"target_protocol"`
	TargetBaseURL string    `json:"target_base_url"`
	TargetModel   string    `json:"target_model"`
	StatusCode    int       `json:"status_code"`
	IsTimeout     bool      `json:"is_timeout"`
	IsConnError   bool      `json:"is_conn_error"`
	Error         string    `json:"error"`
}

// SessionEndRecord is written once when the gocc session exits.
type SessionEndRecord struct {
	Type               string    `json:"type"`
	Time               time.Time `json:"time"`
	SessionID          string    `json:"session_id"`
	DurationMS         int64     `json:"duration_ms"`
	TotalRequests      int64     `json:"total_requests"`
	TotalErrors        int64     `json:"total_errors"`
	TotalCanceled      int64     `json:"total_canceled"`
	TotalInputTokens   int64     `json:"total_input_tokens"`
	TotalOutputTokens  int64     `json:"total_output_tokens"`
}

// requestState tracks per-request state for the SessionLogger.
type requestState struct {
	startTime    time.Time
	hasFirstByte bool
	attemptNum   int
	modelLevel   string
	requestModel string
	streaming    bool
	messageCount int
	toolCount    int
}
