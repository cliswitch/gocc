package stats

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	textSeparatorDouble = "══════════════════════════════════════════════════════"
	textSeparatorSingle = "────────────────────────────────────────────────────"
)

// fmtDuration renders a duration in a human-readable short form.
// Values < 1s use ms unit, values < 1min use one-decimal s, larger use Go's Duration.String().
func fmtDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return d.Truncate(time.Second).String()
}

// extractHost returns just the hostname from a URL string.
// If parsing fails, returns the original string.
func extractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	// Strip port if present to match spec examples like "api.openai.com"
	host := u.Hostname()
	return host
}

// writeTextSessionHeader writes the session start banner to the text log file.
func writeTextSessionHeader(f *os.File, sessionID string, t time.Time, profileName string, pid int) {
	fmt.Fprintf(f, "%s\n", textSeparatorDouble)
	fmt.Fprintf(f, "  gocc session %s\n", sessionID)
	fmt.Fprintf(f, "  %s  profile=%s  pid=%d\n", t.Format("2006-01-02 15:04:05"), profileName, pid)
	fmt.Fprintf(f, "%s\n", textSeparatorDouble)
}

// writeTextRequestStart writes the request separator and START line.
func writeTextRequestStart(f *os.File, reqIDShort string, t time.Time, profileName string, rs *requestState) {
	fmt.Fprintf(f, "── req=%s %s\n", reqIDShort, strings.Repeat("─", max(0, utf8.RuneCountInString(textSeparatorSingle)-9-len(reqIDShort))))
	line := fmt.Sprintf("%s START  [%s] %s model=%s stream=%v msgs=%d",
		t.Format("15:04:05"), profileName, rs.modelLevel, rs.requestModel,
		rs.streaming, rs.messageCount)
	if rs.toolCount > 0 {
		line += fmt.Sprintf(" tools=%d", rs.toolCount)
	}
	fmt.Fprintln(f, line)
}

// writeTextFirstByte writes the FIRST_BYTE line.
// attemptNum is the number of failed attempts before this first byte; if >= 1, appends "(attempt#N)".
func writeTextFirstByte(f *os.File, t time.Time, ttfb time.Duration, attemptNum int) {
	line := fmt.Sprintf("%s FIRST_BYTE ttfb=%s", t.Format("15:04:05"), fmtDuration(ttfb))
	if attemptNum >= 1 {
		line += fmt.Sprintf(" (attempt#%d)", attemptNum+1)
	}
	fmt.Fprintln(f, line)
}

// writeTextAttemptError writes a FAIL#N line for a failed attempt in a fallback chain.
func writeTextAttemptError(f *os.File, t time.Time, attemptNum int, protocol string, baseURL string, model string, statusCode int, elapsed time.Duration, errMsg string) {
	host := extractHost(baseURL)
	line := fmt.Sprintf("%s FAIL#%d %s %s model=%s status=%d dur=%s %q",
		t.Format("15:04:05"), attemptNum, protocol, host, model, statusCode, fmtDuration(elapsed), errMsg)
	fmt.Fprintln(f, line)
}

// writeTextComplete writes the FINISH line for a completed request.
// status is one of "OK", "ERROR", "CANCELED".
// Zero-value token fields are omitted. tps is omitted when 0.
// actualModel and attemptNum are only shown when relevant (OK with attempt>1, etc.).
func writeTextComplete(f *os.File, t time.Time, status string, totalDur time.Duration, inputTokens int, outputTokens int, cacheReadTokens int, thinkingTokens int, tps float64, actualModel string, attemptNum int, errMsg string) {
	line := fmt.Sprintf("%s FINISH %s dur=%s", t.Format("15:04:05"), status, fmtDuration(totalDur))
	if inputTokens > 0 {
		line += fmt.Sprintf(" in=%d", inputTokens)
	}
	if outputTokens > 0 {
		line += fmt.Sprintf(" out=%d", outputTokens)
	}
	if cacheReadTokens > 0 {
		line += fmt.Sprintf(" cache_r=%d", cacheReadTokens)
	}
	if thinkingTokens > 0 {
		line += fmt.Sprintf(" think=%d", thinkingTokens)
	}
	if tps > 0 {
		line += fmt.Sprintf(" tok/s=%.1f", tps)
	}
	if actualModel != "" {
		line += fmt.Sprintf(" model=%s", actualModel)
	}
	if attemptNum > 1 {
		line += fmt.Sprintf(" attempt=%d", attemptNum)
	}
	if errMsg != "" {
		line += fmt.Sprintf(" %q", errMsg)
	}
	fmt.Fprintln(f, line)
}

// writeTextRequestEnd writes the closing separator for a request block.
func writeTextRequestEnd(f *os.File) {
	fmt.Fprintf(f, "%s\n", textSeparatorSingle)
}

// writeTextSessionFooter writes the session end banner to the text log file.
func writeTextSessionFooter(f *os.File, dur time.Duration, totalReqs int, totalErrors int, totalCanceled int, totalInput int, totalOutput int, totalActiveTime time.Duration) {
	fmt.Fprintf(f, "%s\n", textSeparatorDouble)
	fmt.Fprintf(f, "  session end  dur=%s  requests=%d  errors=%d  canceled=%d\n",
		fmtDuration(dur), totalReqs, totalErrors, totalCanceled)
	tokPerSec := ""
	if totalActiveTime > 0 {
		avgTPS := float64(totalOutput) / totalActiveTime.Seconds()
		tokPerSec = fmt.Sprintf("  avg_tok/s=%.1f", avgTPS)
	}
	fmt.Fprintf(f, "  tokens: in=%d out=%d%s\n", totalInput, totalOutput, tokPerSec)
	fmt.Fprintf(f, "%s\n", textSeparatorDouble)
}

