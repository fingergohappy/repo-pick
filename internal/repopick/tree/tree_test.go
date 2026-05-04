package tree

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListReturnsDirectChildren(t *testing.T) {
	root := createRepoTree(t)

	entries, err := List(root, "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := []Entry{
		{Name: "docs", Path: "docs", Type: EntryDir},
		{Name: "README.md", Path: "README.md", Type: EntryFile, Size: int64(len("readme\n"))},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("List() = %#v, want %#v", entries, want)
	}
}

func TestListReturnsNestedChildren(t *testing.T) {
	root := createRepoTree(t)

	entries, err := List(root, "docs")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := []Entry{{Name: "guide.md", Path: "docs/guide.md", Type: EntryFile, Size: int64(len("guide\n"))}}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("List() = %#v, want %#v", entries, want)
	}
}

func TestListRejectsEscapingPath(t *testing.T) {
	_, err := List(t.TempDir(), "../outside")
	if err == nil {
		t.Fatal("List() error = nil, want escaping path error")
	}
}

func TestSearchMatchesPathsAndSkipsGit(t *testing.T) {
	root := createRepoTree(t)

	entries, err := Search(root, "guide")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	want := []Entry{{Name: "guide.md", Path: "docs/guide.md", Type: EntryFile, Size: int64(len("guide\n"))}}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("Search() = %#v, want %#v", entries, want)
	}
}

func TestSearchFuzzyMatchesPath(t *testing.T) {
	root := createRepoTree(t)

	entries, err := Search(root, "dg")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	want := []Entry{{Name: "guide.md", Path: "docs/guide.md", Type: EntryFile, Size: int64(len("guide\n"))}}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("Search() = %#v, want %#v", entries, want)
	}
}

func createRepoTree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "ignored"), []byte("ignored\n"), 0o644); err != nil {
		t.Fatalf("write .git file: %v", err)
	}
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
