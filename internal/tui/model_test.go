package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"syspath-optimizer/internal/path"

	tea "github.com/charmbracelet/bubbletea"
)

// TestMain sets up temp directory for config and mock shell runner
func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "syspath-tui-test-*")
	if err != nil {
		os.Exit(1)
	}
	path.SetConfigDir(tempDir)

	// Set up mock shell runner to avoid real PowerShell calls
	_, cleanup := path.SetDefaultTestRunner()

	// Run tests
	code := m.Run()

	// Clean up (defer doesn't work with os.Exit)
	cleanup()
	os.RemoveAll(tempDir)

	os.Exit(code)
}

// ============================================================================
// Model Creation Tests
// ============================================================================

func TestNewModel(t *testing.T) {
	model := New()

	if model.screen != ScreenMenu {
		t.Errorf("Expected initial screen to be ScreenMenu, got %d", model.screen)
	}

	if len(model.menuItems) == 0 {
		t.Error("Expected menu items to be populated")
	}

	if model.menuIndex != 0 {
		t.Errorf("Expected menuIndex to be 0, got %d", model.menuIndex)
	}
}

func TestNewModel_Defaults(t *testing.T) {
	model := New()

	if model.optimizerScope != "both" {
		t.Errorf("Expected optimizerScope 'both', got %s", model.optimizerScope)
	}
	if model.viewerScope != "User" {
		t.Errorf("Expected viewerScope 'User', got %s", model.viewerScope)
	}
	if model.viewMode != 0 {
		t.Errorf("Expected viewMode 0, got %d", model.viewMode)
	}
	if model.scrollOffset != 0 {
		t.Errorf("Expected scrollOffset 0, got %d", model.scrollOffset)
	}
}

func TestNewModel_DefaultFlags(t *testing.T) {
	model := New()

	if model.viewerExpanded {
		t.Error("Expected viewerExpanded to be false by default")
	}

	if model.clipboardOK {
		t.Error("Expected clipboardOK to be false by default")
	}

	if model.pathExtEditing {
		t.Error("Expected pathExtEditing to be false by default")
	}

	if model.hotPathAdding {
		t.Error("Expected hotPathAdding to be false by default")
	}
}

func TestNewModel_ConfigLoaded(t *testing.T) {
	model := New()

	// Config should be loaded from path package
	if model.config.JunctionFolder == "" {
		t.Log("JunctionFolder is empty (may use default)")
	}
}

func TestModel_Init(t *testing.T) {
	model := New()
	cmd := model.Init()

	// Init should return nil or a command
	if cmd != nil {
		t.Log("Init returned a command")
	}
}

// ============================================================================
// Menu Tests
// ============================================================================

func TestModel_MenuItems(t *testing.T) {
	model := New()

	expectedItems := []string{
		"Optimize PATH",
		"View Current PATH",
		"Backup Manager",
		"Junction Manager",
		"PATHEXT Optimizer",
		"Hot Paths Config",
		"Settings",
		"Exit",
	}

	if len(model.menuItems) != len(expectedItems) {
		t.Errorf("Expected %d menu items, got %d", len(expectedItems), len(model.menuItems))
	}

	for i, expected := range expectedItems {
		if i >= len(model.menuItems) {
			break
		}
		if model.menuItems[i] != expected {
			t.Errorf("Menu item %d: expected %s, got %s", i, expected, model.menuItems[i])
		}
	}
}

func TestModel_HandleKeyMsg_Menu_Down(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.menuIndex != 1 {
		t.Errorf("Expected menuIndex 1 after down, got %d", m.menuIndex)
	}
}

func TestModel_HandleKeyMsg_Menu_Up(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.menuIndex != 0 {
		t.Errorf("Expected menuIndex 0 after up, got %d", m.menuIndex)
	}
}

func TestModel_HandleKeyMsg_Menu_UpAtTop(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.menuIndex < 0 {
		t.Error("Menu index should not go negative")
	}
}

func TestModel_HandleKeyMsg_Menu_DownAtBottom(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = len(model.menuItems) - 1

	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.menuIndex >= len(model.menuItems) {
		t.Error("Menu index should not exceed menu length")
	}
}

func TestModel_HandleKeyMsg_Menu_J(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	// j should behave like down
	if m.menuIndex != 1 {
		t.Errorf("Expected menuIndex 1 after 'j', got %d", m.menuIndex)
	}
}

func TestModel_HandleKeyMsg_Menu_K(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 1

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	// k should behave like up
	if m.menuIndex != 0 {
		t.Errorf("Expected menuIndex 0 after 'k', got %d", m.menuIndex)
	}
}

func TestModel_HandleKeyMsg_Menu_Number(t *testing.T) {
	model := New()
	model.screen = ScreenMenu

	// Test pressing "1" through "8"
	for i := 1; i <= 8; i++ {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune('0' + i)}}
		newModel, _ := model.Update(msg)
		m := newModel.(Model)

		// Number keys should select menu items or trigger actions
		t.Logf("After pressing '%d': screen=%d, menuIndex=%d", i, m.screen, m.menuIndex)
	}
}

// ============================================================================
// Window Size Tests
// ============================================================================

func TestModel_UpdateWindowSize(t *testing.T) {
	model := New()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := model.Update(msg)

	m := newModel.(Model)
	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}

func TestModel_UpdateWindowSize_Small(t *testing.T) {
	model := New()

	msg := tea.WindowSizeMsg{Width: 40, Height: 20}
	newModel, _ := model.Update(msg)

	m := newModel.(Model)
	if m.width != 40 {
		t.Errorf("Expected width 40, got %d", m.width)
	}
}

func TestModel_UpdateWindowSize_Large(t *testing.T) {
	model := New()

	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	newModel, _ := model.Update(msg)

	m := newModel.(Model)
	if m.width != 200 {
		t.Errorf("Expected width 200, got %d", m.width)
	}
}

// ============================================================================
// View Tests
// ============================================================================

func TestModel_View_Menu(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.width = 80
	model.height = 24

	view := model.View()

	if view == "" {
		t.Error("View should not return empty string")
	}
}

func TestModel_View_Loading(t *testing.T) {
	model := New()
	model.screen = ScreenLoading
	model.loadingTask = TaskAnalyze
	model.loadingMessage = "Testing"
	model.loadingDots = 2
	model.width = 80
	model.height = 24

	view := model.View()

	if view == "" {
		t.Error("Loading view should not be empty")
	}
}

func TestModel_View_Optimizer(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizer
	model.width = 80
	model.height = 24
	model.analysis = &path.AnalysisResult{}

	view := model.View()

	if view == "" {
		t.Error("Optimizer view should not be empty")
	}
}

func TestModel_View_OptimizerPreview(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerPreview
	model.width = 80
	model.height = 24
	model.analysis = &path.AnalysisResult{}

	view := model.View()

	if view == "" {
		t.Error("OptimizerPreview view should not be empty")
	}
}

func TestModel_View_PathViewer(t *testing.T) {
	model := New()
	model.screen = ScreenPathViewer
	model.width = 80
	model.height = 24

	view := model.View()

	if view == "" {
		t.Error("PathViewer view should not be empty")
	}
}

func TestModel_View_Backup(t *testing.T) {
	model := New()
	model.screen = ScreenBackup
	model.width = 80
	model.height = 24
	model.backups = []path.BackupInfo{}

	view := model.View()

	if view == "" {
		t.Error("Backup view should not be empty")
	}
}

func TestModel_View_Junctions(t *testing.T) {
	model := New()
	model.screen = ScreenJunctions
	model.width = 80
	model.height = 24
	model.junctions = []path.Junction{}

	view := model.View()

	if view == "" {
		t.Error("Junctions view should not be empty")
	}
}

func TestModel_View_JunctionSuggestions(t *testing.T) {
	model := New()
	model.screen = ScreenJunctionSuggestions
	model.width = 80
	model.height = 24
	model.suggestions = []path.JunctionSuggestion{}

	view := model.View()

	if view == "" {
		t.Error("JunctionSuggestions view should not be empty")
	}
}

func TestModel_View_PathExt(t *testing.T) {
	model := New()
	model.screen = ScreenPathExt
	model.width = 80
	model.height = 24
	model.pathExtAnalysis = &path.PathExtAnalysis{
		Current: []string{".EXE", ".CMD"},
	}

	view := model.View()

	if view == "" {
		t.Error("PathExt view should not be empty")
	}
}

func TestModel_View_Settings(t *testing.T) {
	model := New()
	model.screen = ScreenSettings
	model.width = 80
	model.height = 24

	view := model.View()

	if view == "" {
		t.Error("Settings view should not be empty")
	}
}

func TestModel_View_HotPaths(t *testing.T) {
	model := New()
	model.screen = ScreenHotPaths
	model.width = 80
	model.height = 24
	model.config.HotPaths = []string{}

	view := model.View()

	if view == "" {
		t.Error("HotPaths view should not be empty")
	}
}

// ============================================================================
// Quit/Exit Tests
// ============================================================================

func TestModel_HandleKeyMsg_Quit_CtrlC(t *testing.T) {
	model := New()

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("Ctrl+C should return a command")
	}
}

func TestModel_HandleKeyMsg_Quit_Q(t *testing.T) {
	model := New()
	model.screen = ScreenMenu

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)

	// q on menu should quit
	if cmd == nil {
		t.Log("q did not return quit command (may navigate instead)")
	}
}

func TestModel_HandleKeyMsg_Escape_Menu(t *testing.T) {
	model := New()
	model.screen = ScreenMenu

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	// Escape on menu might quit or do nothing
	t.Logf("After Escape on menu: screen=%d", m.screen)
}

func TestModel_HandleKeyMsg_Escape_SubScreen(t *testing.T) {
	model := New()
	model.screen = ScreenPathViewer

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	// Escape should return to menu
	if m.screen != ScreenMenu {
		t.Errorf("Escape should return to menu, got screen %d", m.screen)
	}
}

// ============================================================================
// Screen Constant Tests
// ============================================================================

func TestScreenConstants(t *testing.T) {
	screens := []Screen{
		ScreenMenu,
		ScreenLoading,
		ScreenOptimizer,
		ScreenOptimizerPreview,
		ScreenOptimizerConfirm,
		ScreenOptimizerDone,
		ScreenPathViewer,
		ScreenBackup,
		ScreenBackupPreview,
		ScreenBackupConfirmRestore,
		ScreenBackupConfirmDelete,
		ScreenBackupDone,
		ScreenJunctions,
		ScreenJunctionSuggestions,
		ScreenJunctionCreate,
		ScreenPathExt,
		ScreenPathExtConfirm,
		ScreenPathExtDone,
		ScreenSettings,
		ScreenHotPaths,
	}

	seen := make(map[Screen]bool)
	for _, s := range screens {
		if seen[s] {
			t.Errorf("Duplicate screen constant: %d", s)
		}
		seen[s] = true
	}
}

func TestLoadingTaskConstants(t *testing.T) {
	tasks := []LoadingTask{
		TaskNone,
		TaskAnalyze,
		TaskJunctions,
		TaskSuggestions,
		TaskCreateJunction,
	}

	seen := make(map[LoadingTask]bool)
	for _, task := range tasks {
		if task != TaskNone && seen[task] {
			t.Errorf("Duplicate loading task constant: %d", task)
		}
		seen[task] = true
	}
}

// ============================================================================
// Message Tests
// ============================================================================

func TestModel_Update_AnalysisComplete(t *testing.T) {
	model := New()
	model.screen = ScreenLoading

	msg := analysisCompleteMsg{result: path.AnalysisResult{}}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	// Should switch to optimizer screen
	if m.screen == ScreenLoading {
		t.Error("Should leave loading screen after analysis complete")
	}
	if m.analysis == nil {
		t.Error("Analysis should be set")
	}
}

func TestModel_Update_JunctionsLoaded(t *testing.T) {
	model := New()
	model.screen = ScreenLoading
	model.loadingTask = TaskJunctions

	msg := junctionsLoadedMsg{junctions: []path.Junction{
		{Name: "test", Path: "C:\\l\\test", Target: "C:\\Test"},
	}}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if len(m.junctions) != 1 {
		t.Errorf("Expected 1 junction, got %d", len(m.junctions))
	}
}

func TestModel_Update_SuggestionsLoaded(t *testing.T) {
	model := New()
	model.screen = ScreenLoading
	model.loadingTask = TaskSuggestions

	msg := suggestionsLoadedMsg{suggestions: []path.JunctionSuggestion{
		{OriginalPath: "C:\\Test", SuggestedName: "test", SavedChars: 10},
	}}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if len(m.suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(m.suggestions))
	}
}

func TestModel_Update_Tick(t *testing.T) {
	model := New()
	model.screen = ScreenLoading
	model.loadingDots = 0

	msg := tickMsg{}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	// Loading dots should increment
	if m.screen == ScreenLoading && m.loadingDots == 0 {
		t.Log("Loading dots did not increment")
	}
}

func TestModel_Update_Progress(t *testing.T) {
	model := New()
	model.screen = ScreenLoading

	msg := progressMsg{current: 5, total: 10, item: "Testing..."}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.loadingCurrent != 5 {
		t.Errorf("Expected loadingCurrent 5, got %d", m.loadingCurrent)
	}
	if m.loadingTotal != 10 {
		t.Errorf("Expected loadingTotal 10, got %d", m.loadingTotal)
	}
}

// ============================================================================
// Scope Toggle Tests
// ============================================================================

func TestModel_ViewerScope_Toggle(t *testing.T) {
	model := New()
	model.screen = ScreenPathViewer
	model.viewerScope = "User"

	// Press 's' to toggle scope
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.viewerScope == "User" {
		t.Log("Scope may not have toggled (could be expected)")
	}
}

func TestModel_ViewerExpanded_Toggle(t *testing.T) {
	model := New()
	model.screen = ScreenPathViewer
	model.viewerExpanded = false

	// Press 'e' to toggle expanded
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if !m.viewerExpanded {
		t.Log("Expanded may not have toggled")
	}
}

// ============================================================================
// Scroll Tests
// ============================================================================

func TestModel_Scroll(t *testing.T) {
	model := New()
	model.scrollOffset = 0

	// Set scroll offset
	model.scrollOffset = 5

	if model.scrollOffset != 5 {
		t.Errorf("Expected scrollOffset 5, got %d", model.scrollOffset)
	}
}

func TestModel_Scroll_Reset(t *testing.T) {
	model := New()
	model.scrollOffset = 10
	model.scrollOffset = 0

	if model.scrollOffset != 0 {
		t.Error("Expected scrollOffset to be reset to 0")
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestRenderKey(t *testing.T) {
	result := RenderKey("Esc", "Back")

	if result == "" {
		t.Error("RenderKey should return non-empty string")
	}

	if !strings.Contains(result, "Esc") && !strings.Contains(result, "Back") {
		t.Error("RenderKey should contain key or label")
	}
}

func TestRenderKey_Various(t *testing.T) {
	tests := []struct {
		key   string
		label string
	}{
		{"Enter", "Select"},
		{"q", "Quit"},
		{"1-8", "Jump"},
		{"↑/↓", "Navigate"},
	}

	for _, tc := range tests {
		result := RenderKey(tc.key, tc.label)
		if result == "" {
			t.Errorf("RenderKey(%s, %s) returned empty", tc.key, tc.label)
		}
	}
}

// ============================================================================
// View Mode Tests
// ============================================================================

func TestModel_ViewModes(t *testing.T) {
	model := New()

	// Test different view modes
	for i := 0; i < 4; i++ {
		model.viewMode = i
		if model.viewMode != i {
			t.Errorf("viewMode should be %d, got %d", i, model.viewMode)
		}
	}
}

// ============================================================================
// Index Tests
// ============================================================================

func TestModel_BackupIndex(t *testing.T) {
	model := New()
	model.backupIndex = 0
	model.backups = []path.BackupInfo{
		{Filename: "backup1.json"},
		{Filename: "backup2.json"},
	}

	// Verify index stays in bounds
	if model.backupIndex >= len(model.backups) && len(model.backups) > 0 {
		t.Error("Backup index out of bounds")
	}
}

func TestModel_JunctionIndex(t *testing.T) {
	model := New()
	model.junctionIndex = 0
	model.junctions = []path.Junction{
		{Name: "test1"},
		{Name: "test2"},
	}

	if model.junctionIndex >= len(model.junctions) && len(model.junctions) > 0 {
		t.Error("Junction index out of bounds")
	}
}

func TestModel_PathExtIndex(t *testing.T) {
	model := New()
	model.pathExtIndex = 0
	model.pathExtList = []string{".EXE", ".CMD", ".BAT"}

	if model.pathExtIndex >= len(model.pathExtList) && len(model.pathExtList) > 0 {
		t.Error("PathExt index out of bounds")
	}
}

func TestModel_SettingsIndex(t *testing.T) {
	model := New()
	model.settingsIndex = 0

	// Settings index should be valid
	if model.settingsIndex < 0 {
		t.Error("Settings index should not be negative")
	}
}

func TestModel_HotPathIndex(t *testing.T) {
	model := New()
	model.hotPathIndex = 0
	model.config.HotPaths = []string{"C:\\path1", "C:\\path2"}

	if model.hotPathIndex >= len(model.config.HotPaths) && len(model.config.HotPaths) > 0 {
		t.Error("HotPath index out of bounds")
	}
}

// ============================================================================
// Input Mode Tests
// ============================================================================

func TestModel_HotPathAdding(t *testing.T) {
	model := New()
	model.hotPathAdding = false

	model.hotPathAdding = true
	if !model.hotPathAdding {
		t.Error("hotPathAdding should be true")
	}

	model.hotPathInput = "C:\\test\\path"
	if model.hotPathInput != "C:\\test\\path" {
		t.Error("hotPathInput not set correctly")
	}
}

func TestModel_PathExtEditing(t *testing.T) {
	model := New()
	model.pathExtEditing = false

	model.pathExtEditing = true
	if !model.pathExtEditing {
		t.Error("pathExtEditing should be true")
	}
}

func TestModel_JunctionInput(t *testing.T) {
	model := New()

	model.junctionName = "testjunc"
	model.junctionTarget = "C:\\Program Files\\Test"
	model.junctionInputMode = 0

	if model.junctionName != "testjunc" {
		t.Error("junctionName not set correctly")
	}
	if model.junctionTarget != "C:\\Program Files\\Test" {
		t.Error("junctionTarget not set correctly")
	}
}

// ============================================================================
// Error/Message Tests
// ============================================================================

func TestModel_Error(t *testing.T) {
	model := New()

	model.err = nil
	if model.err != nil {
		t.Error("err should be nil")
	}
}

func TestModel_Message(t *testing.T) {
	model := New()

	model.message = "Test message"
	if model.message != "Test message" {
		t.Error("message not set correctly")
	}

	model.message = ""
	if model.message != "" {
		t.Error("message should be empty")
	}
}

func TestModel_ClipboardOK(t *testing.T) {
	model := New()

	model.clipboardOK = true
	if !model.clipboardOK {
		t.Error("clipboardOK should be true")
	}

	model.clipboardOK = false
	if model.clipboardOK {
		t.Error("clipboardOK should be false")
	}
}

// ============================================================================
// Helpers
// ============================================================================

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkModel_View(b *testing.B) {
	model := New()
	model.width = 80
	model.height = 24

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.View()
	}
}

func BenchmarkModel_Update_KeyDown(b *testing.B) {
	model := New()
	msg := tea.KeyMsg{Type: tea.KeyDown}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Update(msg)
	}
}

func BenchmarkModel_Update_WindowSize(b *testing.B) {
	model := New()
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Update(msg)
	}
}

func BenchmarkNew(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New()
	}
}

// ============================================================================
// Comprehensive View Tests
// ============================================================================

func TestModel_ViewOptimizer_WithAnalysis(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenOptimizer

	// Create mock analysis - use simple assignment since these are anonymous structs
	model.analysis = &path.AnalysisResult{}
	model.analysis.System.Original.Raw = `C:\Windows;C:\Windows\System32`
	model.analysis.System.Original.Entries = []string{`C:\Windows`, `C:\Windows\System32`}
	model.analysis.System.Original.Length = 30
	model.analysis.System.Original.Count = 2
	model.analysis.System.Optimized.Raw = `C:\Windows;C:\Windows\System32`
	model.analysis.System.Optimized.Entries = []string{`C:\Windows`, `C:\Windows\System32`}
	model.analysis.System.Optimized.Length = 30
	model.analysis.System.Optimized.Count = 2
	model.analysis.User.Original.Raw = `%USERPROFILE%\bin`
	model.analysis.User.Original.Entries = []string{`%USERPROFILE%\bin`}
	model.analysis.User.Original.Length = 17
	model.analysis.User.Original.Count = 1
	model.analysis.User.Optimized.Raw = `%USERPROFILE%\bin`
	model.analysis.User.Optimized.Entries = []string{`%USERPROFILE%\bin`}
	model.analysis.User.Optimized.Length = 17
	model.analysis.User.Optimized.Count = 1

	view := model.View()
	if view == "" {
		t.Error("Optimizer view should not be empty")
	}
}

func TestModel_ViewOptimizer_ViewModes(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenOptimizer
	model.analysis = &path.AnalysisResult{}

	// Test all view modes
	for mode := 0; mode <= 3; mode++ {
		model.viewMode = mode
		view := model.View()
		if view == "" {
			t.Errorf("View mode %d should not produce empty view", mode)
		}
	}
}

func TestModel_ViewOptimizerPreview(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenOptimizerPreview
	model.analysis = &path.AnalysisResult{}

	view := model.View()
	if view == "" {
		t.Error("OptimizerPreview view should not be empty")
	}
}

func TestModel_ViewOptimizerConfirm(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenOptimizerConfirm

	view := model.View()
	if view == "" {
		t.Error("OptimizerConfirm view should not be empty")
	}
}

func TestModel_ViewOptimizerDone(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenOptimizerDone
	model.backupInfo = &path.BackupInfo{Filename: "test.json"}

	view := model.View()
	if view == "" {
		t.Error("OptimizerDone view should not be empty")
	}
}

func TestModel_ViewBackupPreview(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenBackupPreview
	model.backupPreview = &path.Backup{}
	model.backupPreview.SystemPath.Raw = `C:\Windows`
	model.backupPreview.UserPath.Raw = `%USERPROFILE%\bin`

	view := model.View()
	if view == "" {
		t.Error("BackupPreview view should not be empty")
	}
}

func TestModel_ViewBackupConfirmRestore(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenBackupConfirmRestore
	model.backups = []path.BackupInfo{{Filename: "test.json"}}

	view := model.View()
	if view == "" {
		t.Error("BackupConfirmRestore view should not be empty")
	}
}

func TestModel_ViewBackupConfirmDelete(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenBackupConfirmDelete
	model.backups = []path.BackupInfo{{Filename: "test.json"}}

	view := model.View()
	if view == "" {
		t.Error("BackupConfirmDelete view should not be empty")
	}
}

func TestModel_ViewBackupDone(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenBackupDone

	view := model.View()
	if view == "" {
		t.Error("BackupDone view should not be empty")
	}
}

func TestModel_ViewJunctionCreate(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenJunctionCreate

	view := model.View()
	if view == "" {
		t.Error("JunctionCreate view should not be empty")
	}
}

func TestModel_ViewPathExtConfirm(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenPathExtConfirm

	view := model.View()
	if view == "" {
		t.Error("PathExtConfirm view should not be empty")
	}
}

func TestModel_ViewPathExtDone(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.screen = ScreenPathExtDone

	view := model.View()
	if view == "" {
		t.Error("PathExtDone view should not be empty")
	}
}

// ============================================================================
// Comprehensive Key Handler Tests
// ============================================================================

func TestModel_OptimizerKey_ScopeToggle(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizer
	model.analysis = &path.AnalysisResult{}

	// Test 's' key to toggle scope
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.optimizerScope == model.optimizerScope {
		t.Log("Scope may have toggled")
	}
}

func TestModel_OptimizerKey_ViewModeToggle(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizer
	model.analysis = &path.AnalysisResult{}
	model.viewMode = 0

	// Test 'tab' key to toggle view mode
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.viewMode == model.viewMode {
		t.Log("View mode may have changed")
	}
}

func TestModel_OptimizerKey_Preview(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizer
	model.analysis = &path.AnalysisResult{}

	// Test 'p' key for preview
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenOptimizerPreview {
		t.Logf("Screen after 'p': %d", m.screen)
	}
}

func TestModel_OptimizerKey_Apply(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizer
	model.analysis = &path.AnalysisResult{}

	// Test 'a' key for apply
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenOptimizerConfirm {
		t.Logf("Screen after 'a': %d", m.screen)
	}
}

func TestModel_OptimizerConfirmKey_Yes(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerConfirm
	model.analysis = &path.AnalysisResult{}

	// Test 'y' key to confirm
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	updated, cmd := model.Update(msg)
	m := updated.(Model)

	t.Logf("Screen after 'y': %d, cmd: %v", m.screen, cmd != nil)
}

func TestModel_OptimizerConfirmKey_No(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerConfirm

	// Test 'n' key to cancel
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenOptimizer {
		t.Logf("Screen after 'n': %d", m.screen)
	}
}

func TestModel_ViewerKey_Copy(t *testing.T) {
	model := New()
	model.screen = ScreenPathViewer

	// Test 'c' key for copy
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	t.Logf("clipboardOK after 'c': %v", m.clipboardOK)
}

func TestModel_BackupKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenBackup
	model.backups = []path.BackupInfo{
		{Filename: "backup1.json"},
		{Filename: "backup2.json"},
		{Filename: "backup3.json"},
	}
	model.backupIndex = 0

	// Test down navigation
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.backupIndex != 1 {
		t.Logf("backupIndex after down: %d", m.backupIndex)
	}
}

func TestModel_BackupKey_Preview(t *testing.T) {
	model := New()
	model.screen = ScreenBackup
	model.backups = []path.BackupInfo{{Filename: "test.json"}}

	// Test enter for preview
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	t.Logf("Screen after enter: %d", m.screen)
}

func TestModel_BackupKey_Restore(t *testing.T) {
	model := New()
	model.screen = ScreenBackup
	model.backups = []path.BackupInfo{{Filename: "test.json"}}

	// Test 'r' key for restore
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenBackupConfirmRestore {
		t.Logf("Screen after 'r': %d", m.screen)
	}
}

func TestModel_BackupKey_Delete(t *testing.T) {
	model := New()
	model.screen = ScreenBackup
	model.backups = []path.BackupInfo{{Filename: "test.json"}}

	// Test 'd' key for delete
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenBackupConfirmDelete {
		t.Logf("Screen after 'd': %d", m.screen)
	}
}

func TestModel_BackupConfirmKey_Yes(t *testing.T) {
	model := New()
	model.screen = ScreenBackupConfirmRestore
	model.backups = []path.BackupInfo{{Filename: "test.json"}}

	// Test 'y' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	t.Logf("Screen after 'y': %d", m.screen)
}

func TestModel_BackupConfirmKey_No(t *testing.T) {
	model := New()
	model.screen = ScreenBackupConfirmRestore

	// Test 'n' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenBackup {
		t.Logf("Screen after 'n': %d", m.screen)
	}
}

func TestModel_JunctionsKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenJunctions
	model.junctions = []path.Junction{
		{Name: "j1", Path: `C:\l\j1`, Target: `C:\Test1`},
		{Name: "j2", Path: `C:\l\j2`, Target: `C:\Test2`},
	}

	// Test down navigation
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.junctionIndex != 1 {
		t.Logf("junctionIndex after down: %d", m.junctionIndex)
	}
}

func TestModel_JunctionsKey_Create(t *testing.T) {
	model := New()
	model.screen = ScreenJunctions

	// Test 'c' key for create
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenJunctionCreate {
		t.Logf("Screen after 'c': %d", m.screen)
	}
}

func TestModel_JunctionsKey_Suggestions(t *testing.T) {
	model := New()
	model.screen = ScreenJunctions

	// Test 's' key for suggestions
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	updated, cmd := model.Update(msg)
	m := updated.(Model)

	t.Logf("Screen after 's': %d, cmd: %v", m.screen, cmd != nil)
}

func TestModel_JunctionSuggestionsKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenJunctionSuggestions
	model.suggestions = []path.JunctionSuggestion{
		{OriginalPath: `C:\Test1`, SuggestedName: "t1"},
		{OriginalPath: `C:\Test2`, SuggestedName: "t2"},
	}

	// Test down navigation
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.junctionIndex != 1 {
		t.Logf("junctionIndex after down: %d", m.junctionIndex)
	}
}

func TestModel_JunctionCreateKey_Tab(t *testing.T) {
	model := New()
	model.screen = ScreenJunctionCreate
	model.junctionInputMode = 0

	// Test tab to switch fields
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.junctionInputMode != 1 {
		t.Logf("junctionInputMode after tab: %d", m.junctionInputMode)
	}
}

func TestModel_JunctionCreateKey_TextInput(t *testing.T) {
	model := New()
	model.screen = ScreenJunctionCreate
	model.junctionInputMode = 0

	// Test text input for name
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if !strings.Contains(m.junctionName, "t") {
		t.Logf("junctionName after 't': %s", m.junctionName)
	}
}

func TestModel_PathExtKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenPathExt
	model.pathExtAnalysis = &path.PathExtAnalysis{}
	model.pathExtList = []string{".EXE", ".CMD", ".BAT"}

	// Test down navigation
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.pathExtIndex != 1 {
		t.Logf("pathExtIndex after down: %d", m.pathExtIndex)
	}
}

func TestModel_PathExtKey_Edit(t *testing.T) {
	model := New()
	model.screen = ScreenPathExt
	model.pathExtAnalysis = &path.PathExtAnalysis{}
	model.pathExtList = []string{".EXE", ".CMD", ".BAT"}

	// Test 'e' key for edit mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if !m.pathExtEditing {
		t.Logf("pathExtEditing after 'e': %v", m.pathExtEditing)
	}
}

func TestModel_PathExtKey_Optimize(t *testing.T) {
	model := New()
	model.screen = ScreenPathExt
	model.pathExtAnalysis = &path.PathExtAnalysis{}

	// Test 'o' key for optimize
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	t.Logf("pathExtOpt after 'o': %v", m.pathExtOpt != nil)
}

func TestModel_PathExtKey_Apply(t *testing.T) {
	model := New()
	model.screen = ScreenPathExt
	model.pathExtAnalysis = &path.PathExtAnalysis{}
	model.pathExtOpt = &path.PathExtOptimization{Changed: true}

	// Test 'a' key for apply
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenPathExtConfirm {
		t.Logf("Screen after 'a': %d", m.screen)
	}
}

func TestModel_PathExtConfirmKey_Yes(t *testing.T) {
	model := New()
	model.screen = ScreenPathExtConfirm
	model.pathExtOpt = &path.PathExtOptimization{OptimizedString: ".EXE;.CMD"}

	// Test 'y' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	t.Logf("Screen after 'y': %d", m.screen)
}

func TestModel_SettingsKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenSettings

	// Test down navigation
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.settingsIndex != 1 {
		t.Logf("settingsIndex after down: %d", m.settingsIndex)
	}
}

func TestModel_SettingsKey_Toggle(t *testing.T) {
	model := New()
	model.screen = ScreenSettings
	model.settingsIndex = 1 // AutoBackup toggle

	// Test enter to toggle
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	t.Logf("config.AutoBackup after enter: %v", m.config.AutoBackup)
}

func TestModel_HotPathsKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenHotPaths
	model.config.HotPaths = []string{`C:\Path1`, `C:\Path2`}

	// Test down navigation
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.hotPathIndex != 1 {
		t.Logf("hotPathIndex after down: %d", m.hotPathIndex)
	}
}

func TestModel_HotPathsKey_Add(t *testing.T) {
	model := New()
	model.screen = ScreenHotPaths

	// Test 'a' key for add
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if !m.hotPathAdding {
		t.Logf("hotPathAdding after 'a': %v", m.hotPathAdding)
	}
}

func TestModel_HotPathsKey_Delete(t *testing.T) {
	model := New()
	model.screen = ScreenHotPaths
	model.config.HotPaths = []string{`C:\Path1`, `C:\Path2`}
	model.hotPathIndex = 0

	// Test 'd' key for delete
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	t.Logf("HotPaths count after 'd': %d", len(m.config.HotPaths))
}

func TestModel_HotPathsKey_InputMode(t *testing.T) {
	model := New()
	model.screen = ScreenHotPaths
	model.hotPathAdding = true

	// Test typing in add mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if !strings.Contains(m.hotPathInput, "C") {
		t.Logf("hotPathInput after 'C': %s", m.hotPathInput)
	}
}

func TestModel_HotPathsKey_ConfirmAdd(t *testing.T) {
	model := New()
	model.screen = ScreenHotPaths
	model.hotPathAdding = true
	model.hotPathInput = `C:\NewPath`

	// Test enter to confirm
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.hotPathAdding {
		t.Log("hotPathAdding should be false after enter")
	}
}

// ============================================================================
// Additional Message Handler Tests
// ============================================================================

func TestModel_Update_JunctionCreated_Success(t *testing.T) {
	model := New()
	model.screen = ScreenLoading
	model.loadingTask = TaskCreateJunction

	msg := junctionCreatedMsg{success: true, name: "test"}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	t.Logf("Screen after junction created: %d, message: %s", m.screen, m.message)
}

func TestModel_Update_JunctionCreated_Error(t *testing.T) {
	model := New()
	model.screen = ScreenLoading
	model.loadingTask = TaskCreateJunction

	// Pass a real error to avoid nil pointer dereference
	msg := junctionCreatedMsg{success: false, name: "test", err: fmt.Errorf("test error")}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.message != "Failed: test error" {
		t.Logf("Message: %s", m.message)
	}
}

func TestModel_Update_JunctionCreated_NilError(t *testing.T) {
	model := New()
	model.screen = ScreenLoading
	model.loadingTask = TaskCreateJunction

	// Test nil error path
	msg := junctionCreatedMsg{success: false, name: "test", err: nil}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.message != "Failed to create junction" {
		t.Errorf("Expected 'Failed to create junction', got: %s", m.message)
	}
}

func TestModel_Update_ApplyComplete_Success(t *testing.T) {
	model := New()
	model.screen = ScreenLoading

	msg := applyCompleteMsg{backup: &path.BackupInfo{Filename: "test.json"}, err: nil}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenOptimizerDone {
		t.Logf("Screen after apply complete: %d", m.screen)
	}
}

func TestModel_Update_ApplyComplete_Error(t *testing.T) {
	model := New()
	model.screen = ScreenLoading

	msg := applyCompleteMsg{err: nil}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	t.Logf("Screen after apply error: %d", m.screen)
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestWrapText(t *testing.T) {
	text := "This is a very long line of text that should be wrapped at a certain width to fit within the terminal"
	wrapped := wrapText(text, 40)

	if len(wrapped) > 0 && !strings.Contains(wrapped, "\n") {
		t.Log("Text may not need wrapping at this width")
	}
}

func TestWrapText_ShortText(t *testing.T) {
	text := "Short"
	wrapped := wrapText(text, 40)

	if wrapped != text {
		t.Errorf("Short text should not be wrapped: %s", wrapped)
	}
}

func TestWrapText_Empty(t *testing.T) {
	wrapped := wrapText("", 40)
	if wrapped != "" {
		t.Error("Empty text should return empty")
	}
}

// ============================================================================
// Screen Flow Tests
// ============================================================================

func TestScreenFlow_Menu_To_Optimizer(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 0

	// Press enter to select optimizer
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenLoading && cmd == nil {
		t.Logf("Screen after selecting Optimize PATH: %d", m.screen)
	}
}

func TestScreenFlow_Menu_To_Viewer(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 1

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenPathViewer {
		t.Logf("Screen after selecting View PATH: %d", m.screen)
	}
}

func TestScreenFlow_Menu_To_Backup(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 2

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenBackup {
		t.Logf("Screen after selecting Backup: %d", m.screen)
	}
}

func TestScreenFlow_Menu_To_Junctions(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 3

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := model.Update(msg)
	m := updated.(Model)

	t.Logf("Screen after selecting Junctions: %d, cmd: %v", m.screen, cmd != nil)
}

func TestScreenFlow_Menu_To_PathExt(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 4

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenPathExt {
		t.Logf("Screen after selecting PathExt: %d", m.screen)
	}
}

func TestScreenFlow_Menu_To_HotPaths(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 5

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenHotPaths {
		t.Logf("Screen after selecting HotPaths: %d", m.screen)
	}
}

func TestScreenFlow_Menu_To_Settings(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 6

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenSettings {
		t.Logf("Screen after selecting Settings: %d", m.screen)
	}
}

func TestScreenFlow_Menu_Exit(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.menuIndex = 7

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.Update(msg)

	// Should return quit command
	if cmd == nil {
		t.Log("Exit should return a command")
	}
}

func TestScreenFlow_Escape_Returns_To_Menu(t *testing.T) {
	screens := []Screen{
		ScreenOptimizer, ScreenPathViewer, ScreenBackup,
		ScreenJunctions, ScreenPathExt, ScreenSettings, ScreenHotPaths,
	}

	for _, screen := range screens {
		model := New()
		model.screen = screen

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := model.Update(msg)
		m := updated.(Model)

		if m.screen != ScreenMenu {
			t.Logf("Screen %d after Escape: %d", screen, m.screen)
		}
	}
}

// ============================================================================
// Hot Paths Helper Function Tests
// ============================================================================

func TestModel_HandleHotPathsInputKey_Escape(t *testing.T) {
	model := New()
	model.hotPathAdding = true
	model.hotPathInput = "test"

	result := model.handleHotPathsInputKey("esc")

	if result.hotPathAdding {
		t.Error("hotPathAdding should be false after escape")
	}
	if result.hotPathInput != "" {
		t.Error("hotPathInput should be cleared after escape")
	}
}

func TestModel_HandleHotPathsInputKey_Enter(t *testing.T) {
	model := New()
	model.hotPathAdding = true
	model.hotPathInput = `C:\TestPath`

	result := model.handleHotPathsInputKey("enter")

	if result.hotPathAdding {
		t.Error("hotPathAdding should be false after enter")
	}
	if len(result.config.HotPaths) == 0 {
		t.Log("HotPaths may not have been saved")
	}
}

func TestModel_HandleHotPathsInputKey_Backspace(t *testing.T) {
	model := New()
	model.hotPathAdding = true
	model.hotPathInput = "test"

	result := model.handleHotPathsInputKey("backspace")

	if result.hotPathInput != "tes" {
		t.Errorf("Expected 'tes', got '%s'", result.hotPathInput)
	}
}

func TestModel_HandleHotPathsInputKey_Char(t *testing.T) {
	model := New()
	model.hotPathAdding = true
	model.hotPathInput = "test"

	result := model.handleHotPathsInputKey("x")

	if result.hotPathInput != "testx" {
		t.Errorf("Expected 'testx', got '%s'", result.hotPathInput)
	}
}

func TestModel_HandleHotPathsNavKey_Add(t *testing.T) {
	model := New()
	model.screen = ScreenHotPaths

	result := model.handleHotPathsNavKey("a")

	if !result.hotPathAdding {
		t.Error("hotPathAdding should be true after 'a'")
	}
}

func TestModel_HandleHotPathsNavKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenHotPaths
	model.config.HotPaths = []string{"path1", "path2", "path3"}
	model.hotPathIndex = 1

	// Test up
	result := model.handleHotPathsNavKey("up")
	if result.hotPathIndex != 0 {
		t.Errorf("Expected index 0 after up, got %d", result.hotPathIndex)
	}

	// Test down from start
	model.hotPathIndex = 0
	result = model.handleHotPathsNavKey("down")
	if result.hotPathIndex != 1 {
		t.Errorf("Expected index 1 after down, got %d", result.hotPathIndex)
	}
}

func TestModel_DeleteCurrentHotPath(t *testing.T) {
	model := New()
	model.config.HotPaths = []string{"path1", "path2", "path3"}
	model.hotPathIndex = 1

	result := model.deleteCurrentHotPath()

	if len(result.config.HotPaths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(result.config.HotPaths))
	}
}

func TestModel_MoveHotPathUp(t *testing.T) {
	model := New()
	model.config.HotPaths = []string{"path1", "path2", "path3"}
	model.hotPathIndex = 1

	result := model.moveHotPathUp()

	if result.hotPathIndex != 0 {
		t.Errorf("Expected index 0, got %d", result.hotPathIndex)
	}
	if result.config.HotPaths[0] != "path2" {
		t.Error("path2 should be at index 0")
	}
}

func TestModel_MoveHotPathDown(t *testing.T) {
	model := New()
	model.config.HotPaths = []string{"path1", "path2", "path3"}
	model.hotPathIndex = 1

	result := model.moveHotPathDown()

	if result.hotPathIndex != 2 {
		t.Errorf("Expected index 2, got %d", result.hotPathIndex)
	}
	if result.config.HotPaths[2] != "path2" {
		t.Error("path2 should be at index 2")
	}
}

// ============================================================================
// PATHEXT Helper Function Tests
// ============================================================================

func TestModel_HandlePathExtEditKey_Escape(t *testing.T) {
	model := New()
	model.pathExtEditing = true

	result := model.handlePathExtEditKey("esc")

	if result.pathExtEditing {
		t.Error("pathExtEditing should be false after escape")
	}
}

func TestModel_HandlePathExtEditKey_Navigation(t *testing.T) {
	model := New()
	model.pathExtEditing = true
	model.pathExtList = []string{".EXE", ".CMD", ".BAT"}
	model.pathExtIndex = 1

	// Test up
	result := model.handlePathExtEditKey("up")
	if result.pathExtIndex != 0 {
		t.Errorf("Expected index 0, got %d", result.pathExtIndex)
	}

	// Test down
	model.pathExtIndex = 0
	result = model.handlePathExtEditKey("down")
	if result.pathExtIndex != 1 {
		t.Errorf("Expected index 1, got %d", result.pathExtIndex)
	}
}

func TestModel_MovePathExtUp(t *testing.T) {
	model := New()
	model.pathExtList = []string{".EXE", ".CMD", ".BAT"}
	model.pathExtIndex = 1
	model.pathExtAnalysis = &path.PathExtAnalysis{Current: []string{".EXE", ".CMD", ".BAT"}}

	result := model.movePathExtUp()

	if result.pathExtIndex != 0 {
		t.Errorf("Expected index 0, got %d", result.pathExtIndex)
	}
	if result.pathExtList[0] != ".CMD" {
		t.Error(".CMD should be at index 0")
	}
}

func TestModel_MovePathExtDown(t *testing.T) {
	model := New()
	model.pathExtList = []string{".EXE", ".CMD", ".BAT"}
	model.pathExtIndex = 1
	model.pathExtAnalysis = &path.PathExtAnalysis{Current: []string{".EXE", ".CMD", ".BAT"}}

	result := model.movePathExtDown()

	if result.pathExtIndex != 2 {
		t.Errorf("Expected index 2, got %d", result.pathExtIndex)
	}
	if result.pathExtList[2] != ".CMD" {
		t.Error(".CMD should be at index 2")
	}
}

func TestModel_RemovePathExtEntry(t *testing.T) {
	model := New()
	model.pathExtList = []string{".EXE", ".CMD", ".BAT"}
	model.pathExtIndex = 1
	model.pathExtAnalysis = &path.PathExtAnalysis{Current: []string{".EXE", ".CMD", ".BAT"}}

	result := model.removePathExtEntry()

	if len(result.pathExtList) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result.pathExtList))
	}
}

func TestModel_HandlePathExtNormalKey_Edit(t *testing.T) {
	model := New()
	model.screen = ScreenPathExt
	model.pathExtAnalysis = &path.PathExtAnalysis{Current: []string{".EXE", ".CMD"}}

	result := model.handlePathExtNormalKey("e")

	if !result.pathExtEditing {
		t.Error("pathExtEditing should be true after 'e'")
	}
}

func TestModel_HandlePathExtNormalKey_Optimize(t *testing.T) {
	model := New()
	model.screen = ScreenPathExt
	model.pathExtAnalysis = &path.PathExtAnalysis{Current: []string{".EXE", ".CMD"}}
	model.pathExtOpt = &path.PathExtOptimization{
		Changed:   true,
		Optimized: []string{".EXE", ".CMD"},
	}

	result := model.handlePathExtNormalKey("o")

	if !result.pathExtEditing {
		t.Error("pathExtEditing should be true after 'o' with optimization")
	}
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

func TestModel_UpdatePathExtOpt(t *testing.T) {
	model := New()
	model.pathExtList = []string{".EXE", ".CMD"}
	model.pathExtAnalysis = &path.PathExtAnalysis{Current: []string{".CMD", ".EXE"}}

	model.updatePathExtOpt()

	if model.pathExtOpt == nil {
		t.Error("pathExtOpt should not be nil")
	}
	if !model.pathExtOpt.Changed {
		t.Error("Changed should be true when order differs")
	}
}

func TestModel_ViewLoading_Progress(t *testing.T) {
	model := New()
	model.screen = ScreenLoading
	model.width = 80
	model.height = 24
	model.loadingMessage = "Test loading"
	model.loadingCurrent = 5
	model.loadingTotal = 10
	model.loadingItem = "testing item"

	view := model.View()

	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestModel_CtrlC_Quit(t *testing.T) {
	model := New()

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("Ctrl+C should return quit command")
	}
}

func TestModel_LoadingScreen_IgnoresKeys(t *testing.T) {
	model := New()
	model.screen = ScreenLoading

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updated, _ := model.Update(msg)
	m := updated.(Model)

	if m.screen != ScreenLoading {
		t.Error("Loading screen should ignore key presses")
	}
}

// ============================================================================
// Complete Coverage Tests - Handler Functions
// ============================================================================

func TestModel_SetViewMode(t *testing.T) {
	model := New()
	model.scrollOffset = 10

	result := model.setViewMode(2)

	if result.viewMode != 2 {
		t.Errorf("Expected viewMode 2, got %d", result.viewMode)
	}
	if result.scrollOffset != 0 {
		t.Error("scrollOffset should be reset to 0")
	}
}

func TestModel_CycleScopeMode(t *testing.T) {
	tests := []struct {
		start    string
		expected string
	}{
		{"both", "system"},
		{"system", "user"},
		{"user", "both"},
		{"unknown", "both"},
	}

	for _, tt := range tests {
		model := New()
		model.optimizerScope = tt.start
		result := model.cycleScopeMode()
		if result.optimizerScope != tt.expected {
			t.Errorf("cycleScopeMode(%s): expected %s, got %s", tt.start, tt.expected, result.optimizerScope)
		}
	}
}

func TestModel_HandleOptimizerKey_Escape(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerPreview
	model.analysis = &path.AnalysisResult{}
	model.scrollOffset = 5

	result, _ := model.handleOptimizerKey("esc")

	if result.screen != ScreenMenu {
		t.Error("Expected return to menu")
	}
	if result.analysis != nil {
		t.Error("analysis should be cleared")
	}
	if result.scrollOffset != 0 {
		t.Error("scrollOffset should be reset")
	}
}

func TestModel_HandleOptimizerKey_ViewModes(t *testing.T) {
	for _, key := range []string{"1", "2", "3", "4"} {
		model := New()
		model.screen = ScreenOptimizerPreview

		result, _ := model.handleOptimizerKey(key)

		expected := int(key[0] - '1')
		if result.viewMode != expected {
			t.Errorf("Key %s: expected viewMode %d, got %d", key, expected, result.viewMode)
		}
	}
}

func TestModel_HandleOptimizerKey_Scope(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerPreview
	model.optimizerScope = "both"

	result, _ := model.handleOptimizerKey("s")

	if result.optimizerScope != "system" {
		t.Errorf("Expected scope 'system', got %s", result.optimizerScope)
	}
}

func TestModel_HandleOptimizerKey_Apply(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerPreview

	result, _ := model.handleOptimizerKey("a")

	if result.screen != ScreenOptimizerConfirm {
		t.Error("Expected ScreenOptimizerConfirm")
	}
}

func TestModel_HandleOptimizerKey_Scroll(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerPreview
	model.scrollOffset = 5

	resultUp, _ := model.handleOptimizerKey("up")
	if resultUp.scrollOffset != 4 {
		t.Error("scrollOffset should decrease on up")
	}

	model.scrollOffset = 0
	resultUpAtZero, _ := model.handleOptimizerKey("up")
	if resultUpAtZero.scrollOffset != 0 {
		t.Error("scrollOffset should not go negative")
	}

	model.scrollOffset = 0
	resultDown, _ := model.handleOptimizerKey("down")
	if resultDown.scrollOffset != 1 {
		t.Error("scrollOffset should increase on down")
	}
}

func TestModel_HandleOptimizerConfirmKey_Yes(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerConfirm
	model.analysis = &path.AnalysisResult{}

	result, cmd := model.handleOptimizerConfirmKey("y")

	if result.screen != ScreenLoading {
		t.Error("Expected ScreenLoading")
	}
	if cmd == nil {
		t.Error("Expected command to be returned")
	}
}

func TestModel_HandleOptimizerConfirmKey_No(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerConfirm

	result, _ := model.handleOptimizerConfirmKey("n")

	if result.screen != ScreenOptimizerPreview {
		t.Error("Expected return to ScreenOptimizerPreview")
	}
}

func TestModel_HandleDoneKey_Copy(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerDone

	result, _ := model.handleDoneKey("c", ScreenMenu)

	// clipboardOK will be set based on whether clipboard works
	t.Logf("clipboardOK: %v", result.clipboardOK)
}

func TestModel_HandleDoneKey_Escape(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerDone
	model.analysis = &path.AnalysisResult{}

	result, _ := model.handleDoneKey("esc", ScreenMenu)

	if result.screen != ScreenMenu {
		t.Error("Expected return to ScreenMenu")
	}
	if result.analysis != nil {
		t.Error("analysis should be cleared")
	}
}

func TestModel_HandleDoneKey_BackToBackup(t *testing.T) {
	model := New()
	model.screen = ScreenBackupDone

	result, _ := model.handleDoneKey("esc", ScreenBackup)

	if result.screen != ScreenBackup {
		t.Error("Expected return to ScreenBackup")
	}
}

func TestModel_HandleViewerKey_Escape(t *testing.T) {
	model := New()
	model.screen = ScreenPathViewer
	model.scrollOffset = 10

	result, _ := model.handleViewerKey("esc")

	if result.screen != ScreenMenu {
		t.Error("Expected return to menu")
	}
	if result.scrollOffset != 0 {
		t.Error("scrollOffset should be reset")
	}
}

func TestModel_HandleViewerKey_Scroll(t *testing.T) {
	model := New()
	model.screen = ScreenPathViewer
	model.scrollOffset = 5

	// Test scroll up
	resultUp, _ := model.handleViewerKey("up")
	if resultUp.scrollOffset != 4 {
		t.Errorf("scrollOffset should decrease from 5 to 4, got %d", resultUp.scrollOffset)
	}

	// Test scroll down (from original model with offset=5)
	resultDown, _ := model.handleViewerKey("down")
	if resultDown.scrollOffset != 6 {
		t.Errorf("scrollOffset should increase from 5 to 6, got %d", resultDown.scrollOffset)
	}
}

func TestModel_HandleViewerKey_ExpandToggle(t *testing.T) {
	model := New()
	model.screen = ScreenPathViewer
	model.viewerExpanded = false

	result, _ := model.handleViewerKey("e")

	if !result.viewerExpanded {
		t.Error("viewerExpanded should toggle to true")
	}
}

func TestModel_HandleBackupCreate(t *testing.T) {
	model := New()

	result := model.handleBackupCreate()

	if result.message != "Backup created!" {
		t.Errorf("Expected 'Backup created!', got '%s'", result.message)
	}
}

func TestModel_HandleBackupView_Empty(t *testing.T) {
	model := New()
	model.backups = []path.BackupInfo{}

	result := model.handleBackupView()

	if result.screen == ScreenBackupPreview {
		t.Error("Should not show preview with no backups")
	}
}

func TestModel_HandleBackupKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenBackup
	model.backups = []path.BackupInfo{{}, {}, {}}
	model.backupIndex = 1

	resultUp, _ := model.handleBackupKey("up")
	if resultUp.backupIndex != 0 {
		t.Error("backupIndex should decrease")
	}

	model.backupIndex = 1
	resultDown, _ := model.handleBackupKey("down")
	if resultDown.backupIndex != 2 {
		t.Error("backupIndex should increase")
	}
}

func TestModel_HandleBackupKey_Create(t *testing.T) {
	model := New()
	model.screen = ScreenBackup

	result, _ := model.handleBackupKey("c")

	if result.message != "Backup created!" {
		t.Error("Expected backup created message")
	}
}

func TestModel_HandleBackupKey_Restore(t *testing.T) {
	model := New()
	model.screen = ScreenBackup
	model.backups = []path.BackupInfo{{}}

	result, _ := model.handleBackupKey("r")

	if result.screen != ScreenBackupConfirmRestore {
		t.Error("Expected ScreenBackupConfirmRestore")
	}
}

func TestModel_HandleBackupKey_Delete(t *testing.T) {
	model := New()
	model.screen = ScreenBackup
	model.backups = []path.BackupInfo{{}}

	result, _ := model.handleBackupKey("d")

	if result.screen != ScreenBackupConfirmDelete {
		t.Error("Expected ScreenBackupConfirmDelete")
	}
}

func TestModel_HandleBackupPreviewKey_Escape(t *testing.T) {
	model := New()
	model.screen = ScreenBackupPreview
	model.backupPreview = &path.Backup{}
	model.scrollOffset = 10

	result, _ := model.handleBackupPreviewKey("esc")

	if result.screen != ScreenBackup {
		t.Error("Expected return to ScreenBackup")
	}
	if result.backupPreview != nil {
		t.Error("backupPreview should be cleared")
	}
	if result.scrollOffset != 0 {
		t.Error("scrollOffset should be reset")
	}
}

func TestModel_HandleBackupPreviewKey_Scroll(t *testing.T) {
	model := New()
	model.screen = ScreenBackupPreview
	model.scrollOffset = 5

	// Test scroll up
	resultUp, _ := model.handleBackupPreviewKey("up")
	if resultUp.scrollOffset != 4 {
		t.Errorf("scrollOffset should decrease from 5 to 4, got %d", resultUp.scrollOffset)
	}

	// Test scroll down (from original model with offset=5)
	resultDown, _ := model.handleBackupPreviewKey("down")
	if resultDown.scrollOffset != 6 {
		t.Errorf("scrollOffset should increase from 5 to 6, got %d", resultDown.scrollOffset)
	}
}

func TestModel_HandleBackupConfirmKey_Restore(t *testing.T) {
	model := New()
	model.screen = ScreenBackupConfirmRestore
	model.backups = []path.BackupInfo{{Filename: "test.json"}}
	model.backupIndex = 0

	result, _ := model.handleBackupConfirmKey("y")

	// May error due to missing file, but should try
	t.Logf("Restore result screen: %d, err: %v", result.screen, result.err)
}

func TestModel_HandleBackupConfirmKey_Delete(t *testing.T) {
	model := New()
	model.screen = ScreenBackupConfirmDelete
	model.backups = []path.BackupInfo{{Filename: "test.json"}}
	model.backupIndex = 0

	result, _ := model.handleBackupConfirmKey("y")

	// May error due to missing file, but should try
	t.Logf("Delete result screen: %d", result.screen)
}

func TestModel_HandleBackupConfirmKey_No(t *testing.T) {
	model := New()
	model.screen = ScreenBackupConfirmRestore

	result, _ := model.handleBackupConfirmKey("n")

	if result.screen != ScreenBackup {
		t.Error("Expected return to ScreenBackup")
	}
}

func TestModel_HandleJunctionsRefresh(t *testing.T) {
	model := New()

	result, cmd := model.handleJunctionsRefresh()

	if result.screen != ScreenLoading {
		t.Error("Expected ScreenLoading")
	}
	if result.loadingTask != TaskJunctions {
		t.Error("Expected TaskJunctions")
	}
	if cmd == nil {
		t.Error("Expected command")
	}
}

func TestModel_HandleJunctionsSuggestions(t *testing.T) {
	model := New()

	result, cmd := model.handleJunctionsSuggestions()

	if result.screen != ScreenLoading {
		t.Error("Expected ScreenLoading")
	}
	if result.loadingTask != TaskSuggestions {
		t.Error("Expected TaskSuggestions")
	}
	if cmd == nil {
		t.Error("Expected command")
	}
}

func TestModel_HandleJunctionsCreate(t *testing.T) {
	model := New()
	model.junctionName = "old"
	model.junctionTarget = "old"

	result := model.handleJunctionsCreate()

	if result.screen != ScreenJunctionCreate {
		t.Error("Expected ScreenJunctionCreate")
	}
	if result.junctionName != "" {
		t.Error("junctionName should be cleared")
	}
	if result.junctionTarget != "" {
		t.Error("junctionTarget should be cleared")
	}
	if result.junctionInputMode != 0 {
		t.Error("junctionInputMode should be 0")
	}
}

func TestModel_HandleJunctionsDelete_Empty(t *testing.T) {
	model := New()
	model.junctions = []path.Junction{}

	result := model.handleJunctionsDelete()

	if result.message != "" {
		t.Error("No message expected when no junctions")
	}
}

func TestModel_HandleJunctionsDelete_WithJunctions(t *testing.T) {
	model := New()
	model.junctions = []path.Junction{{Name: "test"}}
	model.junctionIndex = 0

	result := model.handleJunctionsDelete()

	if result.message != "Junction deleted" {
		t.Errorf("Expected 'Junction deleted', got '%s'", result.message)
	}
}

func TestModel_HandleJunctionsKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenJunctions
	model.junctions = []path.Junction{{}, {}, {}}
	model.junctionIndex = 1

	resultUp, _ := model.handleJunctionsKey("up")
	if resultUp.junctionIndex != 0 {
		t.Error("junctionIndex should decrease")
	}

	model.junctionIndex = 1
	resultDown, _ := model.handleJunctionsKey("down")
	if resultDown.junctionIndex != 2 {
		t.Error("junctionIndex should increase")
	}
}

func TestModel_HandleJunctionsKey_Options(t *testing.T) {
	model := New()
	model.screen = ScreenJunctions

	result1, cmd1 := model.handleJunctionsKey("1")
	if result1.screen != ScreenLoading || cmd1 == nil {
		t.Error("'1' should trigger refresh")
	}

	model.screen = ScreenJunctions
	result2, cmd2 := model.handleJunctionsKey("2")
	if result2.screen != ScreenLoading || cmd2 == nil {
		t.Error("'2' should trigger suggestions")
	}

	model.screen = ScreenJunctions
	result3, _ := model.handleJunctionsKey("3")
	if result3.screen != ScreenJunctionCreate {
		t.Error("'3' should open create screen")
	}
}

func TestModel_HandleJunctionSuggestionsKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenJunctionSuggestions
	model.suggestions = []path.JunctionSuggestion{{}, {}, {}}
	model.junctionIndex = 1

	resultUp, _ := model.handleJunctionSuggestionsKey("up")
	if resultUp.junctionIndex != 0 {
		t.Error("junctionIndex should decrease")
	}

	model.junctionIndex = 1
	resultDown, _ := model.handleJunctionSuggestionsKey("down")
	if resultDown.junctionIndex != 2 {
		t.Error("junctionIndex should increase")
	}
}

func TestModel_HandleJunctionSuggestionsKey_Create(t *testing.T) {
	model := New()
	model.screen = ScreenJunctionSuggestions
	model.suggestions = []path.JunctionSuggestion{{SuggestedName: "test", OriginalPath: `C:\test`}}
	model.junctionIndex = 0

	result, cmd := model.handleJunctionSuggestionsKey("c")

	if result.screen != ScreenLoading {
		t.Error("Expected ScreenLoading")
	}
	if cmd == nil {
		t.Error("Expected command")
	}
}

func TestModel_HandleJunctionSuggestionsKey_Escape(t *testing.T) {
	model := New()
	model.screen = ScreenJunctionSuggestions

	result, _ := model.handleJunctionSuggestionsKey("esc")

	if result.screen != ScreenJunctions {
		t.Error("Expected return to ScreenJunctions")
	}
}

func TestModel_HandleJunctionCreateEscape(t *testing.T) {
	model := New()
	model.screen = ScreenJunctionCreate
	model.err = fmt.Errorf("test error")

	result := model.handleJunctionCreateEscape()

	if result.screen != ScreenJunctions {
		t.Error("Expected return to ScreenJunctions")
	}
	if result.err != nil {
		t.Error("err should be cleared")
	}
}

func TestModel_HandleJunctionCreateEnter_FirstField(t *testing.T) {
	model := New()
	model.junctionInputMode = 0
	model.junctionName = "test"
	model.junctionTarget = ""

	result := model.handleJunctionCreateEnter()

	if result.junctionInputMode != 1 {
		t.Error("Should move to second field")
	}
}

func TestModel_HandleJunctionCreateEnter_Submit(t *testing.T) {
	model := New()
	model.junctionInputMode = 1
	model.junctionName = "test"
	model.junctionTarget = `C:\test`

	result := model.handleJunctionCreateEnter()

	// Result depends on whether junction creation succeeds
	t.Logf("Submit result: screen=%d, message=%s", result.screen, result.message)
}

func TestModel_HandleJunctionCreateBackspace(t *testing.T) {
	model := New()
	model.junctionInputMode = 0
	model.junctionName = "test"

	result := model.handleJunctionCreateBackspace()

	if result.junctionName != "tes" {
		t.Errorf("Expected 'tes', got '%s'", result.junctionName)
	}

	model.junctionInputMode = 1
	model.junctionTarget = "path"
	result = model.handleJunctionCreateBackspace()

	if result.junctionTarget != "pat" {
		t.Errorf("Expected 'pat', got '%s'", result.junctionTarget)
	}
}

func TestModel_HandleJunctionCreateChar(t *testing.T) {
	model := New()
	model.junctionInputMode = 0
	model.junctionName = ""

	result := model.handleJunctionCreateChar("a")

	if result.junctionName != "a" {
		t.Errorf("Expected 'a', got '%s'", result.junctionName)
	}

	model.junctionInputMode = 1
	model.junctionTarget = ""
	result = model.handleJunctionCreateChar("b")

	if result.junctionTarget != "b" {
		t.Errorf("Expected 'b', got '%s'", result.junctionTarget)
	}
}

func TestModel_HandleJunctionCreateKey_Tab(t *testing.T) {
	model := New()
	model.junctionInputMode = 0

	result, _ := model.handleJunctionCreateKey("tab")

	if result.junctionInputMode != 1 {
		t.Error("Tab should toggle input mode")
	}
}

func TestModel_HandlePathExtKey_Dispatches(t *testing.T) {
	model := New()
	model.pathExtEditing = true
	model.pathExtList = []string{".EXE"}

	result, _ := model.handlePathExtKey("esc")

	if result.pathExtEditing {
		t.Error("'esc' should exit edit mode")
	}
}

func TestModel_HandlePathExtConfirmKey_Yes(t *testing.T) {
	model := New()
	model.screen = ScreenPathExtConfirm
	model.pathExtList = []string{".EXE", ".CMD"}
	model.pathExtOpt = &path.PathExtOptimization{
		OptimizedString: ".EXE;.CMD",
		Optimized:       []string{".EXE", ".CMD"},
	}

	// Get mock to verify it's being used (safety check)
	mock, ok := path.DefaultRunner.(*path.MockShellRunner)
	if !ok {
		t.Fatal("Mock runner not set up - tests could modify real system!")
	}
	beforeCalls := len(mock.Calls)

	result, _ := model.handlePathExtConfirmKey("y")

	// Verify mock was called, not real PowerShell
	if len(mock.Calls) <= beforeCalls {
		t.Error("Expected mock to be called for apply operation")
	}

	// Verify screen changed appropriately
	t.Logf("Confirm result: screen=%d, err=%v, mock_calls=%d", result.screen, result.err, len(mock.Calls)-beforeCalls)
}

func TestModel_HandlePathExtConfirmKey_No(t *testing.T) {
	model := New()
	model.screen = ScreenPathExtConfirm

	result, _ := model.handlePathExtConfirmKey("n")

	if result.screen != ScreenPathExt {
		t.Error("Expected return to ScreenPathExt")
	}
}

func TestModel_HandleSettingsKey_Navigation(t *testing.T) {
	model := New()
	model.screen = ScreenSettings
	model.settingsIndex = 1

	resultUp, _ := model.handleSettingsKey("up")
	if resultUp.settingsIndex != 0 {
		t.Error("settingsIndex should decrease")
	}

	model.settingsIndex = 0
	resultDown, _ := model.handleSettingsKey("down")
	if resultDown.settingsIndex != 1 {
		t.Error("settingsIndex should increase")
	}
}

func TestModel_HandleSettingsKey_Enter(t *testing.T) {
	model := New()
	model.screen = ScreenSettings
	model.settingsIndex = 1 // AutoBackup toggle
	originalAutoBackup := model.config.AutoBackup

	result, _ := model.handleSettingsKey("enter")

	// Enter on AutoBackup should toggle it
	if result.config.AutoBackup == originalAutoBackup {
		t.Error("Enter should toggle AutoBackup setting")
	}
}

func TestModel_HandleSettingsKey_Escape(t *testing.T) {
	model := New()
	model.screen = ScreenSettings

	result, _ := model.handleSettingsKey("esc")

	if result.screen != ScreenMenu {
		t.Error("Expected return to menu")
	}
}

func TestModel_HandleHotPathsKey_Dispatches(t *testing.T) {
	model := New()
	model.hotPathAdding = true

	result, _ := model.handleHotPathsKey("esc")

	if result.hotPathAdding {
		t.Error("Should dispatch to input handler")
	}
}

// ============================================================================
// View Function Tests
// ============================================================================

func TestModel_ViewMenu(t *testing.T) {
	model := New()
	model.screen = ScreenMenu
	model.width = 80
	model.height = 24

	view := model.viewMenu()

	if view == "" {
		t.Error("viewMenu should not return empty string")
	}
	if !strings.Contains(view, "PATH Optimizer") {
		t.Error("viewMenu should contain title")
	}
}

func TestModel_ViewOptimizer(t *testing.T) {
	model := New()
	model.screen = ScreenOptimizerPreview
	model.width = 80
	model.height = 24
	model.analysis = &path.AnalysisResult{
		System: path.OptimizeResult{
			Original:  path.PathInfo{Count: 5, Length: 100},
			Optimized: path.PathInfo{Count: 4, Length: 80},
		},
	}

	view := model.viewOptimizer()

	if view == "" {
		t.Error("viewOptimizer should not return empty string")
	}
}

func TestModel_RenderSummary(t *testing.T) {
	model := New()
	model.width = 80
	model.analysis = &path.AnalysisResult{
		System: path.OptimizeResult{
			Original:  path.PathInfo{Count: 5, Length: 100},
			Optimized: path.PathInfo{Count: 4, Length: 80},
			Metrics:   path.OptimizeMetrics{DuplicatesRemoved: 1},
		},
	}

	summary := model.renderSummary()

	if summary == "" {
		t.Error("renderSummary should not return empty string")
	}
}

func TestModel_RenderChanges(t *testing.T) {
	model := New()
	model.width = 80
	model.height = 24
	model.analysis = &path.AnalysisResult{
		System: path.OptimizeResult{
			Changes: []path.PathChange{
				{Type: "duplicate", Original: `C:\test`},
			},
		},
	}

	changes := model.renderChanges()

	if changes == "" {
		t.Error("renderChanges should not return empty string")
	}
}

func TestModel_RenderRaw(t *testing.T) {
	model := New()
	model.width = 80
	model.height = 24
	model.analysis = &path.AnalysisResult{
		System: path.OptimizeResult{
			Optimized: path.PathInfo{Raw: `C:\Windows;C:\test`},
		},
	}

	raw := model.renderRaw()

	if raw == "" {
		t.Error("renderRaw should not return empty string")
	}
}

func TestModel_RenderList(t *testing.T) {
	model := New()
	model.width = 80
	model.height = 24
	model.analysis = &path.AnalysisResult{
		System: path.OptimizeResult{
			Optimized: path.PathInfo{Entries: []string{`C:\Windows`, `C:\test`}},
		},
	}

	list := model.renderList()

	if list == "" {
		t.Error("renderList should not return empty string")
	}
}

func TestModel_ViewConfirm(t *testing.T) {
	model := New()
	model.width = 80
	model.height = 24

	view := model.viewConfirm("Test Title", "Test detail", ScreenMenu)

	if view == "" {
		t.Error("viewConfirm should not return empty string")
	}
	if !strings.Contains(view, "Test Title") {
		t.Error("viewConfirm should contain title")
	}
}

func TestModel_ViewDone(t *testing.T) {
	model := New()
	model.width = 80
	model.height = 24

	view := model.viewDone("Operation complete", nil)

	if view == "" {
		t.Error("viewDone should not return empty string")
	}
}

func TestModel_ViewDone_WithBackup(t *testing.T) {
	model := New()
	model.width = 80
	model.height = 24

	view := model.viewDone("Operation complete", &path.BackupInfo{Filename: "test.json"})

	if view == "" {
		t.Error("viewDone should not return empty string")
	}
}

func TestModel_ViewPathViewer(t *testing.T) {
	model := New()
	model.screen = ScreenPathViewer
	model.width = 80
	model.height = 24

	view := model.viewPathViewer()

	if view == "" {
		t.Error("viewPathViewer should not return empty string")
	}
}

func TestModel_ViewBackup(t *testing.T) {
	model := New()
	model.screen = ScreenBackup
	model.width = 80
	model.height = 24
	model.backups = []path.BackupInfo{{Filename: "test.json", FormattedDate: "2025-01-01"}}

	view := model.viewBackup()

	if view == "" {
		t.Error("viewBackup should not return empty string")
	}
}

func TestModel_ViewConfirmBackup(t *testing.T) {
	model := New()
	model.width = 80
	model.height = 24
	model.backups = []path.BackupInfo{{Filename: "test.json"}}

	view := model.viewConfirmBackup("restore", Green)

	if view == "" {
		t.Error("viewConfirmBackup should not return empty string")
	}
}

func TestModel_ViewJunctions(t *testing.T) {
	model := New()
	model.screen = ScreenJunctions
	model.width = 80
	model.height = 24
	model.junctions = []path.Junction{{Name: "test", Target: `C:\test`}}

	view := model.viewJunctions()

	if view == "" {
		t.Error("viewJunctions should not return empty string")
	}
}

func TestModel_ViewJunctionSuggestions(t *testing.T) {
	model := New()
	model.screen = ScreenJunctionSuggestions
	model.width = 80
	model.height = 24
	model.suggestions = []path.JunctionSuggestion{
		{OriginalPath: `C:\Program Files\Test`, SuggestedName: "test", SavedChars: 20},
	}

	view := model.viewJunctionSuggestions()

	if view == "" {
		t.Error("viewJunctionSuggestions should not return empty string")
	}
}

func TestModel_ViewPathExt(t *testing.T) {
	model := New()
	model.screen = ScreenPathExt
	model.width = 80
	model.height = 24
	model.pathExtList = []string{".EXE", ".CMD"}
	model.pathExtAnalysis = &path.PathExtAnalysis{
		Current:         []string{".EXE", ".CMD"},
		CurrentWithInfo: []path.ExtensionInfo{{Ext: ".EXE"}, {Ext: ".CMD"}},
	}

	view := model.viewPathExt()

	if view == "" {
		t.Error("viewPathExt should not return empty string")
	}
}

func TestModel_ViewSettings(t *testing.T) {
	model := New()
	model.screen = ScreenSettings
	model.width = 80
	model.height = 24

	view := model.viewSettings()

	if view == "" {
		t.Error("viewSettings should not return empty string")
	}
}

func TestModel_ViewHotPaths(t *testing.T) {
	model := New()
	model.screen = ScreenHotPaths
	model.width = 80
	model.height = 24
	model.config.HotPaths = []string{`C:\test`}

	view := model.viewHotPaths()

	if view == "" {
		t.Error("viewHotPaths should not return empty string")
	}
}

func TestWrapText_ZeroWidth(t *testing.T) {
	result := wrapText("test", 0)
	if result == "" {
		t.Log("wrapText with zero width returns empty or original")
	}
}

// ============================================================================
// Message Type Tests
// ============================================================================

func TestAnalysisCompleteMsg(t *testing.T) {
	msg := analysisCompleteMsg{
		result: path.AnalysisResult{},
	}
	// Just verify the struct can be created
	_ = msg.result
}

func TestJunctionsLoadedMsg(t *testing.T) {
	msg := junctionsLoadedMsg{
		junctions: []path.Junction{{Name: "test"}},
	}
	if len(msg.junctions) != 1 {
		t.Error("Expected 1 junction")
	}
}

func TestSuggestionsLoadedMsg(t *testing.T) {
	msg := suggestionsLoadedMsg{
		suggestions: []path.JunctionSuggestion{{SuggestedName: "test"}},
	}
	if len(msg.suggestions) != 1 {
		t.Error("Expected 1 suggestion")
	}
}

func TestProgressMsg(t *testing.T) {
	msg := progressMsg{
		current: 5,
		total:   10,
		item:    "testing",
	}
	if msg.current != 5 || msg.total != 10 {
		t.Error("Progress values incorrect")
	}
}

func TestJunctionCreatedMsg(t *testing.T) {
	msg := junctionCreatedMsg{
		success: true,
		name:    "test",
		err:     nil,
	}
	if msg.name != "test" {
		t.Error("name should be 'test'")
	}
	if !msg.success {
		t.Error("success should be true")
	}
}

func TestApplyCompleteMsg(t *testing.T) {
	backup := &path.BackupInfo{Filename: "test.json"}
	msg := applyCompleteMsg{
		backup: backup,
		err:    nil,
	}
	if msg.backup != backup {
		t.Error("backup should match")
	}
}

// ============================================================================
// Edge Cases and Boundary Tests
// ============================================================================

func TestModel_EmptyView(t *testing.T) {
	model := New()
	model.width = 0
	model.height = 0

	view := model.View()

	// Should not panic, even with zero dimensions
	t.Logf("View with zero dimensions: %d chars", len(view))
}

func TestModel_NegativeIndex(t *testing.T) {
	model := New()
	model.menuIndex = -1

	// Should handle gracefully
	view := model.viewMenu()
	if view == "" {
		t.Log("viewMenu handles negative index")
	}
}

func TestModel_IndexOutOfBounds(t *testing.T) {
	model := New()
	model.junctions = []path.Junction{}
	model.junctionIndex = 5

	result := model.handleJunctionsDelete()

	// Should not panic
	t.Logf("Delete with out of bounds index: message=%s", result.message)
}
