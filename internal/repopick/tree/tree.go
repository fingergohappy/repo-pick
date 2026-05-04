// Package tree 负责读取 repo cache 工作区中的文件和目录条目。
package tree

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// EntryType 表示仓库条目的类型。
type EntryType string

const (
	// EntryFile 表示普通文件。
	EntryFile EntryType = "file"
	// EntryDir 表示目录。
	EntryDir EntryType = "dir"
)

// Entry 表示仓库中的一个文件或目录。
type Entry struct {
	// Name 是文件或目录名。
	Name string
	// Path 是相对仓库根目录的 slash 风格路径。
	Path string
	// Type 是条目类型。
	Type EntryType
	// Size 是普通文件大小；目录固定为 0。
	Size int64
}

// List 返回指定目录的直接子级条目。
func List(root string, dirPath string) ([]Entry, error) {
	dir, cleanPath, err := resolvePath(root, dirPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("stat repo dir %q: %w", cleanPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("repo path %q is not a directory", cleanPath)
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read repo dir %q: %w", cleanPath, err)
	}

	entries := make([]Entry, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		if dirEntry.Name() == ".git" {
			continue
		}
		entry, ok, err := entryFromDirEntry(cleanPath, dirEntry)
		if err != nil {
			return nil, err
		}
		if ok {
			entries = append(entries, entry)
		}
	}
	sortEntries(entries)
	return entries, nil
}

// Search 遍历当前仓库全部路径，并按相对路径模糊匹配 query。
func Search(root string, query string) ([]Entry, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))

	var entries []Entry
	err = filepath.WalkDir(rootAbs, func(current string, dirEntry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == rootAbs {
			return nil
		}

		rel, err := filepath.Rel(rootAbs, current)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ".git" || strings.HasPrefix(rel, ".git/") {
			if dirEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !fuzzyMatch(strings.ToLower(rel), query) {
			return nil
		}

		entry, ok, err := entryFromPath(rel, dirEntry)
		if err != nil {
			return err
		}
		if ok {
			entries = append(entries, entry)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sortEntries(entries)
	return entries, nil
}

// fuzzyMatch 判断 query 中的字符是否按顺序出现在 target 中。
func fuzzyMatch(target string, query string) bool {
	if query == "" {
		return true
	}

	targetRunes := []rune(target)
	targetIndex := 0
	for _, queryRune := range []rune(query) {
		found := false
		for targetIndex < len(targetRunes) {
			if targetRunes[targetIndex] == queryRune {
				targetIndex++
				found = true
				break
			}
			targetIndex++
		}
		if !found {
			return false
		}
	}
	return true
}

// entryFromDirEntry 将 os.DirEntry 转换为仓库 Entry。
func entryFromDirEntry(parentPath string, dirEntry os.DirEntry) (Entry, bool, error) {
	entryPath := path.Join(parentPath, dirEntry.Name())
	if parentPath == "" {
		entryPath = dirEntry.Name()
	}
	return entryFromPath(entryPath, dirEntry)
}

// entryFromPath 根据相对路径和文件信息构造 Entry。
func entryFromPath(relPath string, dirEntry os.DirEntry) (Entry, bool, error) {
	info, err := dirEntry.Info()
	if err != nil {
		return Entry{}, false, err
	}
	if dirEntry.IsDir() {
		return Entry{Name: path.Base(relPath), Path: relPath, Type: EntryDir}, true, nil
	}
	if info.Mode().IsRegular() {
		return Entry{Name: path.Base(relPath), Path: relPath, Type: EntryFile, Size: info.Size()}, true, nil
	}
	return Entry{}, false, nil
}

// resolvePath 将仓库内目录路径转换为安全的本地目录路径。
func resolvePath(root string, dirPath string) (string, string, error) {
	if strings.TrimSpace(root) == "" {
		return "", "", errors.New("repo root is required")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}

	cleanPath, err := normalizeDirPath(dirPath)
	if err != nil {
		return "", "", err
	}
	target := filepath.Join(rootAbs, filepath.FromSlash(cleanPath))
	rel, err := filepath.Rel(rootAbs, target)
	if err != nil {
		return "", "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("repo path %q escapes root", dirPath)
	}
	return target, cleanPath, nil
}

// normalizeDirPath 规范化仓库内目录路径并拒绝逃逸片段。
func normalizeDirPath(dirPath string) (string, error) {
	dirPath = strings.TrimSpace(filepath.ToSlash(dirPath))
	if dirPath == "" || dirPath == "/" || dirPath == "." {
		return "", nil
	}
	dirPath = strings.TrimLeft(dirPath, "/")
	for _, part := range strings.Split(dirPath, "/") {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("repo path cannot contain empty, . or .. segments")
		}
	}
	return path.Clean(dirPath), nil
}

// sortEntries 按目录优先和路径升序稳定排序。
func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type == EntryDir
		}
		return entries[i].Path < entries[j].Path
	})
}
