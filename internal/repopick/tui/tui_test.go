package tui

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

	lines := strings.Join(plainLines(m.registryLines()), "\n")

	if !strings.Contains(lines, "暂无 registry") || !strings.Contains(lines, "添加 registry") {
		t.Fatalf("registryLines() = %q, want designed empty placeholder", lines)
	}
	if strings.Contains(lines, "a add") {
		t.Fatalf("registryLines() = %q, should not use old empty placeholder", lines)
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
	treeLoading := strings.Join(plainLines(m.treeLines()), "\n")
	if m.operationKind != operationOpen || !strings.Contains(treeLoading, "loading repo cache: official") || strings.Contains(plainText(m.statusLine()), "loading repo cache") {
		t.Fatalf("operation/tree/status = %v/%q/%q, want open progress in tree only", m.operationKind, treeLoading, plainText(m.statusLine()))
	}
	m = updateModel(t, m, operationProgressMsg{kind: operationOpen, baseLabel: "loading repo cache: official", event: app.ProgressEvent{Text: "Receiving objects: 50%", Percent: 50}})
	if treeLoading = strings.Join(plainLines(m.treeLines()), "\n"); !strings.Contains(treeLoading, "50%") {
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

	treeStatus := m.statusLine()
	if !strings.Contains(treeStatus, "l expand/collapse") || !strings.Contains(treeStatus, "o enter root") || strings.Contains(treeStatus, "r reload list") {
		t.Fatalf("tree statusLine() = %q, want tree help", treeStatus)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlW})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	registryStatus := m.statusLine()
	if !strings.Contains(registryStatus, "r reload list") || !strings.Contains(registryStatus, "u update repo cache") || strings.Contains(registryStatus, "l expand") {
		t.Fatalf("registry statusLine() = %q, want registry help", registryStatus)
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
	lines := plainLines(m.treeLines())
	if len(lines) < 6 || !strings.Contains(lines[2], "search") || !strings.Contains(lines[2], "guide") {
		t.Fatalf("treeLines() = %#v, want search context", lines)
	}
	if !strings.Contains(lines[3], "---") || !strings.Contains(lines[4], "type") {
		t.Fatalf("treeLines() = %#v, want separator and content header", lines)
	}
	if !strings.Contains(lines[5], "docs/guide.md") {
		t.Fatalf("treeLines() = %#v, want search result below content header", lines)
	}
}

func TestSearchInputRendersAboveFileList(t *testing.T) {
	worktree := createWorktree(t)
	service := testService(t, worktree, config.Config{})
	m := openModelWithWorktree(t, service)

	m.mode = modeSearch
	m.input.SetValue("guide")
	lines := plainLines(m.treeLines())

	if len(lines) < 6 || !strings.Contains(lines[2], "search") || !strings.Contains(lines[2], "guide") {
		t.Fatalf("treeLines() = %#v, want search input in context area", lines)
	}
	if !strings.Contains(lines[3], "---") || !strings.Contains(lines[4], "tree") {
		t.Fatalf("treeLines() = %#v, want separator and header before file list", lines)
	}
}

func TestTreeToggleOpensDirectoryWithL(t *testing.T) {
	worktree := createWorktree(t)
	service := testService(t, worktree, config.Config{})
	m := openModelWithWorktree(t, service)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("l command = nil, want load children command")
	}
	m = updateModel(t, m, cmd())

	entries := m.visibleEntries()
	if len(entries) < 2 || entries[0].Path != "docs" || entries[1].Path != "docs/guide.md" {
		t.Fatalf("visibleEntries() = %#v, want expanded docs tree", entries)
	}
	lines := plainLines(m.treeLines())
	if !strings.Contains(lines[4], "├── ▾ docs/") || !strings.Contains(lines[5], "│   └── • guide.md") {
		t.Fatalf("treeLines() = %#v, want expanded directory and child", lines)
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
	m.selectedEntry = 1

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
	treeLoading := strings.Join(plainLines(m.treeLines()), "\n")
	if m.operationKind != operationUpdate || !strings.Contains(treeLoading, "updating repo cache: official") || strings.Contains(plainText(m.statusLine()), "updating repo cache") {
		t.Fatalf("operation/tree/status = %v/%q/%q, want update progress in tree only", m.operationKind, treeLoading, plainText(m.statusLine()))
	}

	m = runOperationBatch(t, m, cmd)
	if m.operationKind != operationNone {
		t.Fatalf("operationKind = %v, want none after update result", m.operationKind)
	}
}

func TestTreeLoadingStartsNearTopThird(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, t.TempDir())
	m.height = 30
	m.startOperation(operationUpdate, "updating repo cache: official")

	lines := plainLines(m.treeLines())
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
	m = updateModel(t, m, cmd())
	if m.mode != modeAddBranch || len(m.pendingBranches) != 2 {
		t.Fatalf("mode/branches = %v/%#v, want branch choices", m.mode, m.pendingBranches)
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
	m = updateModel(t, m, cmd())

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = next.(model)
	if m.mode != modeAddURL || m.input.Value() != "https://github.com/org/tools" {
		t.Fatalf("mode/input = %v/%q, want URL focus from branch shift-tab", m.mode, m.input.Value())
	}

	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	if cmd != nil || m.mode != modeAddBranch {
		t.Fatalf("mode/cmd = %v/%v, want branch focus without reload", m.mode, cmd)
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
	m = updateModel(t, m, cmd())

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
	m = updateModel(t, m, cmd())

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("fl")})
	if m.branchQuery != "fl" || m.selectedBranch != 1 {
		t.Fatalf("branchQuery/selectedBranch = %q/%d, want fl/1", m.branchQuery, m.selectedBranch)
	}
	lines := strings.Join(plainLines(m.branchSelectLines()), "\n")
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
