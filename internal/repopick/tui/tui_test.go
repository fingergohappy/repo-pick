package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/finger/repo-pick/internal/repopick/app"
	"github.com/finger/repo-pick/internal/repopick/cache"
	"github.com/finger/repo-pick/internal/repopick/config"
	"github.com/finger/repo-pick/internal/repopick/install"
	"github.com/finger/repo-pick/internal/repopick/registry"
)

func TestModelStartsWithRegistryFocusAndLoadsRepositories(t *testing.T) {
	service := testService(t, createWorktree(t), config.Config{
		Repositories: []config.Repository{{Name: "official", URL: "repo"}},
	})
	m := newModel(context.Background(), service, "/tmp/session")

	if m.focus != focusRegistry {
		t.Fatalf("focus = %v, want registry", m.focus)
	}
	if m.sessionCWD != "/tmp/session" {
		t.Fatalf("sessionCWD = %q, want /tmp/session", m.sessionCWD)
	}

	cmd := m.Init()
	m = updateModel(t, m, cmd())

	if len(m.repositories) != 1 || m.repositories[0].Name != "official" {
		t.Fatalf("repositories = %#v, want official", m.repositories)
	}
}

func TestEmptyRegistryUsesDesignedPlaceholder(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())

	lines := strings.Join(plainBlock(m.registryLinesView()), "\n")

	if !strings.Contains(lines, "暂无 registry") || !strings.Contains(lines, "添加 registry") {
		t.Fatalf("registryLines() = %q, want designed empty placeholder", lines)
	}
	if !strings.Contains(lines, "╭") || !strings.Contains(lines, "╰") {
		t.Fatalf("registryLines() = %q, want rounded empty placeholder", lines)
	}
	if !strings.Contains(lines, "a") || !strings.Contains(lines, "添加 registry") {
		t.Fatalf("registryLines() = %q, want add registry CTA", lines)
	}
	if strings.Contains(lines, "a add") {
		t.Fatalf("registryLines() = %q, should not use old empty placeholder", lines)
	}
}

func TestPaneTitleIsCenteredWithDivider(t *testing.T) {
	width := 24

	title := plainText(renderPaneTitleLine("Registry (1)", width, true))
	if strings.Contains(title, ">") {
		t.Fatalf("paneTitleLine() = %q, should not include selection cursor", title)
	}
	if !strings.Contains(title, "Registry (1)") || !strings.HasPrefix(title, "      ") {
		t.Fatalf("paneTitleLine() = %q, want centered title", title)
	}

	divider := plainText(renderPaneSeparatorLine(width))
	if lipgloss.Width(divider) != width || !strings.Contains(divider, "─") {
		t.Fatalf("paneTitleDividerLine() = %q, want full-width divider", divider)
	}
}

func TestRegistryLinesShowLastUpdatedAt(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.repositories = []config.Repository{
		{Name: "official", URL: "repo", LastUpdatedAt: "1999-01-02T03:04:05Z"},
		{Name: "personal", URL: "repo2"},
	}

	lines := strings.Join(plainBlock(m.registryLinesView()), "\n")

	if !strings.Contains(lines, "official") || !strings.Contains(lines, "1999") {
		t.Fatalf("registryLines() = %q, want last updated year", lines)
	}
	if !strings.Contains(lines, "personal") || !strings.Contains(lines, "-") {
		t.Fatalf("registryLines() = %q, want empty updated marker", lines)
	}
}

func TestDeleteRepositoryShowsConfirmModalBeforeRemoving(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{Name: "official", URL: "https://github.com/org/tools", Branch: "main"}},
	}}
	cacheSvc := &fakeCache{worktree: cache.Worktree{Dir: createWorktree(t)}}
	service := app.Service{
		Registry:  registry.NewService(store),
		Cache:     cacheSvc,
		Installer: install.Installer{},
	}
	m := newModel(context.Background(), service, t.TempDir())
	m.repositories = store.cfg.Repositories

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = next.(model)
	if cmd != nil || m.mode != modeConfirmDelete {
		t.Fatalf("cmd/mode = %v/%v, want confirm delete without command", cmd, m.mode)
	}
	view := plainText(m.View())
	if !strings.Contains(view, "删除 Registry") || !strings.Contains(view, "official") || !strings.Contains(view, "y 确认删除") {
		t.Fatalf("View() = %q, want delete confirm modal", view)
	}
	if len(store.cfg.Repositories) != 1 {
		t.Fatalf("repositories = %#v, want unchanged before confirm", store.cfg.Repositories)
	}

	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = next.(model)
	if cmd == nil || m.mode != modeNormal {
		t.Fatalf("cmd/mode = %v/%v, want remove command and normal mode", cmd, m.mode)
	}
	m = updateModel(t, m, cmd())

	if len(store.cfg.Repositories) != 0 {
		t.Fatalf("repositories = %#v, want removed after confirm", store.cfg.Repositories)
	}
	if len(cacheSvc.deleted) != 1 || cacheSvc.deleted[0].Name != "official" {
		t.Fatalf("deleted cache = %#v, want official cache deleted", cacheSvc.deleted)
	}
}

func TestHelpDialogIsCenteredInView(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.width = 100
	m.height = 30
	m.showHelp = true

	lines := plainBlock(m.View())
	modalTop := -1
	for i, line := range lines {
		if strings.Contains(line, "╭") {
			modalTop = i
			break
		}
	}

	if modalTop <= 0 {
		t.Fatalf("help modal top = %d in %#v, want vertical centering", modalTop, lines)
	}
	modalLine := lines[modalTop]
	if !strings.HasPrefix(modalLine, " ") {
		t.Fatalf("help modal line = %q, want horizontal centering", modalLine)
	}
	if width := lipgloss.Width(modalLine); width != m.width {
		t.Fatalf("help modal line width = %d, want full view width %d", width, m.width)
	}
}

func TestCtrlWLFocusesRepositoryTreeAndOpensSelectedRepository(t *testing.T) {
	worktree := createWorktree(t)
	service := testService(t, worktree, config.Config{
		Repositories: []config.Repository{{Name: "official", URL: "repo"}},
	})
	m := newModel(context.Background(), service, t.TempDir())
	m.repositories = []config.Repository{{Name: "official", URL: "repo"}}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	m = next.(model)
	if cmd != nil {
		t.Fatalf("ctrl-w command = %v, want nil", cmd)
	}
	if !m.pendingWindowCommand {
		t.Fatalf("pendingWindowCommand = false, want true")
	}

	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("ctrl-w l command = nil, want open repository command")
	}
	treeLoading := strings.Join(plainBlock(m.treeLinesView()), "\n")
	if m.operationKind != operationOpen || !strings.Contains(treeLoading, "loading repo cache: official") || strings.Contains(plainText(m.statusLine()), "loading repo cache") {
		t.Fatalf("operation/tree/status = %v/%q/%q, want open progress in tree only", m.operationKind, treeLoading, plainText(m.statusLine()))
	}
	m = updateModel(t, m, operationProgressMsg{kind: operationOpen, baseLabel: "loading repo cache: official", event: app.ProgressEvent{Text: "Receiving objects: 50%", Percent: 50}})
	if treeLoading = strings.Join(plainBlock(m.treeLinesView()), "\n"); !strings.Contains(treeLoading, "50%") {
		t.Fatalf("treeLines() = %q, want bubbles progress percentage", treeLoading)
	}
	m = runOperationBatch(t, m, cmd)

	if m.focus != focusTree || !m.repoOpened {
		t.Fatalf("focus/repoOpened = %v/%v, want tree/open", m.focus, m.repoOpened)
	}
	if len(m.entries) == 0 || m.entries[0].Path != "docs" {
		t.Fatalf("entries = %#v, want docs first", m.entries)
	}
}

func TestCtrlWHFocusesRegistry(t *testing.T) {
	service := testService(t, createWorktree(t), config.Config{})
	m := openModelWithWorktree(t, service)

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlW})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})

	if m.focus != focusRegistry || m.pendingWindowCommand {
		t.Fatalf("focus/pending = %v/%v, want registry/false", m.focus, m.pendingWindowCommand)
	}
}

func TestStatusLineUsesFocusedPaneHelp(t *testing.T) {
	service := testService(t, createWorktree(t), config.Config{})
	m := openModelWithWorktree(t, service)
	m.width = 200

	treeStatus := m.statusLine()
	for _, want := range []string{"[l]expand", "[h]parent", "[o]root", "[e]editor", "[i]download", "[I]target", "[/]search", "[Tab]registry"} {
		if !strings.Contains(treeStatus, want) {
			t.Fatalf("tree statusLine() = %q, want %q", treeStatus, want)
		}
	}
	if strings.Contains(treeStatus, "[a]add") || strings.Contains(treeStatus, "[d]delete") {
		t.Fatalf("tree statusLine() = %q, want tree help", treeStatus)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlW})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	registryStatus := m.statusLine()
	for _, want := range []string{"[l]open", "[a]add", "[e]edit", "[r]reload", "[d]delete", "[u]update", "[Tab]tree"} {
		if !strings.Contains(registryStatus, want) {
			t.Fatalf("registry statusLine() = %q, want %q", registryStatus, want)
		}
	}
	if strings.Contains(registryStatus, "[i]download") || strings.Contains(registryStatus, "[h]parent") {
		t.Fatalf("registry statusLine() = %q, want registry help", registryStatus)
	}
}

func TestTabSwitchesFocusAndArrowKeysMoveSelection(t *testing.T) {
	service := testService(t, createWorktree(t), config.Config{})
	m := openModelWithWorktree(t, service)
	m.repositories = []config.Repository{
		{Name: "official", URL: "repo"},
		{Name: "personal", URL: "repo2"},
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if m.focus != focusRegistry {
		t.Fatalf("focus = %v, want registry after tab from tree", m.focus)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.selectedRepo != 1 {
		t.Fatalf("selectedRepo = %d, want 1 after down", m.selectedRepo)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.selectedRepo != 0 {
		t.Fatalf("selectedRepo = %d, want 0 after up", m.selectedRepo)
	}
}

func TestHelpViewUsesDesignedOverlay(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.showHelp = true

	view := plainText(m.View())

	for _, want := range []string{"快捷键帮助", "通用", "Registry", "Repository Tree", "Tab", "Enter 或 l"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() = %q, want %q", view, want)
		}
	}
}

func TestNarrowViewRequiresMinimumUsableTerminal(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.width = minTerminalWidth - 1
	m.height = minTerminalHeight

	view := plainText(m.View())

	if !strings.Contains(view, "terminal too small") || !strings.Contains(view, "80x24") {
		t.Fatalf("View() = %q, want terminal size gate", view)
	}
}

func TestTreeContextShowsRepositoryMetadata(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.width = 120
	m.repoOpened = true
	m.openedRepo = config.Repository{
		Name:   "aa",
		URL:    "https://github.com/anthropics/skills.git",
		Branch: "main",
	}
	m.currentPath = "docs"

	lines := strings.Join(plainBlock(m.treeContextLinesView()), "\n")

	for _, want := range []string{"registry", "aa", "url", "https://github.com/anthropics/skills.git", "branch", "main", "path", "/docs"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("treeContextLinesView() = %q, want %q", lines, want)
		}
	}
}

// TestStatusLineSummarizesMultilineError 验证多行错误只在状态栏展示摘要。
func TestStatusLineSummarizesMultilineError(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.width = 32
	m.err = errors.New("git clone failed\nstderr: very long git output")

	status := plainText(m.statusLine())

	if strings.Contains(status, "\n") || strings.Contains(status, "stderr") {
		t.Fatalf("statusLine() = %q, want first-line error summary", status)
	}
	if len(status) > m.width {
		t.Fatalf("statusLine() width = %d, want <= %d: %q", len(status), m.width, status)
	}
}

// TestRegistryLinesWindowAroundSelectedRepo 验证 registry 列表会围绕当前光标裁剪。
func TestRegistryLinesWindowAroundSelectedRepo(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.height = 10
	for i := 0; i < 40; i++ {
		m.repositories = append(m.repositories, config.Repository{Name: fmt.Sprintf("repo-%02d", i), URL: fmt.Sprintf("repo-%02d", i)})
	}
	m.selectedRepo = 30

	lines := strings.Join(plainBlock(m.registryLinesView()), "\n")

	if !strings.Contains(lines, "repo-30") {
		t.Fatalf("registryLines() = %q, want selected repo in window", lines)
	}
	if strings.Contains(lines, "repo-00") {
		t.Fatalf("registryLines() = %q, want clipped list window", lines)
	}
	if got, wantMax := len(plainBlock(m.registryLinesView())), m.paneItemRows(); got > wantMax {
		t.Fatalf("registryLines() length = %d, want <= %d", got, wantMax)
	}
}

func TestRegistrySelectionShowsAnimatedPreview(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.focus = focusRegistry
	m.repositories = []config.Repository{
		{Name: "official", URL: "repo"},
		{Name: "personal", URL: "repo2", Branch: "dev"},
	}
	m.openedRepo = m.repositories[0]
	m.repoOpened = true
	m.resetTreeRoot("", []app.EntryResult{{Name: "README.md", Path: "README.md", Type: app.EntryFile, Size: 6}})

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("registry selection animation command = nil")
	}

	lines := strings.Join(plainBlock(m.treeLinesView()), "\n")
	if !strings.Contains(lines, "已选择 registry") || !strings.Contains(lines, "personal [dev]") || !strings.Contains(lines, "repo2") {
		t.Fatalf("treeLines() = %q, want selected registry preview", lines)
	}
	if strings.Contains(lines, "README.md") {
		t.Fatalf("treeLines() = %q, should hide old opened tree while selection differs", lines)
	}

	m = updateModel(t, m, registrySelectionTickMsg{selectionID: m.registrySelectionID})
	if m.registrySelectionFrame != 1 {
		t.Fatalf("registrySelectionFrame = %d, want 1", m.registrySelectionFrame)
	}
}

func TestRegistrySelectionReturnsToOpenedRegistryPreview(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.focus = focusRegistry
	m.repositories = []config.Repository{
		{Name: "aa", URL: "repo", Branch: "main"},
		{Name: "bb", URL: "repo2", Branch: "dev"},
	}
	m.selectedRepo = 0
	m.openedRepo = m.repositories[0]
	m.repoOpened = true
	m.currentPath = ""
	m.resetTreeRoot("", []app.EntryResult{{Name: "README.md", Path: "README.md", Type: app.EntryFile, Size: 6}})

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if lines := strings.Join(plainBlock(m.treeLinesView()), "\n"); !strings.Contains(lines, "已选择 registry") || strings.Contains(lines, "README.md") {
		t.Fatalf("treeLines() = %q, want registry preview after moving away", lines)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	lines := strings.Join(plainBlock(m.treeLinesView()), "\n")
	if !strings.Contains(lines, "已选择 registry") || !strings.Contains(lines, "aa [main]") || !strings.Contains(lines, "按 l 打开该 repository") {
		t.Fatalf("treeLines() = %q, want registry preview after returning to opened repo", lines)
	}
	if strings.Contains(lines, "README.md") {
		t.Fatalf("treeLines() = %q, should keep registry preview instead of opened tree after j/k movement", lines)
	}
}

func TestRegistryRefreshReloadsRepositories(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{Name: "official", URL: "repo"}},
	}}
	service := app.Service{
		Registry:  registry.NewService(store),
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: createWorktree(t)}},
		Installer: install.Installer{},
	}
	m := newModel(context.Background(), service, t.TempDir())

	store.cfg.Repositories = []config.Repository{{Name: "personal", URL: "repo2"}}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("refresh command = nil")
	}
	m = updateModel(t, m, cmd())

	if len(m.repositories) != 1 || m.repositories[0].Name != "personal" {
		t.Fatalf("repositories = %#v, want refreshed personal", m.repositories)
	}
}

func TestTreeVimNavigationEntersAndLeavesDirectory(t *testing.T) {
	worktree := createWorktree(t)
	service := testService(t, worktree, config.Config{})
	m := openModelWithWorktree(t, service)

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("o command = nil, want list directory command")
	}
	m = updateModel(t, m, cmd())
	if m.currentPath != "docs" {
		t.Fatalf("currentPath = %q, want docs", m.currentPath)
	}
	if len(m.entries) != 1 || m.entries[0].Path != "docs/guide.md" {
		t.Fatalf("entries = %#v, want guide", m.entries)
	}

	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("h command = nil, want parent list command")
	}
	m = updateModel(t, m, cmd())
	if m.currentPath != "" {
		t.Fatalf("currentPath = %q, want root", m.currentPath)
	}
}

func TestSearchShowsRepositoryPathResults(t *testing.T) {
	worktree := createWorktree(t)
	service := testService(t, worktree, config.Config{})
	m := openModelWithWorktree(t, service)

	m.mode = modeSearch
	m.input.SetValue("guide")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("search command = nil")
	}
	m = updateModel(t, m, cmd())

	if !m.showingSearch {
		t.Fatalf("showingSearch = false, want true")
	}
	if len(m.searchResults) != 1 || m.searchResults[0].Path != "docs/guide.md" {
		t.Fatalf("searchResults = %#v, want guide", m.searchResults)
	}
	lines := plainBlock(m.treeLinesView())
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "search") || !strings.Contains(text, "guide") {
		t.Fatalf("treeLines() = %#v, want search context", lines)
	}
	if !strings.Contains(text, "---") || !strings.Contains(text, "type") {
		t.Fatalf("treeLines() = %#v, want separator and content header", lines)
	}
	if !strings.Contains(text, "docs/guide.md") {
		t.Fatalf("treeLines() = %#v, want search result below content header", lines)
	}
}

// TestStaleSearchResultIsIgnored 验证过期搜索结果不会覆盖当前视图。
func TestStaleSearchResultIsIgnored(t *testing.T) {
	service := testService(t, createWorktree(t), config.Config{})
	m := openModelWithWorktree(t, service)
	m.searchRequestID = 2

	m = updateModel(t, m, searchResultMsg{
		requestID:  1,
		repository: m.openedRepo,
		query:      "old",
		entries:    []app.EntryResult{{Name: "old.md", Path: "old.md", Type: app.EntryFile}},
	})

	if m.showingSearch || len(m.searchResults) != 0 {
		t.Fatalf("search state = %v/%#v, want stale result ignored", m.showingSearch, m.searchResults)
	}
}

func TestSearchInputRendersAboveFileList(t *testing.T) {
	worktree := createWorktree(t)
	service := testService(t, worktree, config.Config{})
	m := openModelWithWorktree(t, service)

	m.mode = modeSearch
	m.input.SetValue("guide")
	lines := plainBlock(m.treeLinesView())
	text := strings.Join(lines, "\n")

	if !strings.Contains(text, "search") || !strings.Contains(text, "guide") {
		t.Fatalf("treeLines() = %#v, want search input in context area", lines)
	}
	if !strings.Contains(text, "---") || strings.Contains(text, "repository contents") {
		t.Fatalf("treeLines() = %#v, want separator without repository contents header", lines)
	}
}

func TestTreeToggleOpensDirectoryWithL(t *testing.T) {
	worktree := createWorktree(t)
	service := testService(t, worktree, config.Config{})
	m := openModelWithWorktree(t, service)

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("l command = nil, want load children command")
	}
	m = updateModel(t, m, cmd())

	entries := m.visibleEntries()
	if len(entries) < 3 || entries[0].Path != "" || entries[1].Path != "docs" || entries[2].Path != "docs/guide.md" {
		t.Fatalf("visibleEntries() = %#v, want expanded docs tree", entries)
	}
	lines := plainBlock(m.treeLinesView())
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "├── ▾ docs/") || !strings.Contains(text, "│   └── • guide.md") {
		t.Fatalf("treeLines() = %#v, want expanded directory and child", lines)
	}
}

func TestTreeCollapseAllWithC(t *testing.T) {
	worktree := createWorktree(t)
	service := testService(t, worktree, config.Config{})
	m := openModelWithWorktree(t, service)

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = next.(model)
	m = updateModel(t, m, cmd())

	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	m = next.(model)
	if cmd != nil {
		t.Fatalf("C command = %v, want nil", cmd)
	}
	if len(m.expandedPaths) != 0 {
		t.Fatalf("expandedPaths = %#v, want empty after collapse all", m.expandedPaths)
	}
	entries := m.visibleEntries()
	for _, entry := range entries {
		if entry.Path == "docs/guide.md" {
			t.Fatalf("visibleEntries() = %#v, want expanded children hidden", entries)
		}
	}
	if m.status != "已收起所有目录" {
		t.Fatalf("status = %q, want collapse all message", m.status)
	}
}

func TestTreeRootIsSelectableAndDownloadsRepository(t *testing.T) {
	worktree := createWorktree(t)
	sessionCWD := t.TempDir()
	service := testService(t, worktree, config.Config{})
	m := openModelWithWorktree(t, service)
	m.sessionCWD = sessionCWD

	entries := m.visibleEntries()
	if len(entries) == 0 || entries[0].Path != "" || entries[0].Type != app.EntryDir {
		t.Fatalf("visibleEntries() = %#v, want repository root first", entries)
	}
	lines := strings.Join(plainBlock(m.treeLinesView()), "\n")
	if !strings.Contains(lines, "> /") || strings.Contains(lines, "repository root") {
		t.Fatalf("treeLines() = %q, want selected root row", lines)
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("download command = nil")
	}
	if m.operationKind != operationDownload || !strings.Contains(plainText(m.statusLine()), "downloading official") {
		t.Fatalf("operation/status = %v/%q, want repository download progress", m.operationKind, plainText(m.statusLine()))
	}
	m = runOperationBatch(t, m, cmd)

	assertFileContent(t, filepath.Join(sessionCWD, "official", "README.md"), "readme\n")
}

func TestSelectedLineUsesAnimatedCursorWithoutReverseBackground(t *testing.T) {
	service := testService(t, createWorktree(t), config.Config{})
	m := openModelWithWorktree(t, service)

	rawLine := m.treeLinesView()
	if strings.Contains(rawLine, "[7m") || strings.Contains(rawLine, ";7m") {
		t.Fatalf("treeLines() = %q, should not use reverse background for selected row", rawLine)
	}

	m = updateModel(t, m, selectionCursorTickMsg{})
	lines := strings.Join(plainBlock(m.treeLinesView()), "\n")
	if !strings.Contains(lines, "› /") {
		t.Fatalf("treeLines() = %q, want animated selected cursor", lines)
	}
}

func TestTreeEditFileRequiresEditor(t *testing.T) {
	t.Setenv("EDITOR", "")
	service := testService(t, createWorktree(t), config.Config{})
	m := openModelWithWorktree(t, service)
	m.selectedEntry = 2

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = next.(model)

	if cmd != nil || m.status != "EDITOR 未设置" {
		t.Fatalf("cmd/status = %v/%q, want EDITOR warning without command", cmd, m.status)
	}
}

func TestTreeEditFileUsesEditorCommand(t *testing.T) {
	t.Setenv("EDITOR", "true")
	service := testService(t, createWorktree(t), config.Config{})
	m := openModelWithWorktree(t, service)
	m.selectedEntry = 2

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = next.(model)

	if cmd == nil || !strings.Contains(m.status, "正在用 editor 打开 README.md") {
		t.Fatalf("cmd/status = %v/%q, want editor command", cmd, m.status)
	}
}

func TestDownloadExistingTargetOpensOverwriteConfirm(t *testing.T) {
	worktree := createWorktree(t)
	sessionCWD := t.TempDir()
	if err := os.WriteFile(filepath.Join(sessionCWD, "README.md"), []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("write existing target: %v", err)
	}
	service := testService(t, worktree, config.Config{})
	m := openModelWithWorktree(t, service)
	m.sessionCWD = sessionCWD
	m.selectedEntry = 2

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("download command = nil")
	}
	if m.operationKind != operationDownload || !strings.Contains(plainText(m.statusLine()), "downloading README.md") {
		t.Fatalf("operation/status = %v/%q, want download progress", m.operationKind, plainText(m.statusLine()))
	}
	m = runOperationBatch(t, m, cmd)

	if m.mode != modeConfirmOverwrite || m.pendingDownload == nil {
		t.Fatalf("mode/pendingDownload = %v/%v, want overwrite confirm", m.mode, m.pendingDownload)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.mode != modeNormal {
		t.Fatalf("mode = %v, want normal after confirm", m.mode)
	}
}

func TestUpdateRepositoryShowsProgress(t *testing.T) {
	service := testService(t, createWorktree(t), config.Config{})
	m := newModel(context.Background(), service, t.TempDir())
	m.repositories = []config.Repository{{Name: "official", URL: "repo"}}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("update command = nil")
	}
	treeLoading := strings.Join(plainBlock(m.treeLinesView()), "\n")
	if m.operationKind != operationUpdate || !strings.Contains(treeLoading, "updating repo cache: official") || strings.Contains(plainText(m.statusLine()), "updating repo cache") {
		t.Fatalf("operation/tree/status = %v/%q/%q, want update progress in tree only", m.operationKind, treeLoading, plainText(m.statusLine()))
	}

	m = runOperationBatch(t, m, cmd)
	if m.operationKind != operationNone {
		t.Fatalf("operationKind = %v, want none after update result", m.operationKind)
	}
}

// TestUpdateFailureForOtherRepositoryKeepsOpenedTree 验证其他仓库更新失败不会清空当前树。
func TestUpdateFailureForOtherRepositoryKeepsOpenedTree(t *testing.T) {
	service := testService(t, createWorktree(t), config.Config{})
	m := openModelWithWorktree(t, service)
	openedRepo := m.openedRepo
	entries := append([]app.EntryResult{}, m.entries...)
	operationID := m.startOperation(operationUpdate, "updating other")

	m = updateModel(t, m, repositoryUpdatedMsg{
		operationID: operationID,
		repository:  config.Repository{Name: "other", URL: "other"},
		err:         errors.New("update failed"),
	})

	if !m.repoOpened || !sameRepository(m.openedRepo, openedRepo) || len(m.entries) != len(entries) {
		t.Fatalf("opened state = %v/%#v/%#v, want existing tree preserved", m.repoOpened, m.openedRepo, m.entries)
	}
}

func TestTreeLoadingStartsNearTopThird(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.height = 30
	m.startOperation(operationUpdate, "updating repo cache: official")

	lines := plainBlock(m.treeLinesView())
	firstContentLine := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstContentLine = i
			break
		}
	}

	if firstContentLine != 6 {
		t.Fatalf("treeLines() first content line = %d in %#v, want loading block near top third", firstContentLine, lines)
	}
}

func TestAddRepositoryShowsModalAndUsesDefaultBranch(t *testing.T) {
	store := &memoryStore{}
	service := app.Service{
		Registry:  registry.NewService(store),
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: createWorktree(t)}, branches: cache.RemoteBranches{Default: "main", Branches: []string{"dev", "main"}}},
		Installer: install.Installer{},
	}
	m := newModel(context.Background(), service, t.TempDir())

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.mode != modeAddName {
		t.Fatalf("mode = %v, want add name", m.mode)
	}
	if view := m.View(); !strings.Contains(view, "添加 Registry") {
		t.Fatalf("View() = %q, want add modal", view)
	}
	m.input.SetValue("official")
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.mode != modeAddURL {
		t.Fatalf("mode = %v, want add URL", m.mode)
	}
	m.input.SetValue("https://github.com/org/tools")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("branch command = nil")
	}
	if view := plainText(m.View()); !strings.Contains(view, "- 正在获取远端分支") {
		t.Fatalf("View() = %q, want initial branch loading spinner", view)
	}
	if status := plainText(m.statusLine()); strings.Contains(status, "正在获取远端分支") {
		t.Fatalf("statusLine() = %q, should not show branch loading text", status)
	}
	m = updateModel(t, m, selectionCursorTickMsg{})
	if view := plainText(m.View()); !strings.Contains(view, "\\ 正在获取远端分支") {
		t.Fatalf("View() = %q, want animated branch loading spinner", view)
	}
	m = runBranchCommand(t, m, cmd)
	if m.mode != modeAddBranch || len(m.pendingBranches) != 2 {
		t.Fatalf("mode/branches = %v/%#v, want branch choices", m.mode, m.pendingBranches)
	}
	if view := m.View(); !strings.Contains(view, "已获取 2 个分支") || !strings.Contains(view, "dev") || !strings.Contains(view, "main") || strings.Contains(view, "正在获取远端分支") {
		t.Fatalf("View() = %q, want branch choices immediately after load", view)
	}
	if status := plainText(m.statusLine()); strings.Contains(status, "已获取") {
		t.Fatalf("statusLine() = %q, should not show fetched branch count", status)
	}
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("add command = nil")
	}
	m = updateModel(t, m, cmd())

	if len(store.cfg.Repositories) != 1 {
		t.Fatalf("stored repositories = %#v, want one repository", store.cfg.Repositories)
	}
	got := store.cfg.Repositories[0]
	if got.Name != "official" || got.Branch != "" {
		t.Fatalf("stored repository = %#v, want official with default branch", got)
	}
}

func TestAddRepositoryArrowKeysMoveFocus(t *testing.T) {
	service := app.Service{
		Registry:  registry.NewService(&memoryStore{}),
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: createWorktree(t)}, branches: cache.RemoteBranches{Default: "main", Branches: []string{"dev", "main"}}},
		Installer: install.Installer{},
	}
	m := newModel(context.Background(), service, t.TempDir())

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m.input.SetValue("official")
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if m.mode != modeAddURL || m.pendingName != "official" {
		t.Fatalf("mode/pendingName = %v/%q, want URL focus with name saved", m.mode, m.pendingName)
	}

	m.input.SetValue("https://github.com/org/tools")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(model)
	if m.mode != modeAddName || m.input.Value() != "official" {
		t.Fatalf("mode/input = %v/%q, want name focus restored", m.mode, m.input.Value())
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if m.mode != modeAddURL || m.input.Value() != "https://github.com/org/tools" {
		t.Fatalf("mode/input = %v/%q, want URL focus restored", m.mode, m.input.Value())
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if cmd == nil || m.mode != modeAddBranch {
		t.Fatalf("mode/cmd = %v/%v, want branch focus and load command", m.mode, cmd)
	}
	m = runBranchCommand(t, m, cmd)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = next.(model)
	if m.mode != modeAddURL || m.input.Value() != "https://github.com/org/tools" {
		t.Fatalf("mode/input = %v/%q, want URL focus from branch shift-tab", m.mode, m.input.Value())
	}

	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	if m.mode != modeAddBranch || m.branchLoading {
		t.Fatalf("mode/branchLoading = %v/%v, want branch focus without reload", m.mode, m.branchLoading)
	}
}

func TestAddRepositoryCanChooseBranch(t *testing.T) {
	store := &memoryStore{}
	service := app.Service{
		Registry:  registry.NewService(store),
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: createWorktree(t)}, branches: cache.RemoteBranches{Default: "main", Branches: []string{"dev"}}},
		Installer: install.Installer{},
	}
	m := newModel(context.Background(), service, t.TempDir())

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m.input.SetValue("official")
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	m.input.SetValue("https://github.com/org/tools")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	m = runBranchCommand(t, m, cmd)

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("add command = nil")
	}
	m = updateModel(t, m, cmd())

	if got := store.cfg.Repositories[0].Branch; got != "dev" {
		t.Fatalf("Branch = %q, want dev", got)
	}
}

func TestAddRepositoryCanSearchRemoteBranch(t *testing.T) {
	store := &memoryStore{}
	service := app.Service{
		Registry: registry.NewService(store),
		Cache: &fakeCache{
			worktree: cache.Worktree{Dir: createWorktree(t)},
			branches: cache.RemoteBranches{
				Default:  "main",
				Branches: []string{"dev", "feature/login", "release"},
			},
		},
		Installer: install.Installer{},
	}
	m := newModel(context.Background(), service, t.TempDir())

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m.input.SetValue("official")
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	m.input.SetValue("https://github.com/org/tools")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	m = runBranchCommand(t, m, cmd)

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("fl")})
	if m.branchQuery != "fl" || m.selectedBranch != 1 {
		t.Fatalf("branchQuery/selectedBranch = %q/%d, want fl/1", m.branchQuery, m.selectedBranch)
	}
	lines := strings.Join(plainBlock(m.branchSelectView()), "\n")
	if !strings.Contains(lines, "feature/login") || strings.Contains(lines, "release") || strings.Contains(lines, "dev") {
		t.Fatalf("branchSelectLines() = %q, want filtered feature/login only", lines)
	}

	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("add command = nil")
	}
	m = updateModel(t, m, cmd())

	if got := store.cfg.Repositories[0].Branch; got != "feature/login" {
		t.Fatalf("Branch = %q, want feature/login", got)
	}
}

func TestBranchSelectorKeepsCursorNextToBranchLabel(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.mode = modeAddBranch
	m.pendingDefaultBranch = "main"
	m.pendingBranches = []string{"main"}
	m.selectionCursorFrame = 2

	lines := plainBlock(m.branchSelectView())
	selectedLine := lineContaining(lines, "使用远端默认分支")
	cursorCol := strings.Index(selectedLine, "»")
	branchCol := strings.Index(selectedLine, "使用远端默认分支")

	if cursorCol < 0 || branchCol < 0 {
		t.Fatalf("branchSelectView() = %#v, want selected default branch line", lines)
	}
	cursorVisibleCol := lipgloss.Width(selectedLine[:cursorCol])
	branchVisibleCol := lipgloss.Width(selectedLine[:branchCol])
	if branchVisibleCol != cursorVisibleCol+2 {
		t.Fatalf("branch column = %d, cursor column = %d in %q, want one space after cursor", branchCol, cursorCol, selectedLine)
	}
}

func TestRegistryModalBranchFocusUsesTableLikeLayout(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.mode = modeAddBranch
	m.pendingName = "aa"
	m.pendingURL = "git@github.com:anthropics/skills.git"
	m.pendingDefaultBranch = "main"
	m.pendingBranches = []string{"andibrae/create-top-level-namespace", "klazuka/add-3p-notices"}

	lines := plainBlock(m.addRepositoryModalView())
	nameLine := lineContaining(lines, "Name")
	urlLine := lineContaining(lines, "git@github.com:anthropics/skills.git")
	branchLine := lineContaining(lines, "Branch")
	statusLine := lineContaining(lines, "已获取 2 个分支")
	footerLine := lineContaining(lines, "[Tab/Shift+Tab]")

	if nameLine == "" || urlLine == "" || branchLine == "" {
		t.Fatalf("modal lines = %#v, want Name/URL/Branch form rows", lines)
	}
	nameColon := strings.Index(nameLine, ":")
	urlColon := strings.Index(urlLine, ":")
	branchColon := strings.Index(branchLine, ":")
	if nameColon < 0 || nameColon != urlColon || nameColon != branchColon {
		t.Fatalf("colon columns name/url/branch = %d/%d/%d in %#v, want aligned", nameColon, urlColon, branchColon, lines)
	}
	if statusLine == "" || !strings.Contains(statusLine, "已获取 2 个分支") || !strings.Contains(statusLine, "默认分支: main") || !strings.Contains(statusLine, "[ 1/2 ]") {
		t.Fatalf("status line = %q in %#v, want fetched branch count, default branch, and matching scroll total", statusLine, lines)
	}
	branchIndex := lineIndex(lines, branchLine)
	statusIndex := lineIndex(lines, statusLine)
	if branchIndex < 0 || statusIndex <= branchIndex+1 || !strings.Contains(lines[statusIndex-1], "─") {
		t.Fatalf("modal lines = %#v, want horizontal divider between form and branch list", lines)
	}
	if footerLine == "" || !strings.Contains(footerLine, " | ") || !strings.Contains(footerLine, "[Enter] 确认") {
		t.Fatalf("footer line = %q, want bracketed key groups separated by pipes", footerLine)
	}
}

func TestBranchSelectorHighlightsSelectedRowWithBackground(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.mode = modeAddBranch
	m.pendingDefaultBranch = "main"
	m.pendingBranches = []string{"dev", "main"}
	m.selectedBranch = 1

	lines := splitRenderedLines(m.branchSelectView())
	selectedLine := lineContaining(lines, "dev")
	if selectedLine == "" {
		t.Fatalf("branchSelectView() = %#v, want selected dev row", plainLines(lines))
	}
	if width := lipgloss.Width(plainText(selectedLine)); width < 70 {
		t.Fatalf("selected line width = %d in %q, want row highlight to span list width", width, plainText(selectedLine))
	}
	if got := modalBranchSelectedLineStyle(70).GetBackground(); got != lipgloss.Color("24") {
		t.Fatalf("selected background = %#v, want color 24", got)
	}
}

func TestEditRepositoryShowsPrefilledModalAndPersists(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{Name: "official", URL: "https://github.com/org/tools", Branch: "main"}},
	}}
	cacheSvc := &fakeCache{
		worktree: cache.Worktree{Dir: createWorktree(t)},
		branches: cache.RemoteBranches{
			Default:  "main",
			Branches: []string{"dev", "main"},
		},
	}
	service := app.Service{
		Registry:  registry.NewService(store),
		Cache:     cacheSvc,
		Installer: install.Installer{},
	}
	m := newModel(context.Background(), service, t.TempDir())
	m.repositories = store.cfg.Repositories

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if m.mode != modeAddName || !m.editingRepositoryActive || m.input.Value() != "official" {
		t.Fatalf("mode/editing/input = %v/%v/%q, want edit name for official", m.mode, m.editingRepositoryActive, m.input.Value())
	}
	if view := m.View(); !strings.Contains(view, "编辑 Registry") || !strings.Contains(view, "main") {
		t.Fatalf("View() = %q, want edit modal with current branch", view)
	}

	m.input.SetValue("tools")
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	m.input.SetValue("https://github.com/org/tools")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("branch command = nil")
	}
	m = runBranchCommand(t, m, cmd)
	if m.mode != modeAddBranch || m.selectedBranch != 2 {
		t.Fatalf("mode/selectedBranch = %v/%d, want branch selection on main", m.mode, m.selectedBranch)
	}

	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("edit command = nil")
	}
	m = updateModel(t, m, cmd())

	want := []config.Repository{{Name: "tools", URL: "https://github.com/org/tools", Branch: "main"}}
	if !reflect.DeepEqual(store.cfg.Repositories, want) {
		t.Fatalf("Repositories = %#v, want %#v", store.cfg.Repositories, want)
	}
	if m.editingRepositoryActive || m.mode != modeNormal {
		t.Fatalf("editing/mode = %v/%v, want cleared normal mode", m.editingRepositoryActive, m.mode)
	}
	if len(cacheSvc.deleted) != 0 {
		t.Fatalf("deleted caches = %#v, want none for rename", cacheSvc.deleted)
	}
}

func TestRegistryModalInactiveURLAlignsAfterNameCursor(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.mode = modeAddName
	m.pendingURL = "git@github.com:anthropics/skills.git"

	lines := plainBlock(m.addRepositoryModalView())
	nameLine := lineContaining(lines, "Name")
	urlLine := lineContaining(lines, m.pendingURL)
	cursorCol := strings.Index(nameLine, ">")
	urlCol := strings.Index(urlLine, m.pendingURL)

	if cursorCol < 0 || urlCol < 0 {
		t.Fatalf("modal lines = %#v, want name cursor and url", lines)
	}
	if urlCol != cursorCol+2 {
		t.Fatalf("url column = %d, cursor column = %d in %#v, want url aligned with input text after prompt", urlCol, cursorCol, lines)
	}
}

func TestRegistryModalColumnsStayStableWhenURLInputChanges(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.pendingName = "aa"
	m, _ = m.focusAddURL("")

	placeholderLines := plainBlock(m.addRepositoryModalView())
	placeholderNameLine := lineContaining(placeholderLines, "aa")
	placeholderURLLine := lineContaining(placeholderLines, "repo url")
	placeholderNameCol := strings.Index(placeholderNameLine, "aa")
	placeholderURLCol := strings.Index(placeholderURLLine, "repo url")

	m.input.SetValue("i")
	inputLines := plainBlock(m.addRepositoryModalView())
	inputNameLine := lineContaining(inputLines, "aa")
	inputURLLine := lineContaining(inputLines, "> i")
	inputNameCol := strings.Index(inputNameLine, "aa")
	inputURLCol := strings.Index(inputURLLine, "i")

	if placeholderNameCol < 0 || placeholderURLCol < 0 || inputNameCol < 0 || inputURLCol < 0 {
		t.Fatalf("modal lines before = %#v, after = %#v, want name and URL input columns", placeholderLines, inputLines)
	}
	if placeholderNameCol != inputNameCol || placeholderURLCol != inputURLCol {
		t.Fatalf("columns before name/url = %d/%d, after = %d/%d; lines before = %#v, after = %#v", placeholderNameCol, placeholderURLCol, inputNameCol, inputURLCol, placeholderLines, inputLines)
	}
	if inputNameCol != inputURLCol {
		t.Fatalf("name column = %d, URL input column = %d in %#v, want aligned field values", inputNameCol, inputURLCol, inputLines)
	}
}

func TestViewUsesRepositoryTreeInsteadOfSkillPreview(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())

	view := m.View()

	if !strings.Contains(view, "Repository Tree") {
		t.Fatalf("View() = %q, want Repository Tree", view)
	}
	if strings.Contains(view, "Skills") || strings.Contains(view, "Preview") {
		t.Fatalf("View() = %q, should not contain old skill panes", view)
	}
}

var ansiEscapeRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plainText(value string) string {
	return ansiEscapeRegexp.ReplaceAllString(value, "")
}

func plainLines(lines []string) []string {
	plain := make([]string, 0, len(lines))
	for _, line := range lines {
		plain = append(plain, plainText(line))
	}
	return plain
}

func plainBlock(value string) []string {
	return plainLines(splitRenderedLines(value))
}

func lineContaining(lines []string, value string) string {
	for _, line := range lines {
		if strings.Contains(line, value) {
			return line
		}
	}
	return ""
}

func lineIndex(lines []string, value string) int {
	for i, line := range lines {
		if line == value {
			return i
		}
	}
	return -1
}

func runBranchCommand(t *testing.T, m model, cmd tea.Cmd) model {
	t.Helper()

	if cmd == nil {
		t.Fatalf("branch command = nil")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return updateModel(t, m, msg)
	}
	for _, child := range batch {
		if child == nil {
			continue
		}
		childMsg := child()
		if childMsg == nil {
			continue
		}
		if _, ok := childMsg.(branchesLoadedMsg); ok {
			m = updateModel(t, m, childMsg)
			return m
		}
	}
	return m
}

func runOperationBatch(t *testing.T, m model, cmd tea.Cmd) model {
	t.Helper()

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return updateModel(t, m, msg)
	}
	if len(batch) < 2 {
		t.Fatalf("operation batch length = %d, want at least 2", len(batch))
	}

	worker := batch[len(batch)-1]
	if msg := worker(); msg != nil {
		m = updateModel(t, m, msg)
	}

	listen := batch[1]
	for i := 0; i < 20 && listen != nil; i++ {
		msg := listen()
		if msg == nil {
			break
		}
		next, nextCmd := m.Update(msg)
		m = next.(model)
		if m.operationKind == operationNone {
			break
		}
		listen = nextCmd
	}
	return m
}

func openModelWithWorktree(t *testing.T, service app.Service) model {
	t.Helper()

	m := newModel(context.Background(), service, t.TempDir())
	m.repositories = []config.Repository{{Name: "official", URL: "repo"}}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	m = next.(model)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("open command = nil")
	}
	return runOperationBatch(t, m, cmd)
}

func updateModel(t *testing.T, m model, msg tea.Msg) model {
	t.Helper()

	next, _ := m.Update(msg)
	return next.(model)
}

func testService(t *testing.T, worktree string, cfg config.Config) app.Service {
	t.Helper()

	return app.Service{
		Registry:  registry.NewService(&memoryStore{cfg: cfg}),
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: worktree}},
		Installer: install.Installer{},
	}
}

type memoryStore struct {
	// cfg 是测试内存配置。
	cfg config.Config
}

// Load 返回测试内存配置。
func (s *memoryStore) Load() (config.Config, error) {
	return s.cfg, nil
}

// Save 覆盖测试内存配置。
func (s *memoryStore) Save(cfg config.Config) error {
	s.cfg = cfg
	return nil
}

type fakeCache struct {
	// worktree 是 Ensure 和 Update 返回的工作区。
	worktree cache.Worktree
	// deleted 记录被删除的仓库。
	deleted []config.Repository
	// branches 是 ListRemoteBranches 返回的分支信息。
	branches cache.RemoteBranches
}

// Ensure 返回测试工作区。
func (c *fakeCache) Ensure(ctx context.Context, repo config.Repository) (cache.Worktree, error) {
	return c.worktree, nil
}

// EnsureWithProgress 返回测试工作区并发送测试进度。
func (c *fakeCache) EnsureWithProgress(ctx context.Context, repo config.Repository, progress cache.ProgressFunc) (cache.Worktree, error) {
	if progress != nil {
		progress(cache.Progress{Text: "Receiving objects: 50%", Percent: 50})
	}
	return c.Ensure(ctx, repo)
}

// Update 返回测试工作区。
func (c *fakeCache) Update(ctx context.Context, repo config.Repository) (cache.Worktree, error) {
	return c.worktree, nil
}

// UpdateWithProgress 返回测试工作区并发送测试进度。
func (c *fakeCache) UpdateWithProgress(ctx context.Context, repo config.Repository, progress cache.ProgressFunc) (cache.Worktree, error) {
	if progress != nil {
		progress(cache.Progress{Text: "Receiving objects: 50%", Percent: 50})
	}
	return c.Update(ctx, repo)
}

// Delete 记录 cache 删除请求。
func (c *fakeCache) Delete(repo config.Repository) error {
	c.deleted = append(c.deleted, repo)
	return nil
}

// ListRemoteBranches 返回测试远端分支。
func (c *fakeCache) ListRemoteBranches(ctx context.Context, repoURL string) (cache.RemoteBranches, error) {
	return c.branches, nil
}

func createWorktree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("readme\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "guide.md"), []byte("guide\n"), 0o644); err != nil {
		t.Fatalf("write guide: %v", err)
	}
	return root
}

// assertFileContent 校验指定文件内容。
func assertFileContent(t *testing.T, filePath string, want string) {
	t.Helper()

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read %s: %v", filePath, err)
	}
	if string(content) != want {
		t.Fatalf("%s = %q, want %q", filePath, string(content), want)
	}
}
