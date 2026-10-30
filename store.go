package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	ErrNotFound  = errors.New("task not found")
	ErrAmbiguous = errors.New("reference matches several tasks")
	ErrInvalid   = errors.New("invalid task")
)

var (
	statuses   = []string{"todo", "in_progress", "done"}
	priorities = []string{"critical", "high", "medium", "low"}
)

type Task struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Plan        string    `json:"plan"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

// Patch is a partial update; nil fields are left untouched.
type Patch struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Plan        *string `json:"plan"`
	Priority    *string `json:"priority"`
	Status      *string `json:"status"`
}

type board struct {
	NextID int    `json:"next_id"`
	Tasks  []Task `json:"tasks"`
}

// Store keeps the whole board in one JSON file; every call re-reads it, so the
// web server and OneDrive edits all see the same state.
type Store struct{ path, lock string }

func NewStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	// Lock lives in the temp dir so it is not synced with the board file.
	h := fnv.New64a()
	h.Write([]byte(abs))
	return &Store{path: abs, lock: filepath.Join(os.TempDir(), fmt.Sprintf("kainban-%x.lock", h.Sum64()))}, nil
}

func (s *Store) with(write bool, fn func(*board) error) error {
	lf, err := os.OpenFile(s.lock, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lf.Close()
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)

	b := board{NextID: 1}
	data, err := os.ReadFile(s.path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &b); err != nil {
			return fmt.Errorf("parse %s: %w", s.path, err)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	if err := fn(&b); err != nil || !write {
		return err
	}
	out, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	// Sync before rename so a crash cannot leave an empty or cut-off board.
	if _, err := f.Write(out); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func rank(list []string, v string) int {
	for i, x := range list {
		if x == v {
			return i
		}
	}
	return -1
}

// resolve finds a task by id ("12" or "#12") or by case-insensitive title text.
func (b *board) resolve(ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	if id, err := strconv.Atoi(strings.TrimPrefix(ref, "#")); err == nil {
		for i, t := range b.Tasks {
			if t.ID == id {
				return i, nil
			}
		}
		return 0, ErrNotFound
	}
	q := strings.ToLower(ref)
	found := -1
	for i, t := range b.Tasks {
		title := strings.ToLower(t.Title)
		if title == q {
			return i, nil
		}
		if q != "" && strings.Contains(title, q) {
			if found >= 0 {
				return 0, ErrAmbiguous
			}
			found = i
		}
	}
	if found < 0 {
		return 0, ErrNotFound
	}
	return found, nil
}

func apply(t *Task, p Patch) error {
	if p.Title != nil {
		t.Title = strings.TrimSpace(*p.Title)
	}
	if p.Description != nil {
		t.Description = *p.Description
	}
	if p.Plan != nil {
		t.Plan = *p.Plan
	}
	if p.Priority != nil {
		t.Priority = *p.Priority
	}
	if p.Status != nil {
		t.Status = *p.Status
	}
	switch {
	case t.Title == "":
		return fmt.Errorf("%w: title is required", ErrInvalid)
	case rank(priorities, t.Priority) < 0:
		return fmt.Errorf("%w: priority must be one of %v", ErrInvalid, priorities)
	case rank(statuses, t.Status) < 0:
		return fmt.Errorf("%w: status must be one of %v", ErrInvalid, statuses)
	}
	t.Updated = time.Now().UTC()
	return nil
}

// List returns tasks ordered by priority then id; q matches title, description and plan.
func (s *Store) List(q, status string) ([]Task, error) {
	var out []Task
	err := s.with(false, func(b *board) error {
		q = strings.ToLower(q)
		for _, t := range b.Tasks {
			if status != "" && t.Status != status {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(t.Title+"\n"+t.Description+"\n"+t.Plan), q) {
				continue
			}
			out = append(out, t)
		}
		return nil
	})
	sort.SliceStable(out, func(i, j int) bool {
		if ri, rj := rank(priorities, out[i].Priority), rank(priorities, out[j].Priority); ri != rj {
			return ri < rj
		}
		return out[i].ID < out[j].ID
	})
	return out, err
}

func (s *Store) Get(ref string) (t Task, err error) {
	err = s.with(false, func(b *board) error {
		i, err := b.resolve(ref)
		if err == nil {
			t = b.Tasks[i]
		}
		return err
	})
	return
}

func (s *Store) Create(p Patch) (t Task, err error) {
	err = s.with(true, func(b *board) error {
		now := time.Now().UTC()
		t = Task{ID: b.NextID, Priority: "medium", Status: "todo", Created: now}
		if err := apply(&t, p); err != nil {
			return err
		}
		b.NextID++
		b.Tasks = append(b.Tasks, t)
		return nil
	})
	return
}

func (s *Store) Update(ref string, p Patch) (t Task, err error) {
	err = s.with(true, func(b *board) error {
		i, err := b.resolve(ref)
		if err != nil {
			return err
		}
		t = b.Tasks[i]
		if err := apply(&t, p); err != nil {
			return err
		}
		b.Tasks[i] = t
		return nil
	})
	return
}

func (s *Store) Delete(ref string) error {
	return s.with(true, func(b *board) error {
		i, err := b.resolve(ref)
		if err != nil {
			return err
		}
		b.Tasks = append(b.Tasks[:i], b.Tasks[i+1:]...)
		return nil
	})
}
