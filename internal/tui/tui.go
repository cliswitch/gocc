package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cliswitch/gocc/internal/config"
)

// TuiMode represents the current UI mode.
type TuiMode int

const (
	ModeProfileList TuiMode = iota
	ModeProfileEdit
	ModeProfileAdd
	ModeDeleteConfirm
	ModeGlobalConfig
	ModeHeadersEdit
	ModeExtraBodyEdit
	ModeFallbackEdit
	ModeProxyEdit
	ModeEnvEdit
	ModeArgsEdit
	ModeGlobalEnvEdit
)

// DisplayProfile is a summary of a profile for list display.
type DisplayProfile struct {
	ID       string
	Name     string
	Protocol string
}

// Callbacks provides the interface between the TUI and config persistence.
type Callbacks struct {
	Reload        func() ([]DisplayProfile, error)
	GetProfile    func(id string) (config.Profile, bool)
	SaveProfile   func(p config.Profile, isNew bool) error
	DeleteProfile func(id string) error
	MoveProfile   func(id string, delta int) error
	CopyProfile   func(id string) error
	SaveGlobal    func(g config.GlobalConfig) error
	GetGlobal     func() config.GlobalConfig
}

// Model is the top-level bubbletea model for the TUI.
type Model struct {
	profiles  []DisplayProfile
	cursor    int
	chosen    string // selected profile ID (empty until chosen)
	quitting  bool
	mode      TuiMode
	callbacks Callbacks
	statusMsg string

	// Sub-models (initialized on mode switch)
	profileForm  *profileFormModel
	globalConfig *globalConfigModel
	headersEdit  *headersEditModel
	extraBodyEdit *extraBodyEditModel
	fallbackEdit *fallbackEditModel
	proxyEdit    *proxyEditModel
	envEdit      *envEditModel
	argsEdit     *argsEditModel
	deleteTarget *DisplayProfile // for delete confirm
}

// NewModel creates a new TUI model.
func NewModel(profiles []DisplayProfile, initialCursor int, callbacks Callbacks) Model {
	return Model{
		profiles:  profiles,
		cursor:    initialCursor,
		mode:      ModeProfileList,
		callbacks: callbacks,
	}
}

// Chosen returns the selected profile ID, or empty if none chosen.
func (m Model) Chosen() string { return m.chosen }

// returnToProfileForm sets the mode back to the appropriate profile form mode.
func (m *Model) returnToProfileForm() {
	if m.profileForm != nil && m.profileForm.isNew {
		m.mode = ModeProfileAdd
	} else {
		m.mode = ModeProfileEdit
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeProfileList:
		return m.updateProfileList(msg)
	case ModeProfileEdit, ModeProfileAdd:
		return m.updateProfileForm(msg)
	case ModeDeleteConfirm:
		return m.updateDeleteConfirm(msg)
	case ModeGlobalConfig:
		return m.updateGlobalConfig(msg)
	case ModeHeadersEdit:
		return m.updateHeadersEdit(msg)
	case ModeExtraBodyEdit:
		return m.updateExtraBodyEdit(msg)
	case ModeFallbackEdit:
		return m.updateFallbackEdit(msg)
	case ModeProxyEdit:
		return m.updateProxyEdit(msg)
	case ModeEnvEdit:
		return m.updateEnvEdit(msg)
	case ModeArgsEdit:
		return m.updateArgsEdit(msg)
	case ModeGlobalEnvEdit:
		return m.updateGlobalEnvEdit(msg)
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	switch m.mode {
	case ModeProfileList:
		return m.viewProfileList()
	case ModeProfileEdit, ModeProfileAdd:
		return m.viewProfileForm()
	case ModeDeleteConfirm:
		return m.viewDeleteConfirm()
	case ModeGlobalConfig:
		return m.viewGlobalConfig()
	case ModeHeadersEdit:
		return m.viewHeadersEdit()
	case ModeExtraBodyEdit:
		return m.viewExtraBodyEdit()
	case ModeFallbackEdit:
		return m.viewFallbackEdit()
	case ModeProxyEdit:
		return m.viewProxyEdit()
	case ModeEnvEdit, ModeGlobalEnvEdit:
		return m.viewEnvEdit()
	case ModeArgsEdit:
		return m.viewArgsEdit()
	}
	return ""
}
