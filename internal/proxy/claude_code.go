package proxy

import (
	"strings"

	"github.com/llmapimux/llmapimux"
)

// claudeCodeBillingHeaderPrefix identifies the per-request billing header that
// Claude Code injects as the first system text block when calling the
// Anthropic API. A typical line looks like:
//
//	x-anthropic-billing-header: cc_version=2.1.114.c4f; cc_entrypoint=cli; cch=<nonce>;
//
// The `cch=` portion is a nonce that changes every request. Anthropic knows
// how to consume this marker; non-Anthropic providers treat it as ordinary
// system prompt content, which means the per-request nonce destroys their
// token-level prefix match and prevents prompt caching from ever hitting.
const claudeCodeBillingHeaderPrefix = "x-anthropic-billing-header:"

// stripClaudeCodeBillingHeader removes Claude Code's billing header line from
// the first system text part when the outbound target is not Anthropic. For
// Anthropic passthroughs the marker is preserved so Anthropic's own billing
// pipeline keeps working.
//
// The header is always emitted by Claude Code as the first piece of the system
// content. Two shapes are handled:
//   - The first text part is a standalone block whose entire text is the
//     header line. The part is dropped.
//   - The first text part starts with the header followed by a newline and
//     the actual system prompt. Only the header line is trimmed.
func stripClaudeCodeBillingHeader(req *llmapimux.Request, target llmapimux.RouteResult) {
	if req == nil || target.Protocol == llmapimux.ProtocolAnthropic {
		return
	}
	if len(req.SystemPrompt) == 0 {
		return
	}
	first := &req.SystemPrompt[0]
	if first.Type != llmapimux.ContentTypeText || first.Text == nil {
		return
	}
	text := first.Text.Text
	if !strings.HasPrefix(text, claudeCodeBillingHeaderPrefix) {
		return
	}
	if nl := strings.IndexByte(text, '\n'); nl >= 0 {
		first.Text.Text = text[nl+1:]
		return
	}
	req.SystemPrompt = req.SystemPrompt[1:]
}
