package tui

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/finger/skill-down/internal/skilldown/app"
	"github.com/finger/skill-down/internal/skilldown/config"
	"github.com/finger/skill-down/internal/skilldown/install"
	"github.com/finger/skill-down/internal/skilldown/registry"
	"github.com/finger/skill-down/internal/skilldown/repo"
)

func TestModelMovesFocusWithVimKeys(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, "")

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l"), Alt: false})
	if m.focus != paneSkills {
		t.Fatalf("focus = %v, want skills pane", m.focus)
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l"), Alt: false})
	if m.focus != panePreview {
		t.Fatalf("focus = %v, want preview pane", m.focus)
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H"), Alt: false})
	if m.focus != paneRegistry {
		t.Fatalf("focus = %v, want registry pane", m.focus)
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L"), Alt: false})
	if m.focus != panePreview {
		t.Fatalf("focus = %v, want preview pane", m.focus)
	}
}

func TestModelSelectsSkillsWithSpace(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, "")
	m.focus = paneSkills
	m.skills = []app.SkillResult{{Name: "alpha"}, {Name: "beta"}}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if !m.selected["alpha"] {
		t.Fatalf("selected = %#v, want alpha selected", m.selected)
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if m.selected["alpha"] {
		t.Fatalf("selected = %#v, want alpha unselected", m.selected)
	}
}

func TestModelDebouncesRegistryLoad(t *testing.T) {
	service := app.Service{
		Registry:  registry.NewService(&memoryStore{cfg: config.Config{Repositories: []config.Repository{{Name: "one", URL: "one"}, {Name: "two", URL: "two", SkillDir: "custom"}}}}),
		Cloner:    &countingCloner{},
		Installer: install.Installer{},
	}
	m := newModel(context.Background(), service, "")
	m.repositories = []config.Repository{{Name: "one", URL: "one"}, {Name: "two", URL: "two", SkillDir: "custom"}}
	cloner := service.Cloner.(*countingCloner)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j"), Alt: false})
	m = next.(model)
	if m.registryCursor != 1 {
		t.Fatalf("registryCursor = %d, want 1", m.registryCursor)
	}
	if cmd == nil {
		t.Fatalf("Update() command is nil, want debounce command")
	}
	if cloner.cloneCount != 0 {
		t.Fatalf("cloneCount = %d, want no immediate search", cloner.cloneCount)
	}

	next, cmd = m.Update(registryLoadDueMsg{token: m.registryLoadToken, repo: m.repositories[m.registryCursor]})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("registryLoadDueMsg command is nil, want search command")
	}
	_ = cmd()
	if cloner.cloneCount != 1 {
		t.Fatalf("cloneCount = %d, want one debounced search", cloner.cloneCount)
	}
	if got := cloner.sparsePaths[0]; got != "custom" {
		t.Fatalf("sparse path = %q, want registry skill dir", got)
	}
}

func TestModelOpensInstallConfirm(t *testing.T) {
	m := newModel(context.Background(), app.Service{}, "")
	m.focus = paneSkills
	m.skills = []app.SkillResult{{Name: "alpha"}}
	m.selected["alpha"] = true

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i"), Alt: false})

	if m.mode != modeInstallConfirm {
		t.Fatalf("mode = %v, want install confirm", m.mode)
	}
	if !m.input.Focused() {
		t.Fatalf("install target input is not focused")
	}
}

func updateModel(t *testing.T, m model, msg tea.Msg) model {
	t.Helper()

	next, _ := m.Update(msg)
	return next.(model)
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

type countingCloner struct {
	// cloneCount 记录 Clone 被调用的次数。
	cloneCount int
	// sparsePaths 记录最近一次 Clone 请求的稀疏路径。
	sparsePaths []string
}

// Clone 记录 clone 请求并返回空 worktree。
func (c *countingCloner) Clone(ctx context.Context, repoURL string, options repo.CloneOptions) (repo.Worktree, error) {
	c.cloneCount++
	c.sparsePaths = options.SparsePaths
	return repo.Worktree{}, nil
}

// Cleanup 在测试中不执行实际清理。
func (c *countingCloner) Cleanup(worktree repo.Worktree) error {
	return nil
}

var errUnexpectedCopy = errors.New("unexpected copy")

// CopyDir 防止测试意外执行安装复制。
func (c *countingCloner) CopyDir(ctx context.Context, sourceDir string, targetDir string, force bool) install.Result {
	return install.Result{Err: errUnexpectedCopy}
}
