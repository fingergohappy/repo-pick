// Package skill 负责从本地 worktree 中发现可安装的 skill。
package skill

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill 表示 worktree 中一个可安装的 skill。
type Skill struct {
	// Name 是用于展示、匹配和安装目录拼接的安全名称。
	Name string
	// Path 是相对 worktree 根目录的 slash 风格 skill 目录路径。
	Path string
	// Description 是 SKILL.md front matter 中可选的展示描述。
	Description string
	// Content 是完整 SKILL.md 文件内容，供 preview 或详情展示使用。
	Content string
}

type frontMatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Discover 从 repo worktree 中指定目录路径下发现 skill，dirPath 为空时默认使用 skills。
func Discover(root string, dirPath string) ([]Skill, error) {
	dirPath, err := normalizeDirPath(dirPath)
	if err != nil {
		return nil, err
	}

	skills, err := discoverInDir(root, dirPath)
	if err != nil || len(skills) > 0 || dirPath == "skills" {
		return skills, err
	}
	return discoverInDir(root, path.Join(dirPath, "skills"))
}

// discoverInDir 从指定相对目录路径下发现直接子级 skill。
func discoverInDir(root string, dirPath string) ([]Skill, error) {
	skillsDir := filepath.Join(root, filepath.FromSlash(dirPath))
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skills dir %q: %w", skillsDir, err)
	}

	skills := make([]Skill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := path.Join(dirPath, entry.Name())
		contentPath := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		content, err := os.ReadFile(contentPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", skillPathForError(skillPath), err)
		}

		metadata, err := parseFrontMatter(string(content))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", skillPathForError(skillPath), err)
		}

		name := entry.Name()
		if metadata.Name != "" {
			name = metadata.Name
		}
		if err := validateName(name); err != nil {
			return nil, fmt.Errorf("validate %s name %q: %w", skillPathForError(skillPath), name, err)
		}

		skills = append(skills, Skill{
			Name:        name,
			Path:        skillPath,
			Description: metadata.Description,
			Content:     string(content),
		})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills, nil
}

// normalizeDirPath 校验搜索目录路径，并在空值时应用默认 skills 目录。
func normalizeDirPath(dirPath string) (string, error) {
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		return "skills", nil
	}
	dirPath = filepath.ToSlash(dirPath)
	dirPath = strings.TrimLeft(dirPath, "/")
	if dirPath == "" || dirPath == "." || dirPath == ".." {
		return "", errors.New("dirPath cannot be empty, . or ..")
	}
	for _, part := range strings.Split(dirPath, "/") {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("dirPath cannot contain empty, . or .. segments")
		}
	}
	return path.Clean(dirPath), nil
}

// parseFrontMatter 解析 SKILL.md 顶部可选 YAML front matter。
func parseFrontMatter(content string) (frontMatter, error) {
	if !strings.HasPrefix(content, "---\n") {
		return frontMatter{}, nil
	}

	rest := strings.TrimPrefix(content, "---\n")
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return frontMatter{}, nil
	}

	var metadata frontMatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &metadata); err != nil {
		return frontMatter{}, err
	}
	return metadata, nil
}

// validateName 校验 skill 名称可安全用于展示、匹配和目录拼接。
func validateName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	if name == "." || name == ".." {
		return errors.New("name cannot be . or ..")
	}
	if strings.ContainsAny(name, `/\`) {
		return errors.New("name cannot contain path separators")
	}
	return nil
}

// skillPathForError 为错误消息拼接 SKILL.md 相对路径。
func skillPathForError(skillPath string) string {
	return path.Join(skillPath, "SKILL.md")
}
