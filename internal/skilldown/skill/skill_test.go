package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverFindsSkillsWithMetadataAndContent(t *testing.T) {
	root := filepath.Join("..", "..", "..", "test", "testdata", "skill_repo")

	skills, err := Discover(root, "skills")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 2 {
		t.Fatalf("Discover() returned %d skills, want 2: %#v", len(skills), skills)
	}

	assertSkill(t, skills[0], Skill{
		Name:        "alpha",
		Path:        "skills/alpha",
		Description: "Alpha skill",
	}, "Alpha body")
	assertSkill(t, skills[1], Skill{
		Name: "beta-dir",
		Path: "skills/beta-dir",
	}, "Beta body")
}

func TestDiscoverUsesDefaultSkillsDirWhenDirPathIsEmpty(t *testing.T) {
	root := filepath.Join("..", "..", "..", "test", "testdata", "skill_repo")

	skills, err := Discover(root, "")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 2 {
		t.Fatalf("Discover() returned %d skills, want 2: %#v", len(skills), skills)
	}
}

func TestDiscoverFindsSkillsInCustomDirPathWithoutFallback(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "skills", "ignored", "---\nname: ignored\n---\nIgnored body")
	writeSkill(t, root, "agents", "alpha", "---\nname: agent-alpha\ndescription: Agent alpha\n---\nAgent body")

	skills, err := Discover(root, "agents")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("Discover() returned %d skills, want 1: %#v", len(skills), skills)
	}
	assertSkill(t, skills[0], Skill{
		Name:        "agent-alpha",
		Path:        "agents/alpha",
		Description: "Agent alpha",
	}, "Agent body")

	skills, err = Discover(root, "missing")
	if err != nil {
		t.Fatalf("Discover() missing dir error = %v", err)
	}
	if len(skills) != 0 {
		t.Fatalf("Discover() fell back to another dir, returned %d skills: %#v", len(skills), skills)
	}
}

func TestDiscoverFindsSkillsUnderCustomDirPath(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, filepath.Join("a", "b"), "alpha", "---\nname: alpha\n---\nAlpha body")

	skills, err := Discover(root, "/a/b")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("Discover() returned %d skills, want 1: %#v", len(skills), skills)
	}
	assertSkill(t, skills[0], Skill{
		Name: "alpha",
		Path: "a/b/alpha",
	}, "Alpha body")
}

func TestDiscoverFallsBackToSkillsUnderCustomDirPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b", "not-a-skill"), 0o755); err != nil {
		t.Fatalf("mkdir non-skill dir: %v", err)
	}
	writeSkill(t, root, filepath.Join("a", "b", "skills"), "alpha", "---\nname: alpha\n---\nAlpha body")

	skills, err := Discover(root, "a/b")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("Discover() returned %d skills, want 1: %#v", len(skills), skills)
	}
	assertSkill(t, skills[0], Skill{
		Name: "alpha",
		Path: "a/b/skills/alpha",
	}, "Alpha body")
}

func TestDiscoverDoesNotFallBackWhenCustomDirPathContainsSkills(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, filepath.Join("a", "b"), "alpha", "---\nname: alpha\n---\nAlpha body")
	writeSkill(t, root, filepath.Join("a", "b", "skills"), "ignored", "---\nname: ignored\n---\nIgnored body")

	skills, err := Discover(root, "a/b")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("Discover() returned %d skills, want 1: %#v", len(skills), skills)
	}
	assertSkill(t, skills[0], Skill{
		Name: "alpha",
		Path: "a/b/alpha",
	}, "Alpha body")
}

func TestDiscoverRejectsEscapingDirPath(t *testing.T) {
	root := t.TempDir()

	_, err := Discover(root, "a/../b")
	if err == nil {
		t.Fatal("Discover() error = nil, want invalid dir path error")
	}
}

func TestDiscoverReturnsEmptyListWhenSkillsDirIsMissing(t *testing.T) {
	root := t.TempDir()

	skills, err := Discover(root, "skills")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(skills) != 0 {
		t.Fatalf("Discover() returned %d skills, want 0", len(skills))
	}
}

func TestDiscoverRejectsInvalidFrontMatterName(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "skills", "bad", "---\nname: ../bad\n---\n")

	_, err := Discover(root, "skills")
	if err == nil {
		t.Fatal("Discover() error = nil, want invalid name error")
	}
	if !strings.Contains(err.Error(), "skills/bad/SKILL.md") {
		t.Fatalf("Discover() error = %q, want path context", err)
	}
}

func TestDiscoverRejectsBrokenFrontMatter(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "skills", "bad", "---\nname: [broken\n---\n")

	_, err := Discover(root, "skills")
	if err == nil {
		t.Fatal("Discover() error = nil, want YAML error")
	}
	if !strings.Contains(err.Error(), "skills/bad/SKILL.md") {
		t.Fatalf("Discover() error = %q, want path context", err)
	}
}

func assertSkill(t *testing.T, got Skill, want Skill, contentPart string) {
	t.Helper()

	if got.Name != want.Name || got.Path != want.Path || got.Description != want.Description {
		t.Fatalf("skill = %#v, want name/path/description %#v", got, want)
	}
	if !strings.Contains(got.Content, contentPart) {
		t.Fatalf("Skill.Content = %q, want it to contain %q", got.Content, contentPart)
	}
}

func writeSkill(t *testing.T, root, dirPath, name, content string) {
	t.Helper()

	dir := filepath.Join(root, dirPath, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}
