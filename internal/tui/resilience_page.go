package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cliswitch/gocc/internal/config"
)

const (
	reMaxConcurrent = iota
	reQueueTimeout
	reMaxRetries
	reBaseDelay
	reMaxDelay
	reRetry5xx
	reRetryTransport
	reRetry429
	reFieldCount
)

type resilienceEditModel struct {
	inputs []textinput.Model
	focus  int

	retry5xx       bool
	retryTransport bool
	retry429       bool
	statusMsg      string
}

func newResilienceEditModel(r config.Resilience) *resilienceEditModel {
	re := &resilienceEditModel{
		inputs:         make([]textinput.Model, 5),
		retry5xx:       r.Retry.ShouldRetry5xx(),
		retryTransport: r.Retry.ShouldRetryTransport(),
		retry429:       r.Retry.ShouldRetry429(),
	}
	re.inputs[0] = newTextInput(formatInt(r.MaxConcurrentRequests))
	re.inputs[1] = newTextInput(formatInt(r.QueueTimeoutMS))
	re.inputs[2] = newTextInput(formatInt(r.Retry.MaxRetries))
	re.inputs[3] = newTextInput(formatInt(r.Retry.BaseDelayMS))
	re.inputs[4] = newTextInput(formatInt(r.Retry.MaxDelayMS))
	re.inputs[0].Focus()
	return re
}

func (re *resilienceEditModel) updateFocus() {
	for i := range re.inputs {
		re.inputs[i].Blur()
	}
	if re.focus >= 0 && re.focus < len(re.inputs) {
		re.inputs[re.focus].Focus()
	}
}

func (re *resilienceEditModel) validate() error {
	for i, label := range []string{
		"max_concurrent_requests",
		"queue_timeout_ms",
		"max_retries",
		"base_delay_ms",
		"max_delay_ms",
	} {
		if _, err := parseOptionalNonNegativeInt(label, re.inputs[i].Value()); err != nil {
			return err
		}
	}
	return nil
}

func (re *resilienceEditModel) applyToResilience() (config.Resilience, error) {
	if err := re.validate(); err != nil {
		return config.Resilience{}, err
	}
	maxConcurrent, _ := parseOptionalNonNegativeInt("max_concurrent_requests", re.inputs[0].Value())
	queueTimeout, _ := parseOptionalNonNegativeInt("queue_timeout_ms", re.inputs[1].Value())
	maxRetries, _ := parseOptionalNonNegativeInt("max_retries", re.inputs[2].Value())
	baseDelay, _ := parseOptionalNonNegativeInt("base_delay_ms", re.inputs[3].Value())
	maxDelay, _ := parseOptionalNonNegativeInt("max_delay_ms", re.inputs[4].Value())

	r := config.Resilience{
		MaxConcurrentRequests: maxConcurrent,
		QueueTimeoutMS:        queueTimeout,
		Retry: config.RetryConfig{
			MaxRetries:  maxRetries,
			BaseDelayMS: baseDelay,
			MaxDelayMS:  maxDelay,
		},
	}
	if maxRetries > 0 || !retryTogglesDefault(re.retry5xx, re.retryTransport, re.retry429) {
		r.Retry.Retry5xx = boolPtr(re.retry5xx)
		r.Retry.RetryTransport = boolPtr(re.retryTransport)
		r.Retry.Retry429 = boolPtr(re.retry429)
	}
	return r, nil
}

func parseOptionalNonNegativeInt(label, value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %s", label, value)
	}
	if n < 0 {
		return 0, fmt.Errorf("invalid %s: must be >= 0", label)
	}
	return n, nil
}

func retryTogglesDefault(retry5xx, retryTransport, retry429 bool) bool {
	return retry5xx && retryTransport && !retry429
}

func boolPtr(v bool) *bool {
	return &v
}

func (re *resilienceEditModel) toggleFocused() {
	switch re.focus {
	case reRetry5xx:
		re.retry5xx = !re.retry5xx
	case reRetryTransport:
		re.retryTransport = !re.retryTransport
	case reRetry429:
		re.retry429 = !re.retry429
	}
}

func (m Model) updateResilienceEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	re := m.resilienceEdit
	if re == nil {
		m.returnToProfileForm()
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "ctrl+s":
			resilience, err := re.applyToResilience()
			if err != nil {
				re.statusMsg = err.Error()
				return m, nil
			}
			if m.profileForm != nil {
				m.profileForm.profile.Resilience = resilience
			}
			m.resilienceEdit = nil
			m.returnToProfileForm()
			return m, nil

		case "esc":
			m.resilienceEdit = nil
			m.returnToProfileForm()
			return m, nil

		case "tab", "down":
			re.focus = (re.focus + 1) % reFieldCount
			re.updateFocus()
			return m, nil

		case "shift+tab", "up":
			re.focus = (re.focus - 1 + reFieldCount) % reFieldCount
			re.updateFocus()
			return m, nil

		case "enter", "space", "left", "right":
			re.toggleFocused()
			return m, nil
		}
	}

	if re.focus >= 0 && re.focus < len(re.inputs) {
		var cmd tea.Cmd
		re.inputs[re.focus], cmd = re.inputs[re.focus].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) viewResilienceEdit() string {
	re := m.resilienceEdit
	if re == nil {
		return ""
	}

	s := titleStyle.Render("Edit Resilience") + "\n\n"
	labels := []string{
		"max_concurrent_requests",
		"queue_timeout_ms",
		"retry.max_retries",
		"retry.base_delay_ms",
		"retry.max_delay_ms",
	}
	for i, label := range labels {
		s += re.viewInputField(i, label)
	}
	s += re.viewToggleField(reRetry5xx, "retry.retry_5xx", re.retry5xx)
	s += re.viewToggleField(reRetryTransport, "retry.retry_transport", re.retryTransport)
	s += re.viewToggleField(reRetry429, "retry.retry_429", re.retry429)

	if re.statusMsg != "" {
		s += "\n" + dimStyle.Render("  "+re.statusMsg)
	}

	hints := []string{"tab/↓ next", "shift+tab/↑ prev", "enter/space toggle", "ctrl+s save", "esc cancel"}
	s += "\n" + renderStatusBar(hints)
	return s
}

func (re *resilienceEditModel) viewInputField(focusIdx int, label string) string {
	focused := re.focus == focusIdx
	var content string
	if focused {
		content = re.inputs[focusIdx].View()
	} else {
		v := re.inputs[focusIdx].Value()
		if v == "" {
			content = dimStyle.Render("(empty)")
		} else {
			content = v
		}
	}
	ls := normalStyle
	indicator := "  "
	if focused {
		ls = selectedStyle
		indicator = "> "
	}
	return "  " + ls.Render(fmt.Sprintf("%-*s", labelWidth, label)) + indicator + content + "\n"
}

func (re *resilienceEditModel) viewToggleField(focusIdx int, label string, value bool) string {
	focused := re.focus == focusIdx
	display := "no"
	if value {
		display = "yes"
	}
	ls := normalStyle
	if focused {
		ls = selectedStyle
		return "  " + ls.Render(fmt.Sprintf("%-*s", labelWidth, label)) + "◀ " + display + " ▶\n"
	}
	return "  " + ls.Render(fmt.Sprintf("%-*s", labelWidth, label)) + "  " + display + "\n"
}
