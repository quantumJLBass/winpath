package tui

import (
	"fmt"
	"strings"
	"time"

	"syspath-optimizer/internal/path"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screen represents the current screen
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenLoading
	ScreenOptimizer
	ScreenOptimizerPreview
	ScreenOptimizerConfirm
	ScreenOptimizerDone
	ScreenPathViewer
	ScreenBackup
	ScreenBackupPreview
	ScreenBackupConfirmRestore
	ScreenBackupConfirmDelete
	ScreenBackupDone
	ScreenJunctions
	ScreenJunctionSuggestions
	ScreenJunctionCreate
	ScreenPathExt
	ScreenPathExtConfirm
	ScreenPathExtDone
	ScreenSettings
	ScreenHotPaths
)

// LoadingTask represents a background task
type LoadingTask int

const (
	TaskNone LoadingTask = iota
	TaskAnalyze
	TaskJunctions
	TaskSuggestions
	TaskCreateJunction
)

// Messages for async operations
type analysisCompleteMsg struct{ result path.AnalysisResult }
type junctionsLoadedMsg struct{ junctions []path.Junction }
type suggestionsLoadedMsg struct{ suggestions []path.JunctionSuggestion }
type junctionCreatedMsg struct {
	success bool
	name    string
	err     error
}
type applyCompleteMsg struct {
	backup *path.BackupInfo
	err    error
}
type progressMsg struct {
	current int
	total   int
	item    string
}
type tickMsg time.Time

// Model is the main application model
type Model struct {
	screen      Screen
	width       int
	height      int
	isAdmin     bool
	err         error
	message     string
	clipboardOK bool

	// Loading
	loadingTask    LoadingTask
	loadingMessage string
	loadingDots    int
	loadingCurrent int
	loadingTotal   int
	loadingItem    string

	// Menu
	menuIndex int
	menuItems []string

	// Optimizer
	analysis       *path.AnalysisResult
	optimizerScope string
	viewMode       int
	scrollOffset   int
	backupInfo     *path.BackupInfo

	// Path Viewer
	viewerScope    string
	viewerExpanded bool

	// Backup
	backups       []path.BackupInfo
	backupIndex   int
	backupPreview *path.Backup

	// Junctions
	junctions         []path.Junction
	suggestions       []path.JunctionSuggestion
	junctionIndex     int
	junctionName      string
	junctionTarget    string
	junctionInputMode int

	// PATHEXT
	pathExtAnalysis *path.PathExtAnalysis
	pathExtOpt      *path.PathExtOptimization
	pathExtEditing  bool
	pathExtList     []string
	pathExtIndex    int

	// Settings
	settingsIndex int
	config        path.Config

	// Hot Paths
	hotPathIndex  int
	hotPathAdding bool
	hotPathInput  string
}

// New creates a new model
func New() Model {
	return Model{
		screen:         ScreenMenu,
		isAdmin:        path.IsAdmin(),
		optimizerScope: "both",
		viewerScope:    "User",
		config:         path.LoadConfig(),
		menuItems: []string{
			"Optimize PATH",
			"View Current PATH",
			"Backup Manager",
			"Junction Manager",
			"PATHEXT Optimizer",
			"Hot Paths Config",
			"Settings",
			"Exit",
		},
	}
}

func (m Model) Init() tea.Cmd { return nil }

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Progress channel for async operations
var progressChan = make(chan progressMsg, 100)

func analyzeCmd() tea.Cmd {
	return func() tea.Msg {
		opts := path.DefaultOptions()
		result := path.AnalyzeAllWithProgress(opts, func(current, total int, item string) {
			select {
			case progressChan <- progressMsg{current: current, total: total, item: item}:
			default:
			}
		})
		return analysisCompleteMsg{result: result}
	}
}

func loadJunctionsCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case progressChan <- progressMsg{item: "Scanning C:\\l for junctions..."}:
		default:
		}
		junctions := path.ListJunctions()
		return junctionsLoadedMsg{junctions: junctions}
	}
}

func loadSuggestionsCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case progressChan <- progressMsg{item: "Analyzing PATH for junction candidates..."}:
		default:
		}
		suggestions := path.SuggestJunctionCandidates()
		return suggestionsLoadedMsg{suggestions: suggestions}
	}
}

func listenForProgress() tea.Cmd {
	return func() tea.Msg {
		select {
		case p := <-progressChan:
			return p
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}
}

func createJunctionCmd(name, target string) tea.Cmd {
	return func() tea.Msg {
		err := path.CreateJunction(name, target)
		return junctionCreatedMsg{success: err == nil, name: name, err: err}
	}
}

func applyOptimizationCmd(analysis *path.AnalysisResult, scope string, isAdmin bool) tea.Cmd {
	return func() tea.Msg {
		// Create backup first
		backup, _ := path.CreateBackup("pre-optimize")

		// Apply changes
		var err error
		if scope == "both" || scope == "user" {
			err = path.SetPath(analysis.User.Optimized.Raw, "User")
		}
		if isAdmin && (scope == "both" || scope == "system") && err == nil {
			err = path.SetPath(analysis.System.Optimized.Raw, "System")
		}

		if err == nil {
			path.BroadcastEnvChange()
		}

		return applyCompleteMsg{backup: backup, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.screen == ScreenLoading {
			m.loadingDots = (m.loadingDots + 1) % 4
			return m, tea.Batch(tickCmd(), listenForProgress())
		}
		return m, nil

	case progressMsg:
		m.loadingCurrent = msg.current
		m.loadingTotal = msg.total
		if msg.item != "" {
			m.loadingItem = msg.item
		}
		return m, nil

	case analysisCompleteMsg:
		m.analysis = &msg.result
		m.screen = ScreenOptimizerPreview
		m.loadingTask = TaskNone
		m.loadingCurrent = 0
		m.loadingTotal = 0
		m.loadingItem = ""
		return m, nil

	case junctionsLoadedMsg:
		m.junctions = msg.junctions
		m.junctionIndex = 0
		m.screen = ScreenJunctions
		m.loadingTask = TaskNone
		m.loadingCurrent = 0
		m.loadingTotal = 0
		m.loadingItem = ""
		return m, nil

	case suggestionsLoadedMsg:
		m.suggestions = msg.suggestions
		m.junctionIndex = 0
		m.screen = ScreenJunctionSuggestions
		m.loadingTask = TaskNone
		m.loadingCurrent = 0
		m.loadingTotal = 0
		m.loadingItem = ""
		return m, nil

	case junctionCreatedMsg:
		m.loadingTask = TaskNone
		m.loadingCurrent = 0
		m.loadingTotal = 0
		m.loadingItem = ""
		if msg.success {
			m.message = "Junction '" + msg.name + "' created!"
			m.screen = ScreenLoading
			m.loadingTask = TaskSuggestions
			m.loadingMessage = "Refreshing suggestions"
			return m, tea.Batch(loadSuggestionsCmd(), tickCmd())
		}
		m.err = msg.err
		if msg.err != nil {
			m.message = "Failed: " + msg.err.Error()
		} else {
			m.message = "Failed to create junction"
		}
		m.screen = ScreenJunctionSuggestions
		return m, nil

	case applyCompleteMsg:
		m.loadingTask = TaskNone
		m.loadingCurrent = 0
		m.loadingTotal = 0
		m.loadingItem = ""
		if msg.err != nil {
			m.err = msg.err
			m.message = "Failed to apply: " + msg.err.Error()
			m.screen = ScreenOptimizerPreview
		} else {
			m.backupInfo = msg.backup
			m.screen = ScreenOptimizerDone
			m.clipboardOK = false
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.screen == ScreenLoading {
			return m, nil
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	switch m.screen {
	case ScreenMenu:
		return m.handleMenuKey(key)
	case ScreenOptimizer, ScreenOptimizerPreview:
		return m.handleOptimizerKey(key)
	case ScreenOptimizerConfirm:
		return m.handleOptimizerConfirmKey(key)
	case ScreenOptimizerDone:
		return m.handleDoneKey(key, ScreenMenu)
	case ScreenPathViewer:
		return m.handleViewerKey(key)
	case ScreenBackup:
		return m.handleBackupKey(key)
	case ScreenBackupPreview:
		return m.handleBackupPreviewKey(key)
	case ScreenBackupConfirmRestore, ScreenBackupConfirmDelete:
		return m.handleBackupConfirmKey(key)
	case ScreenBackupDone:
		return m.handleDoneKey(key, ScreenBackup)
	case ScreenJunctions:
		return m.handleJunctionsKey(key)
	case ScreenJunctionSuggestions:
		return m.handleJunctionSuggestionsKey(key)
	case ScreenJunctionCreate:
		return m.handleJunctionCreateKey(key)
	case ScreenPathExt:
		return m.handlePathExtKey(key)
	case ScreenPathExtConfirm:
		return m.handlePathExtConfirmKey(key)
	case ScreenPathExtDone:
		return m.handleDoneKey(key, ScreenMenu)
	case ScreenSettings:
		return m.handleSettingsKey(key)
	case ScreenHotPaths:
		return m.handleHotPathsKey(key)
	}
	return m, nil
}

func (m Model) handleMenuKey(key string) (Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.menuIndex > 0 {
			m.menuIndex--
		}
	case "down", "j":
		if m.menuIndex < len(m.menuItems)-1 {
			m.menuIndex++
		}
	case "enter":
		return m.selectMenuItem()
	case "q", "esc":
		return m, tea.Quit
	case "1", "2", "3", "4", "5", "6", "7", "8":
		idx := int(key[0] - '1')
		if idx < len(m.menuItems) {
			m.menuIndex = idx
			return m.selectMenuItem()
		}
	}
	return m, nil
}

func (m Model) selectMenuItem() (Model, tea.Cmd) {
	switch m.menuIndex {
	case 0: // Optimize
		m.screen = ScreenLoading
		m.loadingTask = TaskAnalyze
		m.loadingMessage = "Analyzing PATH"
		return m, tea.Batch(analyzeCmd(), tickCmd())
	case 1: // View
		m.screen = ScreenPathViewer
		m.scrollOffset = 0
	case 2: // Backup
		m.screen = ScreenBackup
		m.backups = path.ListBackups()
		m.backupIndex = 0
		m.message = ""
	case 3: // Junctions
		m.screen = ScreenLoading
		m.loadingTask = TaskJunctions
		m.loadingMessage = "Loading junctions"
		return m, tea.Batch(loadJunctionsCmd(), tickCmd())
	case 4: // PATHEXT
		m.screen = ScreenPathExt
		analysis := path.AnalyzePathExt()
		m.pathExtAnalysis = &analysis
		opt := path.OptimizePathExt(true)
		m.pathExtOpt = &opt
	case 5: // Hot Paths
		m.screen = ScreenHotPaths
	case 6: // Settings
		m.screen = ScreenSettings
	case 7: // Exit
		return m, tea.Quit
	}
	return m, nil
}

// setViewMode sets the view mode and resets scroll
func (m Model) setViewMode(mode int) Model {
	m.viewMode = mode
	m.scrollOffset = 0
	return m
}

// cycleScopeMode cycles through optimizer scopes
func (m Model) cycleScopeMode() Model {
	switch m.optimizerScope {
	case "both":
		m.optimizerScope = "system"
	case "system":
		m.optimizerScope = "user"
	default:
		m.optimizerScope = "both"
	}
	return m
}

func (m Model) handleOptimizerKey(key string) (Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.screen = ScreenMenu
		m.analysis = nil
		m.scrollOffset = 0
	case "1", "2", "3", "4":
		mode := int(key[0] - '1')
		m = m.setViewMode(mode)
	case "s", "S":
		m = m.cycleScopeMode()
	case "a", "A":
		m.screen = ScreenOptimizerConfirm
	case "up", "k":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
	case "down", "j":
		m.scrollOffset++
	}
	return m, nil
}

func (m Model) handleOptimizerConfirmKey(key string) (Model, tea.Cmd) {
	switch key {
	case "y", "Y":
		m.screen = ScreenLoading
		m.loadingTask = TaskAnalyze // reuse
		m.loadingMessage = "Applying optimization"
		return m, tea.Batch(applyOptimizationCmd(m.analysis, m.optimizerScope, m.isAdmin), tickCmd())
	case "n", "N", "esc":
		m.screen = ScreenOptimizerPreview
	}
	return m, nil
}

func (m Model) handleDoneKey(key string, backTo Screen) (Model, tea.Cmd) {
	switch key {
	case "c", "C":
		err := path.CopyToClipboard(path.GetRefreshCommand())
		m.clipboardOK = err == nil
	case "esc", "q":
		m.screen = backTo
		m.analysis = nil
		if backTo == ScreenBackup {
			m.backups = path.ListBackups()
		}
	}
	return m, nil
}

func (m Model) handleViewerKey(key string) (Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.screen = ScreenMenu
		m.scrollOffset = 0
	case "s", "S":
		if m.viewerScope == "User" {
			m.viewerScope = "System"
		} else {
			m.viewerScope = "User"
		}
		m.scrollOffset = 0
	case "e", "E":
		m.viewerExpanded = !m.viewerExpanded
	case "up", "k":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
	case "down", "j":
		m.scrollOffset++
	}
	return m, nil
}

// handleBackupCreate creates a manual backup
func (m Model) handleBackupCreate() Model {
	_, err := path.CreateBackup("manual")
	if err != nil {
		m.message = "Backup failed: " + err.Error()
		return m
	}
	m.backups = path.ListBackups()
	m.message = "Backup created!"
	return m
}

// handleBackupView loads and shows backup preview
func (m Model) handleBackupView() Model {
	if len(m.backups) == 0 {
		return m
	}
	if backup, err := path.LoadBackup(m.backups[m.backupIndex].Filename); err == nil {
		m.backupPreview = backup
		m.screen = ScreenBackupPreview
		m.scrollOffset = 0
	}
	return m
}

func (m Model) handleBackupKey(key string) (Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.screen = ScreenMenu
		m.message = ""
	case "up", "k":
		if m.backupIndex > 0 {
			m.backupIndex--
		}
	case "down", "j":
		if m.backupIndex < len(m.backups)-1 {
			m.backupIndex++
		}
	case "c", "C":
		m = m.handleBackupCreate()
	case "v", "V":
		m = m.handleBackupView()
	case "r", "R":
		if len(m.backups) > 0 {
			m.screen = ScreenBackupConfirmRestore
		}
	case "d", "D":
		if len(m.backups) > 0 {
			m.screen = ScreenBackupConfirmDelete
		}
	}
	return m, nil
}

func (m Model) handleBackupPreviewKey(key string) (Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.screen = ScreenBackup
		m.backupPreview = nil
		m.scrollOffset = 0
	case "up", "k":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
	case "down", "j":
		m.scrollOffset++
	}
	return m, nil
}

func (m Model) handleBackupConfirmKey(key string) (Model, tea.Cmd) {
	switch key {
	case "y", "Y":
		if m.screen == ScreenBackupConfirmRestore {
			if err := path.RestoreBackup(m.backups[m.backupIndex].Filename, m.isAdmin); err != nil {
				m.err = err
			} else {
				m.message = "Backup restored successfully!"
				m.screen = ScreenBackupDone
				m.clipboardOK = false
			}
		} else {
			if err := path.DeleteBackup(m.backups[m.backupIndex].Filename); err != nil {
				m.message = "Delete failed: " + err.Error()
			} else {
				m.message = "Backup deleted"
			}
			m.backups = path.ListBackups()
			m.screen = ScreenBackup
			if m.backupIndex >= len(m.backups) && m.backupIndex > 0 {
				m.backupIndex--
			}
		}
	case "n", "N", "esc":
		m.screen = ScreenBackup
	}
	return m, nil
}

// handleJunctionsRefresh starts junction refresh
func (m Model) handleJunctionsRefresh() (Model, tea.Cmd) {
	m.screen = ScreenLoading
	m.loadingTask = TaskJunctions
	m.loadingMessage = "Refreshing junctions"
	return m, tea.Batch(loadJunctionsCmd(), tickCmd())
}

// handleJunctionsSuggestions starts suggestion loading
func (m Model) handleJunctionsSuggestions() (Model, tea.Cmd) {
	m.screen = ScreenLoading
	m.loadingTask = TaskSuggestions
	m.loadingMessage = "Analyzing PATH for suggestions"
	m.message = ""
	return m, tea.Batch(loadSuggestionsCmd(), tickCmd())
}

// handleJunctionsCreate opens junction create screen
func (m Model) handleJunctionsCreate() Model {
	m.screen = ScreenJunctionCreate
	m.junctionName = ""
	m.junctionTarget = ""
	m.junctionInputMode = 0
	m.message = ""
	m.err = nil
	return m
}

// handleJunctionsDelete deletes the selected junction
func (m Model) handleJunctionsDelete() Model {
	if len(m.junctions) == 0 {
		return m
	}
	if err := path.RemoveJunction(m.junctions[m.junctionIndex].Name); err != nil {
		m.message = "Delete failed: " + err.Error()
	} else {
		m.message = "Junction deleted"
	}
	m.junctions = path.ListJunctions()
	if m.junctionIndex >= len(m.junctions) && m.junctionIndex > 0 {
		m.junctionIndex--
	}
	return m
}

func (m Model) handleJunctionsKey(key string) (Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.screen = ScreenMenu
		m.message = ""
	case "1":
		return m.handleJunctionsRefresh()
	case "2":
		return m.handleJunctionsSuggestions()
	case "3":
		m = m.handleJunctionsCreate()
	case "d", "D":
		m = m.handleJunctionsDelete()
	case "up", "k":
		if m.junctionIndex > 0 {
			m.junctionIndex--
		}
	case "down", "j":
		if m.junctionIndex < len(m.junctions)-1 {
			m.junctionIndex++
		}
	}
	return m, nil
}

func (m Model) handleJunctionSuggestionsKey(key string) (Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.screen = ScreenJunctions
		m.junctions = path.ListJunctions()
		m.junctionIndex = 0
		m.message = ""
	case "c", "C":
		if len(m.suggestions) > 0 && m.junctionIndex < len(m.suggestions) {
			s := m.suggestions[m.junctionIndex]
			m.screen = ScreenLoading
			m.loadingTask = TaskCreateJunction
			m.loadingMessage = "Creating junction '" + s.SuggestedName + "'"
			return m, tea.Batch(createJunctionCmd(s.SuggestedName, s.OriginalPath), tickCmd())
		}
	case "up", "k":
		if m.junctionIndex > 0 {
			m.junctionIndex--
		}
	case "down", "j":
		if m.junctionIndex < len(m.suggestions)-1 {
			m.junctionIndex++
		}
	}
	return m, nil
}

// handleJunctionCreateEscape handles escape key in junction create
func (m Model) handleJunctionCreateEscape() Model {
	m.screen = ScreenJunctions
	m.junctions = path.ListJunctions()
	m.err = nil
	return m
}

// handleJunctionCreateEnter handles enter key in junction create
func (m Model) handleJunctionCreateEnter() Model {
	if m.junctionInputMode == 0 && m.junctionName != "" {
		m.junctionInputMode = 1
		return m
	}
	if m.junctionInputMode == 1 && m.junctionTarget != "" && m.junctionName != "" {
		if err := path.CreateJunction(m.junctionName, m.junctionTarget); err != nil {
			m.err = err
			m.message = "Failed: " + err.Error()
		} else {
			m.message = "Junction '" + m.junctionName + "' created!"
			m.screen = ScreenJunctions
			m.junctions = path.ListJunctions()
			m.err = nil
		}
	}
	return m
}

// handleJunctionCreateBackspace handles backspace in junction create
func (m Model) handleJunctionCreateBackspace() Model {
	if m.junctionInputMode == 0 && len(m.junctionName) > 0 {
		m.junctionName = m.junctionName[:len(m.junctionName)-1]
	} else if m.junctionInputMode == 1 && len(m.junctionTarget) > 0 {
		m.junctionTarget = m.junctionTarget[:len(m.junctionTarget)-1]
	}
	return m
}

// handleJunctionCreateChar handles character input in junction create
func (m Model) handleJunctionCreateChar(key string) Model {
	if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
		if m.junctionInputMode == 0 {
			m.junctionName += key
		} else {
			m.junctionTarget += key
		}
	}
	return m
}

func (m Model) handleJunctionCreateKey(key string) (Model, tea.Cmd) {
	switch key {
	case "esc":
		m = m.handleJunctionCreateEscape()
	case "tab":
		m.junctionInputMode = (m.junctionInputMode + 1) % 2
	case "enter":
		m = m.handleJunctionCreateEnter()
	case "backspace":
		m = m.handleJunctionCreateBackspace()
	default:
		m = m.handleJunctionCreateChar(key)
	}
	return m, nil
}

// handlePathExtEditKey handles keys when in PATHEXT edit mode
func (m Model) handlePathExtEditKey(key string) Model {
	switch key {
	case "esc":
		m.pathExtEditing = false
	case "up", "k":
		if m.pathExtIndex > 0 {
			m.pathExtIndex--
		}
	case "down", "j":
		if m.pathExtIndex < len(m.pathExtList)-1 {
			m.pathExtIndex++
		}
	case "K", "U", "u":
		m = m.movePathExtUp()
	case "J", "D", "d":
		m = m.movePathExtDown()
	case "x", "X", "delete":
		m = m.removePathExtEntry()
	case "a", "A":
		m.pathExtEditing = false
		m.screen = ScreenPathExtConfirm
	}
	return m
}

// movePathExtUp moves current extension up in priority
func (m Model) movePathExtUp() Model {
	if m.pathExtIndex > 0 {
		m.pathExtList[m.pathExtIndex], m.pathExtList[m.pathExtIndex-1] = m.pathExtList[m.pathExtIndex-1], m.pathExtList[m.pathExtIndex]
		m.pathExtIndex--
		m.updatePathExtOpt()
	}
	return m
}

// movePathExtDown moves current extension down in priority
func (m Model) movePathExtDown() Model {
	if m.pathExtIndex < len(m.pathExtList)-1 {
		m.pathExtList[m.pathExtIndex], m.pathExtList[m.pathExtIndex+1] = m.pathExtList[m.pathExtIndex+1], m.pathExtList[m.pathExtIndex]
		m.pathExtIndex++
		m.updatePathExtOpt()
	}
	return m
}

// removePathExtEntry removes the current extension from the list
func (m Model) removePathExtEntry() Model {
	if len(m.pathExtList) > 1 {
		m.pathExtList = append(m.pathExtList[:m.pathExtIndex], m.pathExtList[m.pathExtIndex+1:]...)
		if m.pathExtIndex >= len(m.pathExtList) {
			m.pathExtIndex = len(m.pathExtList) - 1
		}
		m.updatePathExtOpt()
	}
	return m
}

// handlePathExtNormalKey handles keys in normal PATHEXT view mode
func (m Model) handlePathExtNormalKey(key string) Model {
	switch key {
	case "esc", "q":
		m.screen = ScreenMenu
	case "e", "E":
		m.pathExtEditing = true
		m.pathExtList = make([]string, len(m.pathExtAnalysis.Current))
		copy(m.pathExtList, m.pathExtAnalysis.Current)
		m.pathExtIndex = 0
	case "o", "O":
		if m.pathExtOpt != nil && m.pathExtOpt.Changed {
			m.pathExtList = make([]string, len(m.pathExtOpt.Optimized))
			copy(m.pathExtList, m.pathExtOpt.Optimized)
			m.pathExtEditing = true
			m.pathExtIndex = 0
		}
	case "a", "A":
		if m.pathExtOpt != nil && m.pathExtOpt.Changed {
			m.screen = ScreenPathExtConfirm
		}
	}
	return m
}

func (m Model) handlePathExtKey(key string) (Model, tea.Cmd) {
	if m.pathExtEditing {
		return m.handlePathExtEditKey(key), nil
	}
	return m.handlePathExtNormalKey(key), nil
}

func (m *Model) updatePathExtOpt() {
	optimizedStr := strings.Join(m.pathExtList, ";")
	original := strings.Join(m.pathExtAnalysis.Current, ";")
	m.pathExtOpt = &path.PathExtOptimization{
		Original:        original,
		Optimized:       m.pathExtList,
		OptimizedString: optimizedStr,
		Changed:         optimizedStr != original,
	}
}

func (m Model) handlePathExtConfirmKey(key string) (Model, tea.Cmd) {
	switch key {
	case "y", "Y":
		if m.pathExtOpt == nil {
			m.err = fmt.Errorf("no optimization to apply")
			m.screen = ScreenPathExt
			return m, nil
		}
		scope := "User"
		if m.isAdmin {
			scope = "System"
		}
		if err := path.ApplyPathExt(m.pathExtOpt.OptimizedString, scope); err != nil {
			m.err = err
		} else {
			m.screen = ScreenPathExtDone
			m.clipboardOK = false
		}
	case "n", "N", "esc":
		m.screen = ScreenPathExt
	}
	return m, nil
}

func (m Model) handleSettingsKey(key string) (Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.screen = ScreenMenu
	case "up", "k":
		if m.settingsIndex > 0 {
			m.settingsIndex--
		}
	case "down", "j":
		if m.settingsIndex < 2 {
			m.settingsIndex++
		}
	case "enter", "+", "-":
		switch m.settingsIndex {
		case 0:
			if key == "+" || key == "enter" {
				m.config.MaxBackups++
			} else if m.config.MaxBackups > 1 {
				m.config.MaxBackups--
			}
		case 1:
			m.config.AutoBackup = !m.config.AutoBackup
		}
		_ = path.SaveConfig(m.config) // Intentionally ignore error for UI config
	}
	return m, nil
}

// handleHotPathsInputKey handles keys when in hot path input mode
func (m Model) handleHotPathsInputKey(key string) Model {
	switch key {
	case "esc":
		m.hotPathAdding = false
		m.hotPathInput = ""
	case "enter":
		if m.hotPathInput != "" {
			m.config.HotPaths = append(m.config.HotPaths, m.hotPathInput)
			_ = path.SaveConfig(m.config) // Intentionally ignore error for UI config
			m.hotPathInput = ""
			m.hotPathAdding = false
			m.message = "Path added!"
		}
	case "backspace":
		if len(m.hotPathInput) > 0 {
			m.hotPathInput = m.hotPathInput[:len(m.hotPathInput)-1]
		}
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
			m.hotPathInput += key
		}
	}
	return m
}

// handleHotPathsNavKey handles navigation and action keys for hot paths
func (m Model) handleHotPathsNavKey(key string) Model {
	switch key {
	case "esc", "q":
		m.screen = ScreenMenu
		m.message = ""
	case "a", "A":
		m.hotPathAdding = true
		m.hotPathInput = ""
	case "d", "x", "X":
		m = m.deleteCurrentHotPath()
	case "up", "k":
		if m.hotPathIndex > 0 {
			m.hotPathIndex--
		}
	case "down", "j":
		if m.hotPathIndex < len(m.config.HotPaths)-1 {
			m.hotPathIndex++
		}
	case "K", "U":
		m = m.moveHotPathUp()
	case "J", "D":
		m = m.moveHotPathDown()
	}
	return m
}

// deleteCurrentHotPath removes the currently selected hot path
func (m Model) deleteCurrentHotPath() Model {
	if len(m.config.HotPaths) > 0 && m.hotPathIndex < len(m.config.HotPaths) {
		m.config.HotPaths = append(m.config.HotPaths[:m.hotPathIndex], m.config.HotPaths[m.hotPathIndex+1:]...)
		_ = path.SaveConfig(m.config) // Intentionally ignore error for UI config
		if m.hotPathIndex >= len(m.config.HotPaths) && m.hotPathIndex > 0 {
			m.hotPathIndex--
		}
		m.message = "Path removed"
	}
	return m
}

// moveHotPathUp moves current hot path up in priority
func (m Model) moveHotPathUp() Model {
	if m.hotPathIndex > 0 {
		m.config.HotPaths[m.hotPathIndex], m.config.HotPaths[m.hotPathIndex-1] = m.config.HotPaths[m.hotPathIndex-1], m.config.HotPaths[m.hotPathIndex]
		m.hotPathIndex--
		_ = path.SaveConfig(m.config) // Intentionally ignore error for UI config
	}
	return m
}

// moveHotPathDown moves current hot path down in priority
func (m Model) moveHotPathDown() Model {
	if m.hotPathIndex < len(m.config.HotPaths)-1 {
		m.config.HotPaths[m.hotPathIndex], m.config.HotPaths[m.hotPathIndex+1] = m.config.HotPaths[m.hotPathIndex+1], m.config.HotPaths[m.hotPathIndex]
		m.hotPathIndex++
		_ = path.SaveConfig(m.config) // Intentionally ignore error for UI config
	}
	return m
}

func (m Model) handleHotPathsKey(key string) (Model, tea.Cmd) {
	if m.hotPathAdding {
		return m.handleHotPathsInputKey(key), nil
	}
	return m.handleHotPathsNavKey(key), nil
}

// View renders the UI
func (m Model) View() string {
	switch m.screen {
	case ScreenLoading:
		return m.viewLoading()
	case ScreenMenu:
		return m.viewMenu()
	case ScreenOptimizer, ScreenOptimizerPreview:
		return m.viewOptimizer()
	case ScreenOptimizerConfirm:
		return m.viewConfirm("Apply PATH Optimization?", "Scope: "+m.optimizerScope, ScreenOptimizerPreview)
	case ScreenOptimizerDone:
		return m.viewDone("PATH optimization applied successfully!", m.backupInfo)
	case ScreenPathViewer:
		return m.viewPathViewer()
	case ScreenBackup:
		return m.viewBackup()
	case ScreenBackupPreview:
		return m.viewBackupPreview()
	case ScreenBackupConfirmRestore:
		return m.viewConfirmBackup("Restore", Yellow)
	case ScreenBackupConfirmDelete:
		return m.viewConfirmBackup("Delete", Red)
	case ScreenBackupDone:
		return m.viewDone(m.message, nil)
	case ScreenJunctions:
		return m.viewJunctions()
	case ScreenJunctionSuggestions:
		return m.viewJunctionSuggestions()
	case ScreenJunctionCreate:
		return m.viewJunctionCreate()
	case ScreenPathExt:
		return m.viewPathExt()
	case ScreenPathExtConfirm:
		scope := "User"
		if m.isAdmin {
			scope = "System"
		}
		return m.viewConfirm("Apply PATHEXT Optimization?", "Scope: "+scope, ScreenPathExt)
	case ScreenPathExtDone:
		return m.viewDone("PATHEXT optimized successfully!", nil)
	case ScreenSettings:
		return m.viewSettings()
	case ScreenHotPaths:
		return m.viewHotPaths()
	}
	return ""
}

func (m Model) viewLoading() string {
	dots := strings.Repeat(".", m.loadingDots)
	padding := strings.Repeat(" ", 3-m.loadingDots)
	spinner := SelectedStyle.Render("[") + SuccessStyle.Render(dots) + padding + SelectedStyle.Render("]")

	var b strings.Builder
	b.WriteString("\n\n  " + spinner + "  " + TitleStyle.Render(m.loadingMessage) + "\n\n")

	// Show progress if available
	if m.loadingTotal > 0 {
		// Progress bar
		barWidth := 40
		progress := float64(m.loadingCurrent) / float64(m.loadingTotal)
		filled := int(progress * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		bar := SuccessStyle.Render(strings.Repeat("=", filled)) + DimStyle.Render(strings.Repeat("-", barWidth-filled))
		b.WriteString("  [" + bar + "] " + DimStyle.Render(fmt.Sprintf("%d/%d", m.loadingCurrent, m.loadingTotal)) + "\n\n")
	}

	// Show current item being processed
	if m.loadingItem != "" {
		item := m.loadingItem
		if len(item) > 60 {
			item = "..." + item[len(item)-57:]
		}
		b.WriteString("  " + DimStyle.Render(item) + "\n")
	} else {
		b.WriteString("  " + DimStyle.Render("Please wait...") + "\n")
	}

	return b.String()
}

func (m Model) viewMenu() string {
	var b strings.Builder
	title := TitleStyle.Render("Windows PATH Optimizer")
	if m.isAdmin {
		title += SuccessStyle.Render(" [Admin]")
	}
	b.WriteString(title + "\n\n")

	for i, item := range m.menuItems {
		cursor := "  "
		style := NormalStyle
		if i == m.menuIndex {
			cursor = SelectedStyle.Render("> ")
			style = SelectedStyle
		}
		b.WriteString(cursor + DimStyle.Render(fmt.Sprintf("[%d] ", i+1)) + style.Render(item) + "\n")
	}

	b.WriteString("\n" + FooterStyle.Render("Use arrows or numbers, Enter to select, Q to quit"))
	return b.String()
}

func (m Model) viewOptimizer() string {
	if m.analysis == nil {
		return TitleStyle.Render("Analyzing...") + "\n"
	}

	var b strings.Builder
	b.WriteString(TitleStyle.Render("PATH Optimization Preview") + "\n")

	// Simple tab bar without boxes
	tabs := []string{"Summary", "Changes", "Raw", "List"}
	var tabLine string
	for i, tab := range tabs {
		if i == m.viewMode {
			tabLine += SelectedStyle.Render(fmt.Sprintf("[%d] %s", i+1, tab)) + "  "
		} else {
			tabLine += DimStyle.Render(fmt.Sprintf("[%d] %s", i+1, tab)) + "  "
		}
	}
	b.WriteString(tabLine + "\n\n")

	switch m.viewMode {
	case 0:
		b.WriteString(m.renderSummary())
	case 1:
		b.WriteString(m.renderChanges())
	case 2:
		b.WriteString(m.renderRaw())
	case 3:
		b.WriteString(m.renderList())
	}

	b.WriteString("\n" + RenderKey("1-4", "Tab") + "  " + RenderKey("A", "Apply") + "  " + RenderKey("S", "Scope: "+m.optimizerScope) + "  " + RenderKey("Esc", "Menu"))
	return b.String()
}

func (m Model) renderSummary() string {
	var b strings.Builder
	sys := m.analysis.System
	usr := m.analysis.User

	sysColor := Cyan
	if !m.isAdmin {
		sysColor = Gray
	}
	sysStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(sysColor).Padding(0, 1)

	sysContent := TitleStyle.Render("System PATH") + "\n"
	sysContent += RenderMetric("Entries", sys.Original.Count, sys.Optimized.Count, "") + "\n"
	sysContent += RenderMetric("Length", sys.Original.Length, sys.Optimized.Length, " chars") + "\n"
	sysContent += DimStyle.Render(fmt.Sprintf("Dup: %d  Dead: %d  Short: %d  Vars: %d",
		sys.Metrics.DuplicatesRemoved, sys.Metrics.DeadPathsRemoved,
		sys.Metrics.PathsShortened, sys.Metrics.VarsSubstituted)) + "\n"
	sysContent += SuccessStyle.Render(fmt.Sprintf("Saved: %.1f%%", sys.Metrics.PercentageSaved))
	if !m.isAdmin {
		sysContent += "\n" + WarningStyle.Render("(Read-only - needs admin)")
	}
	b.WriteString(sysStyle.Render(sysContent) + "\n\n")

	usrStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Cyan).Padding(0, 1)
	usrContent := TitleStyle.Render("User PATH") + "\n"
	usrContent += RenderMetric("Entries", usr.Original.Count, usr.Optimized.Count, "") + "\n"
	usrContent += RenderMetric("Length", usr.Original.Length, usr.Optimized.Length, " chars") + "\n"
	usrContent += DimStyle.Render(fmt.Sprintf("Dup: %d  Dead: %d  Short: %d  Vars: %d",
		usr.Metrics.DuplicatesRemoved, usr.Metrics.DeadPathsRemoved,
		usr.Metrics.PathsShortened, usr.Metrics.VarsSubstituted)) + "\n"
	usrContent += SuccessStyle.Render(fmt.Sprintf("Saved: %.1f%%", usr.Metrics.PercentageSaved))
	b.WriteString(usrStyle.Render(usrContent))

	if len(m.analysis.CustomVariables) > 0 {
		b.WriteString("\n\n")
		customStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Magenta).Padding(0, 1)
		customContent := InfoStyle.Render("Custom PATH Variables Detected") + "\n"
		for _, v := range m.analysis.CustomVariables {
			customContent += DimStyle.Render(fmt.Sprintf("  %%%s%% (in %s)", v.Name, v.FoundIn)) + "\n"
		}
		b.WriteString(customStyle.Render(strings.TrimSuffix(customContent, "\n")))
	}

	return b.String()
}

func (m Model) renderChanges() string {
	var b strings.Builder
	var allChanges []string

	addChanges := func(changes []path.PathChange, scope string) {
		for _, c := range changes {
			var line string
			switch c.Type {
			case "duplicate":
				line = WarningStyle.Render("[DUP]") + " " + DimStyle.Render(c.Original)
			case "dead":
				line = ErrorStyle.Render("[DEAD]") + " " + DimStyle.Render(c.Original)
			case "shortened":
				line = SuccessStyle.Render("[8.3]") + " " + DimStyle.Render(c.Original) + "\n        -> " + NormalStyle.Render(c.New)
			case "variable":
				line = SuccessStyle.Render("[VAR]") + " " + DimStyle.Render(c.Original) + "\n        -> " + NormalStyle.Render(c.New)
			}
			allChanges = append(allChanges, SubtitleStyle.Render("["+scope+"]")+" "+line)
		}
	}

	addChanges(m.analysis.System.Changes, "SYS")
	addChanges(m.analysis.User.Changes, "USR")

	if len(allChanges) == 0 {
		return DimStyle.Render("No changes - PATH is already optimized!")
	}

	maxVisible := 12
	start := m.scrollOffset
	if start > len(allChanges)-maxVisible {
		start = len(allChanges) - maxVisible
	}
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(allChanges) {
		end = len(allChanges)
	}

	if start > 0 {
		b.WriteString(DimStyle.Render(fmt.Sprintf("     ... %d more above\n", start)))
	}
	for _, line := range allChanges[start:end] {
		b.WriteString(line + "\n")
	}
	if end < len(allChanges) {
		b.WriteString(DimStyle.Render(fmt.Sprintf("     ... %d more below\n", len(allChanges)-end)))
	}
	b.WriteString(DimStyle.Render(fmt.Sprintf("\n%d total changes", len(allChanges))))

	return b.String()
}

func (m Model) renderRaw() string {
	var data *path.OptimizeResult
	var label string

	switch m.optimizerScope {
	case "system":
		data = &m.analysis.System
		label = "System"
	default:
		data = &m.analysis.User
		label = "User"
	}

	var b strings.Builder
	b.WriteString(SubtitleStyle.Render(fmt.Sprintf("Optimized %s PATH:", label)) + "\n")
	b.WriteString(DimStyle.Render(fmt.Sprintf("Length: %d chars (was %d)", data.Optimized.Length, data.Original.Length)) + "\n\n")

	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Gray).Padding(0, 1)
	b.WriteString(boxStyle.Render(wrapText(data.Optimized.Raw, 72)))
	b.WriteString("\n\n" + DimStyle.Render("Press [S] to switch scope (showing: "+label+")"))

	return b.String()
}

func (m Model) renderList() string {
	var data *path.OptimizeResult
	var label string

	switch m.optimizerScope {
	case "system":
		data = &m.analysis.System
		label = "System"
	default:
		data = &m.analysis.User
		label = "User"
	}

	var b strings.Builder
	entries := data.Optimized.Entries
	b.WriteString(SubtitleStyle.Render(fmt.Sprintf("Optimized %s (%d entries):", label, len(entries))) + "\n\n")

	maxVisible := 16
	start := m.scrollOffset
	if start > len(entries)-maxVisible {
		start = len(entries) - maxVisible
	}
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(entries) {
		end = len(entries)
	}

	if start > 0 {
		b.WriteString(DimStyle.Render(fmt.Sprintf("     ... %d above\n", start)))
	}
	for i := start; i < end; i++ {
		entry := entries[i]
		if len(entry) > 64 {
			entry = entry[:61] + "..."
		}
		b.WriteString(DimStyle.Render(fmt.Sprintf("%3d. ", i+1)) + NormalStyle.Render(entry) + "\n")
	}
	if end < len(entries) {
		b.WriteString(DimStyle.Render(fmt.Sprintf("     ... %d below\n", len(entries)-end)))
	}

	return b.String()
}

func (m Model) viewConfirm(title, detail string, _ Screen) string {
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Yellow).Padding(1, 2)
	content := WarningStyle.Render(title) + "\n\n" + detail + "\n\n" + RenderKey("Y", "Yes") + "  " + RenderKey("N", "No")
	return boxStyle.Render(content)
}

func (m Model) viewDone(msg string, backup *path.BackupInfo) string {
	var b strings.Builder
	b.WriteString(SuccessStyle.Render(msg) + "\n\n")

	if backup != nil {
		b.WriteString(DimStyle.Render("Backup: ") + NormalStyle.Render(backup.Filename) + "\n\n")
	}

	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Yellow).Padding(0, 1)
	content := WarningStyle.Render("To refresh your terminal:") + "\n\n"
	content += DimStyle.Render("PowerShell:") + "\n" + NormalStyle.Render(path.GetRefreshCommand()) + "\n\n"
	content += DimStyle.Render("CMD:") + "\n" + NormalStyle.Render("refreshenv")
	b.WriteString(boxStyle.Render(content) + "\n\n")

	if m.clipboardOK {
		b.WriteString(SuccessStyle.Render("Copied to clipboard!") + "\n")
	} else {
		b.WriteString(RenderKey("C", "Copy to clipboard") + "\n")
	}
	b.WriteString(RenderKey("Esc", "Back"))
	return b.String()
}

func (m Model) viewPathViewer() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Current PATH") + " " + SelectedStyle.Render("["+m.viewerScope+"]") + "\n\n")

	var pathStr string
	if m.viewerExpanded {
		pathStr, _ = path.GetPathExpanded(m.viewerScope)
	} else {
		pathStr, _ = path.GetPathRaw(m.viewerScope)
	}

	entries := path.ParsePath(pathStr)
	maxVisible := 18
	start := m.scrollOffset
	if start > len(entries)-maxVisible {
		start = len(entries) - maxVisible
	}
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(entries) {
		end = len(entries)
	}

	if start > 0 {
		b.WriteString(DimStyle.Render(fmt.Sprintf("      ... %d above\n", start)))
	}
	for i := start; i < end; i++ {
		entry := entries[i]
		exists := path.PathExists(entry)
		marker := SuccessStyle.Render("*")
		if !exists && !strings.Contains(entry, "%") {
			marker = ErrorStyle.Render("!")
		}
		displayEntry := entry
		if len(displayEntry) > 64 {
			displayEntry = displayEntry[:61] + "..."
		}
		b.WriteString(fmt.Sprintf("%s %s %s\n", DimStyle.Render(fmt.Sprintf("%3d.", i+1)), marker, NormalStyle.Render(displayEntry)))
	}
	if end < len(entries) {
		b.WriteString(DimStyle.Render(fmt.Sprintf("      ... %d below\n", len(entries)-end)))
	}

	b.WriteString(DimStyle.Render(fmt.Sprintf("\n%d entries, %d chars", len(entries), len(pathStr))))

	expandLabel := "expanded"
	if m.viewerExpanded {
		expandLabel = "raw"
	}
	b.WriteString("\n\n" + RenderKey("S", "Switch scope") + "  " + RenderKey("E", "Show "+expandLabel) + "  " + RenderKey("Esc", "Menu"))
	return b.String()
}

func (m Model) viewBackup() string {
	var b strings.Builder
	config := path.LoadConfig()
	b.WriteString(TitleStyle.Render("Backup Manager") + " " + DimStyle.Render(fmt.Sprintf("(%d/%d)", len(m.backups), config.MaxBackups)) + "\n\n")

	if m.message != "" {
		b.WriteString(SuccessStyle.Render(m.message) + "\n\n")
	}

	if len(m.backups) == 0 {
		b.WriteString(DimStyle.Render("No backups found.") + "\n\n")
	} else {
		boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Gray).Padding(0, 1)
		var content string
		for i, backup := range m.backups {
			cursor := "  "
			style := NormalStyle
			if i == m.backupIndex {
				cursor = SelectedStyle.Render("> ")
				style = SelectedStyle
			}
			content += cursor + style.Render(fmt.Sprintf("%s [%s]", backup.FormattedDate, backup.Suffix)) + "\n"
		}
		b.WriteString(boxStyle.Render(strings.TrimSuffix(content, "\n")) + "\n\n")
	}

	b.WriteString(RenderKey("C", "Create") + "  ")
	if len(m.backups) > 0 {
		b.WriteString(RenderKey("V", "Preview") + "  " + RenderKey("R", "Restore") + "  " + RenderKey("D", "Delete") + "  ")
	}
	b.WriteString(RenderKey("Esc", "Menu"))
	return b.String()
}

func (m Model) viewBackupPreview() string {
	var b strings.Builder
	if m.backupPreview == nil {
		return "Loading..."
	}

	b.WriteString(TitleStyle.Render("Backup Preview") + "\n")
	b.WriteString(DimStyle.Render(fmt.Sprintf("Created: %s  Host: %s", m.backupPreview.Timestamp.Format("2006-01-02 15:04:05"), m.backupPreview.Hostname)) + "\n\n")

	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Gray).Padding(0, 1)

	sysContent := SubtitleStyle.Render(fmt.Sprintf("System PATH (%d)", len(m.backupPreview.SystemPath.Entries))) + "\n"
	for i, e := range m.backupPreview.SystemPath.Entries {
		if i >= 5 {
			sysContent += DimStyle.Render(fmt.Sprintf("  +%d more", len(m.backupPreview.SystemPath.Entries)-5)) + "\n"
			break
		}
		if len(e) > 58 {
			e = e[:55] + "..."
		}
		sysContent += DimStyle.Render("  "+e) + "\n"
	}
	b.WriteString(boxStyle.Render(strings.TrimSuffix(sysContent, "\n")) + "\n\n")

	usrContent := SubtitleStyle.Render(fmt.Sprintf("User PATH (%d)", len(m.backupPreview.UserPath.Entries))) + "\n"
	for i, e := range m.backupPreview.UserPath.Entries {
		if i >= 5 {
			usrContent += DimStyle.Render(fmt.Sprintf("  +%d more", len(m.backupPreview.UserPath.Entries)-5)) + "\n"
			break
		}
		if len(e) > 58 {
			e = e[:55] + "..."
		}
		usrContent += DimStyle.Render("  "+e) + "\n"
	}
	b.WriteString(boxStyle.Render(strings.TrimSuffix(usrContent, "\n")) + "\n\n")

	b.WriteString(RenderKey("Esc", "Back"))
	return b.String()
}

func (m Model) viewConfirmBackup(action string, color lipgloss.Color) string {
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(color).Padding(1, 2)
	style := lipgloss.NewStyle().Foreground(color)
	content := style.Render(action+" this backup?") + "\n\n"
	content += NormalStyle.Render(m.backups[m.backupIndex].Filename) + "\n\n"
	if action == "Restore" {
		content += DimStyle.Render("Current PATH will be backed up first.") + "\n\n"
	} else {
		content += ErrorStyle.Render("This cannot be undone!") + "\n\n"
	}
	content += RenderKey("Y", "Yes") + "  " + RenderKey("N", "No")
	return boxStyle.Render(content)
}

func (m Model) viewJunctions() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Junction Manager") + "\n")
	b.WriteString(DimStyle.Render("Folder: "+path.GetJunctionFolder()) + "\n\n")

	if m.message != "" {
		b.WriteString(SuccessStyle.Render(m.message) + "\n\n")
	}

	b.WriteString(RenderKey("1", "Refresh") + "  " + RenderKey("2", "Suggestions") + "  " + RenderKey("3", "Create") + "\n\n")

	if len(m.junctions) == 0 {
		b.WriteString(DimStyle.Render("No junctions found.") + "\n\n")
	} else {
		b.WriteString(SubtitleStyle.Render(fmt.Sprintf("Current (%d):", len(m.junctions))) + "\n")
		boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Gray).Padding(0, 1)
		var content string
		for i, j := range m.junctions {
			cursor := "  "
			style := NormalStyle
			if i == m.junctionIndex {
				cursor = SelectedStyle.Render("> ")
				style = SelectedStyle
			}
			target := j.Target
			if len(target) > 48 {
				target = target[:45] + "..."
			}
			content += cursor + style.Render(j.Name) + DimStyle.Render(" -> "+target) + "\n"
		}
		b.WriteString(boxStyle.Render(strings.TrimSuffix(content, "\n")) + "\n\n")
		b.WriteString(RenderKey("D", "Delete") + "  ")
	}

	b.WriteString(RenderKey("Esc", "Menu"))
	return b.String()
}

func (m Model) viewJunctionSuggestions() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Suggestions") + " " + DimStyle.Render(fmt.Sprintf("(%d)", len(m.suggestions))) + "\n\n")

	if m.message != "" {
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.message) + "\n\n")
		} else {
			b.WriteString(SuccessStyle.Render(m.message) + "\n\n")
		}
	}

	if len(m.suggestions) == 0 {
		b.WriteString(DimStyle.Render("No paths would benefit from junctions.") + "\n\n")
	} else {
		maxVisible := 12
		start := 0
		if m.junctionIndex > maxVisible/2 {
			start = m.junctionIndex - maxVisible/2
		}
		if start+maxVisible > len(m.suggestions) {
			start = len(m.suggestions) - maxVisible
		}
		if start < 0 {
			start = 0
		}
		end := start + maxVisible
		if end > len(m.suggestions) {
			end = len(m.suggestions)
		}

		if start > 0 {
			b.WriteString(DimStyle.Render(fmt.Sprintf("      ... %d above\n", start)))
		}
		for i := start; i < end; i++ {
			s := m.suggestions[i]
			cursor := "  "
			style := NormalStyle
			if i == m.junctionIndex {
				cursor = SelectedStyle.Render("> ")
				style = SelectedStyle
			}
			saved := SuccessStyle.Render(fmt.Sprintf("-%d", s.SavedChars))
			origPath := s.OriginalPath
			if len(origPath) > 42 {
				origPath = origPath[:39] + "..."
			}
			b.WriteString(fmt.Sprintf("%s%s %s <- %s\n", cursor, saved, style.Render(s.SuggestedName), DimStyle.Render(origPath)))
		}
		if end < len(m.suggestions) {
			b.WriteString(DimStyle.Render(fmt.Sprintf("      ... %d below\n", len(m.suggestions)-end)))
		}

		b.WriteString("\n" + RenderKey("C", "Create selected") + "  ")
	}

	b.WriteString(RenderKey("Esc", "Back"))
	return b.String()
}

func (m Model) viewJunctionCreate() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Create Junction") + "\n\n")

	if m.message != "" {
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.message) + "\n\n")
		}
	}

	nameStyle := NormalStyle
	nameCursor := "  "
	if m.junctionInputMode == 0 {
		nameStyle = SelectedStyle
		nameCursor = SelectedStyle.Render("> ")
	}
	b.WriteString(nameCursor + nameStyle.Render("Name:   ") + NormalStyle.Render(m.junctionName))
	if m.junctionInputMode == 0 {
		b.WriteString(SelectedStyle.Render("_"))
	}
	b.WriteString("\n")

	targetStyle := NormalStyle
	targetCursor := "  "
	if m.junctionInputMode == 1 {
		targetStyle = SelectedStyle
		targetCursor = SelectedStyle.Render("> ")
	}
	b.WriteString(targetCursor + targetStyle.Render("Target: ") + NormalStyle.Render(m.junctionTarget))
	if m.junctionInputMode == 1 {
		b.WriteString(SelectedStyle.Render("_"))
	}
	b.WriteString("\n\n")

	b.WriteString(DimStyle.Render(fmt.Sprintf("Creates: %s\\%s", path.GetJunctionFolder(), m.junctionName)) + "\n\n")
	b.WriteString(RenderKey("Tab", "Switch") + "  " + RenderKey("Enter", "Create") + "  " + RenderKey("Esc", "Cancel"))
	return b.String()
}

func (m Model) viewPathExt() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("PATHEXT Optimizer") + "\n\n")

	if m.pathExtAnalysis == nil {
		return "Analyzing..."
	}

	if m.pathExtEditing {
		// Edit mode
		b.WriteString(SubtitleStyle.Render("Edit PATHEXT order:") + " " + DimStyle.Render("(editing)") + "\n")
		boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Cyan).Padding(0, 1)
		var content string
		for i, ext := range m.pathExtList {
			cursor := "  "
			style := NormalStyle
			if i == m.pathExtIndex {
				cursor = SelectedStyle.Render("> ")
				style = SelectedStyle
			}
			info := path.GetExtensionInfo(ext)
			desc := DimStyle.Render(info.Description)
			content += cursor + style.Render(fmt.Sprintf("%-6s", ext)) + " " + desc + "\n"
		}
		b.WriteString(boxStyle.Render(strings.TrimSuffix(content, "\n")) + "\n\n")

		if m.pathExtOpt != nil {
			b.WriteString(SubtitleStyle.Render("Result: ") + NormalStyle.Render(m.pathExtOpt.OptimizedString) + "\n\n")
		}

		b.WriteString(RenderKey("j/k", "Select") + "  " + RenderKey("J/K", "Move") + "  " + RenderKey("X", "Remove") + "  " + RenderKey("A", "Apply") + "  " + RenderKey("Esc", "Cancel"))
		return b.String()
	}

	// Normal view
	b.WriteString(SubtitleStyle.Render("Current order:") + "\n")
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Gray).Padding(0, 1)
	var content string
	for i, info := range m.pathExtAnalysis.CurrentWithInfo {
		pos := DimStyle.Render(fmt.Sprintf("%2d. ", i+1))
		ext := NormalStyle.Render(fmt.Sprintf("%-6s", info.Ext))
		desc := DimStyle.Render(info.Description)
		if info.IsLegacy {
			desc = WarningStyle.Render(info.Description)
		}
		content += pos + ext + " " + desc + "\n"
	}
	b.WriteString(boxStyle.Render(strings.TrimSuffix(content, "\n")) + "\n\n")

	if len(m.pathExtAnalysis.Issues) == 0 {
		b.WriteString(SuccessStyle.Render("Already optimal!") + "\n\n")
	} else {
		b.WriteString(SubtitleStyle.Render("Issues:") + "\n")
		for _, issue := range m.pathExtAnalysis.Issues {
			var style lipgloss.Style
			switch issue.Impact {
			case "high":
				style = ErrorStyle
			case "medium":
				style = WarningStyle
			default:
				style = DimStyle
			}
			b.WriteString("  " + style.Render(issue.Message) + "\n")
		}
		b.WriteString("\n")
	}

	if m.pathExtOpt != nil && m.pathExtOpt.Changed {
		b.WriteString(SubtitleStyle.Render("Suggested: ") + SuccessStyle.Render(m.pathExtOpt.OptimizedString) + "\n\n")
	}

	b.WriteString(RenderKey("E", "Edit manually") + "  ")
	if m.pathExtOpt != nil && m.pathExtOpt.Changed {
		b.WriteString(RenderKey("O", "Use optimized") + "  " + RenderKey("A", "Apply suggested") + "  ")
	}
	b.WriteString(RenderKey("Esc", "Menu"))
	return b.String()
}

func (m Model) viewSettings() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Settings") + "\n\n")

	settings := []struct {
		name  string
		value string
	}{
		{"Max Backups", fmt.Sprintf("%d", m.config.MaxBackups)},
		{"Auto Backup", fmt.Sprintf("%v", m.config.AutoBackup)},
		{"Junction Folder", m.config.JunctionFolder},
	}

	for i, s := range settings {
		cursor := "  "
		style := NormalStyle
		if i == m.settingsIndex {
			cursor = SelectedStyle.Render("> ")
			style = SelectedStyle
		}
		b.WriteString(cursor + style.Render(s.name+": ") + NormalStyle.Render(s.value) + "\n")
	}

	b.WriteString("\n" + DimStyle.Render("+/- to change") + "\n")
	b.WriteString(RenderKey("Esc", "Menu"))
	return b.String()
}

func (m Model) viewHotPaths() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Hot Paths") + "\n\n")
	b.WriteString(DimStyle.Render("Paths listed here get priority during PATH optimization.") + "\n")
	b.WriteString(DimStyle.Render("Higher in list = higher priority (checked first).") + "\n\n")

	if m.message != "" {
		b.WriteString(SuccessStyle.Render(m.message) + "\n\n")
	}

	if m.hotPathAdding {
		b.WriteString(SubtitleStyle.Render("Add path (paste or type):") + "\n")
		b.WriteString(SelectedStyle.Render("> ") + NormalStyle.Render(m.hotPathInput) + SelectedStyle.Render("_") + "\n\n")
		b.WriteString(RenderKey("Enter", "Add") + "  " + RenderKey("Esc", "Cancel"))
		return b.String()
	}

	if len(m.config.HotPaths) == 0 {
		b.WriteString(DimStyle.Render("No hot paths configured.") + "\n\n")
	} else {
		boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Gray).Padding(0, 1)
		var content string
		for i, p := range m.config.HotPaths {
			cursor := "  "
			style := NormalStyle
			if i == m.hotPathIndex {
				cursor = SelectedStyle.Render("> ")
				style = SelectedStyle
			}
			displayPath := p
			if len(displayPath) > 60 {
				displayPath = displayPath[:57] + "..."
			}
			content += cursor + DimStyle.Render(fmt.Sprintf("%d. ", i+1)) + style.Render(displayPath) + "\n"
		}
		b.WriteString(boxStyle.Render(strings.TrimSuffix(content, "\n")) + "\n\n")
	}

	b.WriteString(RenderKey("A", "Add path") + "  ")
	if len(m.config.HotPaths) > 0 {
		b.WriteString(RenderKey("x", "Delete") + "  " + RenderKey("J/K", "Reorder") + "  ")
	}
	b.WriteString(RenderKey("Esc", "Menu"))
	return b.String()
}

func wrapText(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}
	var result strings.Builder
	for i := 0; i < len(text); i += width {
		end := i + width
		if end > len(text) {
			end = len(text)
		}
		result.WriteString(text[i:end])
		if end < len(text) {
			result.WriteString("\n")
		}
	}
	return result.String()
}
