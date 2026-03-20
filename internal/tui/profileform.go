package tui

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cliswitch/gocc/internal/config"
)

// Focus indices for non-native profile fields.
const (
	fName = iota
	fProtocol // cycling
	fBaseURL
	fAPIKey
	// ── Models ──
	fMainModel // cycling
	fSmallFastModel
	fHaikuModel
	fSonnetModel
	fOpusModel
	fSubagentModel
	// ── Reasoning ──
	fEffortLevel // cycling
	fMaxOutput
	fMaxThinking
	// ── Fallback ──
	fFallback // summary, enter to edit
	// ── Advanced ──
	fHeaders   // summary, enter to edit
	fExtraBody // summary, enter to edit
	fProxies   // summary, enter to edit
	// ── Custom ──
	fClaudeArgs   // summary, enter to edit
	fCustomEnv    // summary, enter to edit
	fInheritArgs  // inline toggle
	fInheritEnv   // inline toggle
	fFieldCount
)

// Input index mapping for non-native profiles (10 text inputs).
var focusInputMap = [fFieldCount]int{
	0,  // fName
	-1, // fProtocol (cycling)
	1,  // fBaseURL
	2,  // fAPIKey
	-1, // fMainModel (cycling)
	3,  // fSmallFastModel
	4,  // fHaikuModel
	5,  // fSonnetModel
	6,  // fOpusModel
	7,  // fSubagentModel
	-1, // fEffortLevel (cycling)
	8,  // fMaxOutput
	9,  // fMaxThinking
	-1, // fFallback (summary)
	-1, // fHeaders (summary)
	-1, // fExtraBody (summary)
	-1, // fProxies (summary)
	-1, // fClaudeArgs (summary)
	-1, // fCustomEnv (summary)
	-1, // fInheritArgs (toggle)
	-1, // fInheritEnv (toggle)
}

var defaultLevels = []string{"", "haiku", "sonnet", "opus"}
var effortLevels = []string{"", "low", "medium", "high", "max"}

// Focus indices for native profile fields.
const (
	nfProxies = iota
	nfClaudeArgs
	nfCustomEnv
	nfInheritArgs
	nfInheritEnv
	nfFieldCount // sentinel — must be last
)

const labelWidth = 20

type profileFormModel struct {
	profile     config.Profile
	origProfile config.Profile
	isNew       bool
	isNative    bool
	callbacks   Callbacks

	inputs     []textinput.Model
	focus      int
	statusMsg  string
	pendingEsc bool
}

func newProfileFormModel(p config.Profile, isNew bool, cb Callbacks) *profileFormModel {
	m := &profileFormModel{
		profile:     config.CloneProfile(p),
		origProfile: config.CloneProfile(p),
		isNew:       isNew,
		isNative:    p.ID == config.NativeProfileID,
		callbacks:   cb,
	}
	m.initInputs()
	return m
}

func (m *profileFormModel) initInputs() {
	if m.isNative {
		// Native profiles: no inline text inputs, just proxies summary
		m.inputs = nil
	} else {
		m.inputs = make([]textinput.Model, 10)
		m.inputs[0] = newTextInput(m.profile.Name)
		m.inputs[1] = newTextInput(m.profile.BaseURL)
		m.inputs[2] = newTextInput(m.profile.APIKey)
		m.inputs[3] = newTextInput(m.profile.Models.SmallFastModel)
		m.inputs[4] = newTextInput(m.profile.Models.HaikuModel)
		m.inputs[5] = newTextInput(m.profile.Models.SonnetModel)
		m.inputs[6] = newTextInput(m.profile.Models.OpusModel)
		m.inputs[7] = newTextInput(m.profile.Models.SubagentModel)
		m.inputs[8] = newTextInput(formatInt(m.profile.Reasoning.MaxOutputTokens))
		m.inputs[9] = newTextInput(formatInt(m.profile.Reasoning.MaxThinkingTokens))
	}
	if len(m.inputs) > 0 {
		m.inputs[0].Focus()
	}
}

func newTextInput(value string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.SetValue(value)
	ti.CharLimit = 256
	ti.Width = 50
	return ti
}

func formatInt(n int) string {
	if n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func (m *profileFormModel) totalFocusable() int {
	if m.isNative {
		return nfFieldCount
	}
	return fFieldCount
}

func (m *profileFormModel) focusToInputIndex() int {
	if m.isNative {
		return -1
	}
	if m.focus >= 0 && m.focus < fFieldCount {
		return focusInputMap[m.focus]
	}
	return -1
}

func (m *profileFormModel) updateFocus() {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	idx := m.focusToInputIndex()
	if idx >= 0 && idx < len(m.inputs) {
		m.inputs[idx].Focus()
	}
}

// populateFromInputs writes the current text-input values into dst.
func (m *profileFormModel) populateFromInputs(dst *config.Profile) {
	if m.isNative || len(m.inputs) == 0 {
		return
	}
	dst.Name = m.inputs[0].Value()
	dst.BaseURL = strings.TrimRight(m.inputs[1].Value(), "/")
	dst.APIKey = m.inputs[2].Value()
	dst.Models.SmallFastModel = m.inputs[3].Value()
	dst.Models.HaikuModel = m.inputs[4].Value()
	dst.Models.SonnetModel = m.inputs[5].Value()
	dst.Models.OpusModel = m.inputs[6].Value()
	dst.Models.SubagentModel = m.inputs[7].Value()
	if v := m.inputs[8].Value(); v != "" {
		n, _ := strconv.Atoi(v)
		dst.Reasoning.MaxOutputTokens = n
	} else {
		dst.Reasoning.MaxOutputTokens = 0
	}
	if v := m.inputs[9].Value(); v != "" {
		n, _ := strconv.Atoi(v)
		dst.Reasoning.MaxThinkingTokens = n
	} else {
		dst.Reasoning.MaxThinkingTokens = 0
	}
}

func (m *profileFormModel) applyToProfile() {
	m.populateFromInputs(&m.profile)
}

func (m *profileFormModel) validate() error {
	if m.isNative {
		return nil
	}
	if v := m.inputs[8].Value(); v != "" {
		if _, err := strconv.Atoi(v); err != nil {
			return fmt.Errorf("invalid max_output_tokens: %s", v)
		}
	}
	if v := m.inputs[9].Value(); v != "" {
		if _, err := strconv.Atoi(v); err != nil {
			return fmt.Errorf("invalid max_thinking_tokens: %s", v)
		}
	}
	return nil
}

func (m *profileFormModel) isDirty() bool {
	o := m.origProfile
	if m.isNative {
		return m.profile.Proxy != o.Proxy ||
			!slices.Equal(m.profile.ClaudeArgs, o.ClaudeArgs) ||
			!maps.Equal(m.profile.CustomEnv, o.CustomEnv) ||
			!boolPtrEqual(m.profile.InheritGlobalArgs, o.InheritGlobalArgs) ||
			!boolPtrEqual(m.profile.InheritGlobalEnv, o.InheritGlobalEnv)
	}
	// Snapshot: shallow-copy m.profile, overlay text-input values.
	snap := m.profile
	m.populateFromInputs(&snap)
	return snap.Name != o.Name ||
		snap.Protocol != o.Protocol ||
		snap.BaseURL != o.BaseURL ||
		snap.APIKey != o.APIKey ||
		snap.Models != o.Models ||
		snap.Reasoning != o.Reasoning ||
		!maps.Equal(snap.CustomHeaders, o.CustomHeaders) ||
		!reflect.DeepEqual(snap.ExtraBody, o.ExtraBody) ||
		!slices.Equal(snap.FallbackChain, o.FallbackChain) ||
		snap.Proxy != o.Proxy ||
		!slices.Equal(snap.ClaudeArgs, o.ClaudeArgs) ||
		!maps.Equal(snap.CustomEnv, o.CustomEnv) ||
		!boolPtrEqual(snap.InheritGlobalArgs, o.InheritGlobalArgs) ||
		!boolPtrEqual(snap.InheritGlobalEnv, o.InheritGlobalEnv)
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}


func isCycleField(focus int) bool {
	return focus == fProtocol || focus == fMainModel || focus == fEffortLevel
}

func isSummaryField(focus int) bool {
	return focus == fFallback || focus == fHeaders || focus == fExtraBody || focus == fProxies ||
		focus == fClaudeArgs || focus == fCustomEnv
}

func isToggleField(focus int) bool {
	return focus == fInheritArgs || focus == fInheritEnv
}

func toggleBool(b *bool) *bool {
	if b == nil || *b {
		f := false
		return &f
	}
	t := true
	return &t
}

func cycleValue(current string, values []string, direction int) string {
	idx := 0
	for i, v := range values {
		if v == current {
			idx = i
			break
		}
	}
	idx = (idx + direction + len(values)) % len(values)
	return values[idx]
}

func (m Model) updateProfileForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	pf := m.profileForm
	if pf == nil {
		m.mode = ModeProfileList
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() != "esc" {
			pf.pendingEsc = false
		}

		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "ctrl+s":
			if err := pf.validate(); err != nil {
				pf.statusMsg = err.Error()
				return m, nil
			}
			pf.applyToProfile()
			if m.callbacks.SaveProfile != nil {
				if err := m.callbacks.SaveProfile(pf.profile, pf.isNew); err != nil {
					pf.statusMsg = fmt.Sprintf("Save failed: %v", err)
					return m, nil
				}
			}
			m.reloadProfiles()
			m.statusMsg = "Profile saved"
			m.mode = ModeProfileList
			m.profileForm = nil
			return m, nil

		case "esc":
			if pf.isDirty() && !pf.pendingEsc {
				pf.pendingEsc = true
				pf.statusMsg = "Unsaved changes. Press Esc again to discard, Ctrl+S to save."
				return m, nil
			}
			m.mode = ModeProfileList
			m.profileForm = nil
			return m, nil

		case "tab", "down":
			pf.focus = (pf.focus + 1) % pf.totalFocusable()
			pf.updateFocus()
			return m, nil

		case "shift+tab", "up":
			pf.focus = (pf.focus - 1 + pf.totalFocusable()) % pf.totalFocusable()
			pf.updateFocus()
			return m, nil

		case "enter", "space":
			if pf.isNative {
				switch pf.focus {
				case nfProxies:
					m.proxyEdit = newProxyEditModel(pf.profile.Proxy)
					m.mode = ModeProxyEdit
					return m, nil
				case nfClaudeArgs:
					pf.applyToProfile()
					m.argsEdit = newArgsEditModel(pf.profile.ClaudeArgs)
					m.mode = ModeArgsEdit
					return m, nil
				case nfCustomEnv:
					pf.applyToProfile()
					m.envEdit = newEnvEditModel(pf.profile.CustomEnv)
					m.mode = ModeEnvEdit
					return m, nil
				case nfInheritArgs:
					pf.profile.InheritGlobalArgs = toggleBool(pf.profile.InheritGlobalArgs)
					return m, nil
				case nfInheritEnv:
					pf.profile.InheritGlobalEnv = toggleBool(pf.profile.InheritGlobalEnv)
					return m, nil
				}
			} else {
				switch pf.focus {
				case fFallback:
					pf.applyToProfile()
					var allProfiles []DisplayProfile
					if m.callbacks.Reload != nil {
						if ps, err := m.callbacks.Reload(); err == nil {
							allProfiles = ps
						}
					}
					m.fallbackEdit = newFallbackEditModel(pf.profile.ID, pf.profile.FallbackChain, allProfiles)
					m.mode = ModeFallbackEdit
					return m, nil
				case fHeaders:
					pf.applyToProfile()
					m.headersEdit = newHeadersEditModel(pf.profile.CustomHeaders)
					m.mode = ModeHeadersEdit
					return m, nil
				case fExtraBody:
					pf.applyToProfile()
					m.extraBodyEdit = newExtraBodyEditModel(pf.profile.ExtraBody)
					m.mode = ModeExtraBodyEdit
					return m, nil
				case fProxies:
					pf.applyToProfile()
					m.proxyEdit = newProxyEditModel(pf.profile.Proxy)
					m.mode = ModeProxyEdit
					return m, nil
				case fClaudeArgs:
					pf.applyToProfile()
					m.argsEdit = newArgsEditModel(pf.profile.ClaudeArgs)
					m.mode = ModeArgsEdit
					return m, nil
				case fCustomEnv:
					pf.applyToProfile()
					m.envEdit = newEnvEditModel(pf.profile.CustomEnv)
					m.mode = ModeEnvEdit
					return m, nil
				case fInheritArgs:
					pf.profile.InheritGlobalArgs = toggleBool(pf.profile.InheritGlobalArgs)
					return m, nil
				case fInheritEnv:
					pf.profile.InheritGlobalEnv = toggleBool(pf.profile.InheritGlobalEnv)
					return m, nil
				}
			}

		case "left", "right":
			if pf.isNative {
				// Handle toggle fields for native profile
				switch pf.focus {
				case nfInheritArgs:
					pf.profile.InheritGlobalArgs = toggleBool(pf.profile.InheritGlobalArgs)
					return m, nil
				case nfInheritEnv:
					pf.profile.InheritGlobalEnv = toggleBool(pf.profile.InheritGlobalEnv)
					return m, nil
				}
				break
			}
			dir := 1
			if msg.String() == "left" {
				dir = -1
			}
			switch pf.focus {
			case fProtocol:
				pf.profile.Protocol = cycleValue(pf.profile.Protocol, config.ValidProtocols, dir)
				return m, nil
			case fMainModel:
				pf.profile.Models.MainModel = cycleValue(pf.profile.Models.MainModel, defaultLevels, dir)
				return m, nil
			case fEffortLevel:
				pf.profile.Reasoning.EffortLevel = cycleValue(pf.profile.Reasoning.EffortLevel, effortLevels, dir)
				return m, nil
			case fInheritArgs:
				pf.profile.InheritGlobalArgs = toggleBool(pf.profile.InheritGlobalArgs)
				return m, nil
			case fInheritEnv:
				pf.profile.InheritGlobalEnv = toggleBool(pf.profile.InheritGlobalEnv)
				return m, nil
			}
		}
	}

	// Update the focused text input
	idx := pf.focusToInputIndex()
	if idx >= 0 && idx < len(pf.inputs) {
		var cmd tea.Cmd
		pf.inputs[idx], cmd = pf.inputs[idx].Update(msg)
		return m, cmd
	}

	return m, nil
}

// ── View ─────────────────────────────────────────

func (m Model) viewProfileForm() string {
	pf := m.profileForm
	if pf == nil {
		return ""
	}

	// Title with profile name
	title := "Edit Profile"
	if pf.isNew {
		title = "New Profile"
	}
	profileName := pf.profile.Name
	if !pf.isNative && len(pf.inputs) > 0 {
		profileName = pf.inputs[0].Value()
	}
	if profileName != "" {
		title += ": " + profileName
	}
	s := titleStyle.Render(title) + "\n\n"

	if pf.isNative {
		s += pf.viewSummaryField(nfProxies, "proxies", proxiesSummary(pf.profile.Proxy))
		s += sectionDivider("Custom")
		s += pf.viewSummaryField(nfClaudeArgs, "claude_args", argsSummary(pf.profile.ClaudeArgs))
		s += pf.viewSummaryField(nfCustomEnv, "custom_env", envSummary(pf.profile.CustomEnv))
		s += pf.viewToggleField(nfInheritArgs, "inherit_global_args", pf.profile.InheritGlobalArgs)
		s += pf.viewToggleField(nfInheritEnv, "inherit_global_env", pf.profile.InheritGlobalEnv)
	} else {
		s += pf.viewInputField(fName, "Name")
		s += pf.viewCycleField(fProtocol, "Protocol", pf.profile.Protocol)
		s += pf.viewInputField(fBaseURL, "Base URL")
		s += pf.viewAPIKeyField()

		s += sectionDivider("Models")
		s += pf.viewCycleField(fMainModel, "main_model", pf.profile.Models.MainModel)
		s += pf.viewInputField(fSmallFastModel, "small_fast_model")
		s += pf.viewInputField(fHaikuModel, "haiku_model")
		s += pf.viewInputField(fSonnetModel, "sonnet_model")
		s += pf.viewInputField(fOpusModel, "opus_model")
		s += pf.viewInputField(fSubagentModel, "subagent_model")

		s += sectionDivider("Reasoning")
		s += pf.viewCycleField(fEffortLevel, "effort_level", pf.profile.Reasoning.EffortLevel)
		s += pf.viewInputField(fMaxOutput, "max_output_tokens")
		s += pf.viewInputField(fMaxThinking, "max_thinking_tokens")

		s += sectionDivider("Fallback")
		s += pf.viewSummaryField(fFallback, "fallback_chain", fallbackSummary(pf.profile.FallbackChain))

		s += sectionDivider("Advanced")
		s += pf.viewSummaryField(fHeaders, "custom_headers", headersSummary(pf.profile.CustomHeaders))
		s += pf.viewSummaryField(fExtraBody, "extra_body", extraBodySummary(pf.profile.ExtraBody))
		s += pf.viewSummaryField(fProxies, "proxies", proxiesSummary(pf.profile.Proxy))

		s += sectionDivider("Custom")
		s += pf.viewSummaryField(fClaudeArgs, "claude_args", argsSummary(pf.profile.ClaudeArgs))
		s += pf.viewSummaryField(fCustomEnv, "custom_env", envSummary(pf.profile.CustomEnv))
		s += pf.viewToggleField(fInheritArgs, "inherit_global_args", pf.profile.InheritGlobalArgs)
		s += pf.viewToggleField(fInheritEnv, "inherit_global_env", pf.profile.InheritGlobalEnv)
	}

	if pf.statusMsg != "" {
		s += "\n" + dimStyle.Render("  "+pf.statusMsg)
	}

	// Hints change based on field type
	hints := []string{"tab/↓ next", "shift+tab/↑ prev"}
	if !pf.isNative && isCycleField(pf.focus) {
		hints = append(hints, "←/→ cycle value")
	}
	if isSummaryField(pf.focus) || (pf.isNative && pf.focus == nfProxies) {
		hints = append(hints, "enter edit")
	}
	if isToggleField(pf.focus) || (pf.isNative && (pf.focus == nfInheritArgs || pf.focus == nfInheritEnv)) {
		hints = append(hints, "enter/space toggle")
	}
	hints = append(hints, "ctrl+s save", "esc cancel")
	s += "\n" + renderStatusBar(hints)
	return s
}

func (pf *profileFormModel) viewInputField(focusIdx int, label string) string {
	inputIdx := focusInputMap[focusIdx]
	focused := pf.focus == focusIdx
	var content string
	if focused {
		content = pf.inputs[inputIdx].View()
	} else {
		v := pf.inputs[inputIdx].Value()
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

func (pf *profileFormModel) viewCycleField(focusIdx int, label, value string) string {
	focused := pf.focus == focusIdx
	display := value
	if display == "" {
		display = "(none)"
	}

	ls := normalStyle
	if focused {
		ls = selectedStyle
		return "  " + ls.Render(fmt.Sprintf("%-*s", labelWidth, label)) + "◀ " + display + " ▶\n"
	}
	return "  " + ls.Render(fmt.Sprintf("%-*s", labelWidth, label)) + "  " + display + "\n"
}

func (pf *profileFormModel) viewAPIKeyField() string {
	focused := pf.focus == fAPIKey
	var content string
	if focused {
		content = pf.inputs[2].View()
	} else {
		content = maskAPIKey(pf.inputs[2].Value())
	}

	ls := normalStyle
	indicator := "  "
	if focused {
		ls = selectedStyle
		indicator = "> "
	}
	return "  " + ls.Render(fmt.Sprintf("%-*s", labelWidth, "API Key")) + indicator + content + "\n"
}

func (pf *profileFormModel) viewSummaryField(focusIdx int, label, summary string) string {
	focused := pf.focus == focusIdx
	ls := normalStyle
	indicator := "  "
	if focused {
		ls = selectedStyle
		indicator = "> "
	}
	return "  " + ls.Render(fmt.Sprintf("%-*s", labelWidth, label)) + indicator + summary + "\n"
}

func (pf *profileFormModel) viewToggleField(focusIdx int, label string, value *bool) string {
	focused := pf.focus == focusIdx
	display := "yes"
	if value != nil && !*value {
		display = "no"
	}

	ls := normalStyle
	if focused {
		ls = selectedStyle
		return "  " + ls.Render(fmt.Sprintf("%-*s", labelWidth, label)) + "◀ " + display + " ▶\n"
	}
	return "  " + ls.Render(fmt.Sprintf("%-*s", labelWidth, label)) + "  " + display + "\n"
}

func sectionDivider(name string) string {
	prefix := "── " + name + " "
	total := 36
	dashes := total - len(prefix)
	if dashes < 3 {
		dashes = 3
	}
	return "  " + dimStyle.Render(prefix+strings.Repeat("─", dashes)) + "\n"
}

func maskAPIKey(key string) string {
	if key == "" {
		return dimStyle.Render("(empty)")
	}
	if len(key) <= 8 {
		return strings.Repeat("•", len(key))
	}
	return key[:3] + strings.Repeat("•", len(key)-8) + key[len(key)-5:]
}

func fallbackSummary(chain []string) string {
	if len(chain) == 0 {
		return dimStyle.Render("(none)")
	}
	return fmt.Sprintf("%d profile(s)", len(chain))
}

func headersSummary(h map[string]string) string {
	if len(h) == 0 {
		return dimStyle.Render("(empty)")
	}
	return fmt.Sprintf("%d header(s)", len(h))
}

func proxiesSummary(p config.Proxy) string {
	parts := []string{}
	if p.HTTPProxy != "" {
		parts = append(parts, "http="+p.HTTPProxy)
	}
	if p.HTTPSProxy != "" {
		parts = append(parts, "https="+p.HTTPSProxy)
	}
	if p.NoProxy != "" {
		parts = append(parts, "no_proxy="+p.NoProxy)
	}
	if len(parts) == 0 {
		return dimStyle.Render("(empty)")
	}
	return strings.Join(parts, ", ")
}
