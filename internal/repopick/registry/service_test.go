package registry

import (
	"errors"
	"reflect"
	"testing"

	"github.com/finger/repo-pick/internal/repopick/config"
)

func TestServiceAddPersistsRepository(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)

	err := service.Add(config.Repository{
		Name:   "official",
		URL:    "https://github.com/org/tools",
		Branch: "dev",
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	want := []config.Repository{{
		Name:   "official",
		URL:    "https://github.com/org/tools",
		Branch: "dev",
	}}
	if !reflect.DeepEqual(store.cfg.Repositories, want) {
		t.Fatalf("Repositories = %#v, want %#v", store.cfg.Repositories, want)
	}
}

func TestServiceAddRejectsEmptyNameOrURL(t *testing.T) {
	service := NewService(&memoryStore{})

	tests := []struct {
		name string
		repo config.Repository
		want error
	}{
		{name: "name", repo: config.Repository{URL: "https://github.com/org/tools"}, want: ErrEmptyName},
		{name: "url", repo: config.Repository{Name: "official"}, want: ErrEmptyURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Add(tt.repo)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Add() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestServiceAddRejectsDuplicateName(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{
			Name: "official",
			URL:  "https://github.com/org/tools",
		}},
	}}
	service := NewService(store)

	err := service.Add(config.Repository{
		Name: "official",
		URL:  "https://github.com/other/tools",
	})
	if !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("Add() error = %v, want ErrDuplicateName", err)
	}
}

func TestServiceAddAllowsSameURLWithDifferentBranch(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{
			Name:   "official",
			URL:    "https://github.com/org/tools",
			Branch: "main",
		}},
	}}
	service := NewService(store)

	if err := service.Add(config.Repository{
		Name:   "dev",
		URL:    "https://github.com/org/tools",
		Branch: "dev",
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	want := []config.Repository{
		{Name: "official", URL: "https://github.com/org/tools", Branch: "main"},
		{Name: "dev", URL: "https://github.com/org/tools", Branch: "dev"},
	}
	if !reflect.DeepEqual(store.cfg.Repositories, want) {
		t.Fatalf("Repositories = %#v, want %#v", store.cfg.Repositories, want)
	}
}

func TestServiceAddRejectsDuplicateURLAndBranch(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{
			Name:   "official",
			URL:    "https://github.com/org/tools",
			Branch: "main",
		}},
	}}
	service := NewService(store)

	err := service.Add(config.Repository{
		Name:   "mirror",
		URL:    "https://github.com/org/tools",
		Branch: "main",
	})
	if !errors.Is(err, ErrDuplicateURLBranch) {
		t.Fatalf("Add() error = %v, want ErrDuplicateURLBranch", err)
	}
}

func TestServiceAddRejectsDuplicateURLWithDefaultBranch(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{
			Name: "official",
			URL:  "https://github.com/org/tools",
		}},
	}}
	service := NewService(store)

	err := service.Add(config.Repository{
		Name: "mirror",
		URL:  "https://github.com/org/tools",
	})
	if !errors.Is(err, ErrDuplicateURLBranch) {
		t.Fatalf("Add() error = %v, want ErrDuplicateURLBranch", err)
	}
}

func TestServiceUpdatePersistsRepository(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{
			Name: "official",
			URL:  "https://github.com/org/tools",
		}},
	}}
	service := NewService(store)

	err := service.Update("official", config.Repository{
		Name:   "official-dev",
		URL:    " https://github.com/org/tools ",
		Branch: " dev ",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	want := []config.Repository{{
		Name:   "official-dev",
		URL:    "https://github.com/org/tools",
		Branch: "dev",
	}}
	if !reflect.DeepEqual(store.cfg.Repositories, want) {
		t.Fatalf("Repositories = %#v, want %#v", store.cfg.Repositories, want)
	}
}

func TestServiceUpdateRejectsDuplicateName(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{
			{Name: "official", URL: "https://github.com/org/tools"},
			{Name: "personal", URL: "git@github.com:finger/my-tools.git"},
		},
	}}
	service := NewService(store)

	err := service.Update("official", config.Repository{
		Name: "personal",
		URL:  "https://github.com/org/tools",
	})
	if !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("Update() error = %v, want ErrDuplicateName", err)
	}
}

func TestServiceUpdateRejectsDuplicateURLAndBranch(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{
			{Name: "official", URL: "https://github.com/org/tools", Branch: "main"},
			{Name: "personal", URL: "git@github.com:finger/my-tools.git", Branch: "dev"},
		},
	}}
	service := NewService(store)

	err := service.Update("official", config.Repository{
		Name:   "official",
		URL:    "git@github.com:finger/my-tools.git",
		Branch: "dev",
	})
	if !errors.Is(err, ErrDuplicateURLBranch) {
		t.Fatalf("Update() error = %v, want ErrDuplicateURLBranch", err)
	}
}

func TestServiceRemoveDeletesByNameAndPersists(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{
			{Name: "official", URL: "https://github.com/org/tools"},
			{Name: "personal", URL: "git@github.com:finger/my-tools.git"},
		},
	}}
	service := NewService(store)

	if err := service.Remove("official"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	want := []config.Repository{{
		Name: "personal",
		URL:  "git@github.com:finger/my-tools.git",
	}}
	if !reflect.DeepEqual(store.cfg.Repositories, want) {
		t.Fatalf("Repositories = %#v, want %#v", store.cfg.Repositories, want)
	}
}

func TestServiceListReturnsNormalizedRepositories(t *testing.T) {
	store := &memoryStore{cfg: config.Config{
		Repositories: []config.Repository{{
			Name:   " official ",
			URL:    " https://github.com/org/tools ",
			Branch: " dev ",
		}},
	}}
	service := NewService(store)

	repos, err := service.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := []config.Repository{{Name: "official", URL: "https://github.com/org/tools", Branch: "dev"}}
	if !reflect.DeepEqual(repos, want) {
		t.Fatalf("List() = %#v, want %#v", repos, want)
	}
}

type memoryStore struct {
	// cfg 是测试内存配置。
	cfg config.Config
	// loadErr 是测试注入的读取错误。
	loadErr error
	// saveErr 是测试注入的保存错误。
	saveErr error
}

// Load 返回测试内存配置。
func (s *memoryStore) Load() (config.Config, error) {
	if s.loadErr != nil {
		return config.Config{}, s.loadErr
	}
	return s.cfg, nil
}

// Save 覆盖测试内存配置。
func (s *memoryStore) Save(cfg config.Config) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.cfg = cfg
	return nil
}
