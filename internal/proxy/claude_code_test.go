package proxy

import (
	"testing"

	"github.com/llmapimux/llmapimux"
)

func textPart(s string) llmapimux.ContentPart {
	return llmapimux.ContentPart{
		Type: llmapimux.ContentTypeText,
		Text: &llmapimux.TextContent{Text: s},
	}
}

func TestStripClaudeCodeBillingHeader_NonAnthropicDropsInlineHeader(t *testing.T) {
	req := &llmapimux.Request{
		SystemPrompt: []llmapimux.ContentPart{
			textPart("x-anthropic-billing-header: cc_version=2.1; cch=abcd;\nYou are Claude Code."),
		},
	}
	stripClaudeCodeBillingHeader(req, llmapimux.RouteResult{Protocol: llmapimux.ProtocolOpenAIResponses})
	if got := req.SystemPrompt[0].Text.Text; got != "You are Claude Code." {
		t.Fatalf("system text = %q, want %q", got, "You are Claude Code.")
	}
}

func TestStripClaudeCodeBillingHeader_NonAnthropicDropsStandaloneBlock(t *testing.T) {
	req := &llmapimux.Request{
		SystemPrompt: []llmapimux.ContentPart{
			textPart("x-anthropic-billing-header: cc_version=2.1; cch=abcd;"),
			textPart("You are Claude Code."),
		},
	}
	stripClaudeCodeBillingHeader(req, llmapimux.RouteResult{Protocol: llmapimux.ProtocolOpenAIChat})
	if len(req.SystemPrompt) != 1 {
		t.Fatalf("system len = %d, want 1", len(req.SystemPrompt))
	}
	if got := req.SystemPrompt[0].Text.Text; got != "You are Claude Code." {
		t.Errorf("system[0] = %q, want remaining prompt", got)
	}
}

func TestStripClaudeCodeBillingHeader_AnthropicPassthroughUnchanged(t *testing.T) {
	original := "x-anthropic-billing-header: cc_version=2.1; cch=abcd;\nYou are Claude Code."
	req := &llmapimux.Request{
		SystemPrompt: []llmapimux.ContentPart{textPart(original)},
	}
	stripClaudeCodeBillingHeader(req, llmapimux.RouteResult{Protocol: llmapimux.ProtocolAnthropic})
	if got := req.SystemPrompt[0].Text.Text; got != original {
		t.Errorf("Anthropic passthrough altered system: got %q", got)
	}
}

func TestStripClaudeCodeBillingHeader_NoHeaderUnchanged(t *testing.T) {
	original := "You are Claude Code."
	req := &llmapimux.Request{
		SystemPrompt: []llmapimux.ContentPart{textPart(original)},
	}
	stripClaudeCodeBillingHeader(req, llmapimux.RouteResult{Protocol: llmapimux.ProtocolOpenAIResponses})
	if got := req.SystemPrompt[0].Text.Text; got != original {
		t.Errorf("system without header altered: got %q", got)
	}
}

func TestStripClaudeCodeBillingHeader_EmptySystemUnchanged(t *testing.T) {
	req := &llmapimux.Request{}
	stripClaudeCodeBillingHeader(req, llmapimux.RouteResult{Protocol: llmapimux.ProtocolOpenAIResponses})
	if len(req.SystemPrompt) != 0 {
		t.Errorf("empty system gained parts: %v", req.SystemPrompt)
	}
}

func TestStripClaudeCodeBillingHeader_NonTextFirstPartUnchanged(t *testing.T) {
	// If the first system part is not a plain-text block, the helper should
	// leave it alone. Claude Code has never been observed to do this, but
	// the guard keeps the behaviour predictable for unexpected inputs.
	req := &llmapimux.Request{
		SystemPrompt: []llmapimux.ContentPart{
			{Type: llmapimux.ContentTypeImage},
			textPart("x-anthropic-billing-header: cch=abcd;"),
		},
	}
	stripClaudeCodeBillingHeader(req, llmapimux.RouteResult{Protocol: llmapimux.ProtocolOpenAIResponses})
	if len(req.SystemPrompt) != 2 {
		t.Fatalf("system len = %d, want 2 (no strip when non-text first)", len(req.SystemPrompt))
	}
}
