package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cliswitch/gocc/internal/config"
)

func TestResilienceEditApplyParsesFieldsAndToggles(t *testing.T) {
	re := newResilienceEditModel(config.Resilience{})
	re.inputs[0].SetValue("6")
	re.inputs[1].SetValue("1500")
	re.inputs[2].SetValue("2")
	re.inputs[3].SetValue("250")
	re.inputs[4].SetValue("5000")
	re.retry429 = true

	got, err := re.applyToResilience()
	if err != nil {
		t.Fatalf("applyToResilience: %v", err)
	}
	if got.MaxConcurrentRequests != 6 || got.QueueTimeoutMS != 1500 {
		t.Fatalf("resilience = %+v, want max=6 queue=1500", got)
	}
	if got.Retry.MaxRetries != 2 || got.Retry.BaseDelayMS != 250 || got.Retry.MaxDelayMS != 5000 {
		t.Fatalf("retry = %+v, want max=2 base=250 max_delay=5000", got.Retry)
	}
	if got.Retry.Retry5xx == nil || !*got.Retry.Retry5xx {
		t.Fatal("retry_5xx should be explicitly true")
	}
	if got.Retry.RetryTransport == nil || !*got.Retry.RetryTransport {
		t.Fatal("retry_transport should be explicitly true")
	}
	if got.Retry.Retry429 == nil || !*got.Retry.Retry429 {
		t.Fatal("retry_429 should be explicitly true")
	}
}

func TestResilienceEditClearFieldsDisablesConfig(t *testing.T) {
	re := newResilienceEditModel(config.Resilience{
		MaxConcurrentRequests: 6,
		QueueTimeoutMS:        1500,
		Retry: config.RetryConfig{
			MaxRetries: 2,
		},
	})
	for i := range re.inputs {
		re.inputs[i].SetValue("")
	}
	re.retry5xx = true
	re.retryTransport = true
	re.retry429 = false

	got, err := re.applyToResilience()
	if err != nil {
		t.Fatalf("applyToResilience: %v", err)
	}
	if got != (config.Resilience{}) {
		t.Fatalf("cleared resilience = %+v, want zero value", got)
	}
}

func TestResilienceEditRejectsNegativeNumbers(t *testing.T) {
	re := newResilienceEditModel(config.Resilience{})
	re.inputs[0].SetValue("-1")

	err := re.validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "max_concurrent_requests") {
		t.Fatalf("error = %v, want max_concurrent_requests", err)
	}
}

func TestResilienceEditSaveWritesProfileAndMarksDirty(t *testing.T) {
	p := config.Profile{
		ID:       "abc123",
		Name:     "Test",
		Protocol: config.ProtocolOpenAI,
	}
	pf := newProfileFormModel(p, false, Callbacks{})
	m := Model{
		mode:           ModeResilienceEdit,
		profileForm:    pf,
		resilienceEdit: newResilienceEditModel(config.Resilience{}),
	}
	m.resilienceEdit.inputs[0].SetValue("6")
	m.resilienceEdit.inputs[2].SetValue("2")

	m = updateModel(m, keyType(tea.KeyCtrlS))

	if m.mode != ModeProfileEdit {
		t.Fatalf("mode = %v, want ModeProfileEdit", m.mode)
	}
	if m.resilienceEdit != nil {
		t.Fatal("resilienceEdit should be cleared after save")
	}
	if got := m.profileForm.profile.Resilience.MaxConcurrentRequests; got != 6 {
		t.Fatalf("max_concurrent_requests = %d, want 6", got)
	}
	if got := m.profileForm.profile.Resilience.Retry.MaxRetries; got != 2 {
		t.Fatalf("max_retries = %d, want 2", got)
	}
	if !m.profileForm.isDirty() {
		t.Fatal("profile form should be dirty after resilience edit")
	}
}
