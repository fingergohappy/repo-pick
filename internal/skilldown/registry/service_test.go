package registry

import (
	"errors"
	"reflect"
	"testing"

	"github.com/finger/skill-down/internal/skilldown/config"
)

func TestServiceAddDefaultsSkillDirAndPersists(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)

	err := service.Add(config.Repository{
		Name: "official",
		URL:  "https://github.com/org/skills",
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	want := []config.Repository{{
		Name:     "official",
		URL:      "https://github.com/org/skills",
		SkillDir: "skills",
	}}
	if !reflect.DeepEqual(store.cfg.Repositories, want) {
		t.Fatalf("Repositories = %#v, want %#v", store.cfg.Repositories, want)
	}
}

func TestServiceAddRejectsDuplicateName(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{
			Name:     "official",
			URL:      "https://github.com/org/skills",
			SkillDir: "skills",
		}},
	}}
	service := NewService(store)

	err := service.Add(config.Repository{
		Name: "official",
		URL:  "https://github.com/other/skills",
	})
	if !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("Add() error = %v, want ErrDuplicateName", err)
	}
}

func TestServiceRemoveDeletesByNameAndPersists(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{
			{Name: "official", URL: "https://github.com/org/skills", SkillDir: "skills"},
			{Name: "personal", URL: "git@github.com:finger/my-skills.git", SkillDir: "skills"},
		},
	}}
	service := NewService(store)

	if err := service.Remove("official"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	want := []config.Repository{{
		Name:     "personal",
		URL:      "git@github.com:finger/my-skills.git",
		SkillDir: "skills",
	}}
	if !reflect.DeepEqual(store.cfg.Repositories, want) {
		t.Fatalf("Repositories = %#v, want %#v", store.cfg.Repositories, want)
	}
}

func TestServiceResolveUsesExplicitRepoWithoutStore(t *testing.T) {
	store := &memoryStore{loadErr: errors.New("load should not be called")}
	service := NewService(store)

	repos, err := service.Resolve("https://github.com/org/skills")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	want := []config.Repository{{
		Name:     "https://github.com/org/skills",
		URL:      "https://github.com/org/skills",
		SkillDir: "skills",
	}}
	if !reflect.DeepEqual(repos, want) {
		t.Fatalf("Resolve() = %#v, want %#v", repos, want)
	}
}

func TestServiceResolveReturnsRegisteredRepos(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{
			Name:     "official",
			URL:      "https://github.com/org/skills",
			SkillDir: "skills",
		}},
	}}
	service := NewService(store)

	repos, err := service.Resolve("")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !reflect.DeepEqual(repos, store.cfg.Repositories) {
		t.Fatalf("Resolve() = %#v, want %#v", repos, store.cfg.Repositories)
	}
}

type memoryStore struct {
	cfg     config.Config
	loadErr error
	saveErr error
}

func (s *memoryStore) Load() (config.Config, error) {
	if s.loadErr != nil {
		return config.Config{}, s.loadErr
	}
	return s.cfg, nil
}

func (s *memoryStore) Save(cfg config.Config) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.cfg = cfg
	return nil
}
