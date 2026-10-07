package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setup points the store at a temporary file and discards command output.
func setup(t *testing.T) string {
	t.Helper()

	prevFile, prevOut, prevErr := storeFile, stdout, stderr
	path := filepath.Join(t.TempDir(), "tasks.json")
	storeFile, stdout, stderr = path, io.Discard, io.Discard
	t.Cleanup(func() { storeFile, stdout, stderr = prevFile, prevOut, prevErr })
	return path
}

// capture runs a command that must succeed and returns what it printed.
func capture(t *testing.T, args ...string) string {
	t.Helper()

	var buf bytes.Buffer
	prev := stdout
	stdout = &buf
	defer func() { stdout = prev }()

	if err := run(args); err != nil {
		t.Fatalf("run(%q) returned error: %v", args, err)
	}
	return buf.String()
}

// fails runs a command that must fail and returns its error.
func fails(t *testing.T, args ...string) error {
	t.Helper()

	prev := stdout
	stdout = io.Discard
	defer func() { stdout = prev }()

	err := run(args)
	if err == nil {
		t.Fatalf("run(%q) = nil, want an error", args)
	}
	return err
}

func storedTasks(t *testing.T, path string) []Task {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		t.Fatalf("tasks.json is not valid JSON: %v", err)
	}
	return tasks
}

func sampleTask(id int64, status Status) Task {
	now := time.Now()
	return Task{
		ID:          id,
		Description: "task",
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestLoadTasksCreatesMissingFile(t *testing.T) {
	path := setup(t)

	tasks, err := loadTasks(path)
	if err != nil {
		t.Fatalf("loadTasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("got %d tasks, want 0", len(tasks))
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to be created: %v", path, err)
	}
}

func TestLoadTasksEmptyFile(t *testing.T) {
	path := setup(t)
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("writing empty file: %v", err)
	}

	tasks, err := loadTasks(path)
	if err != nil {
		t.Fatalf("loadTasks on empty file: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("got %d tasks, want 0", len(tasks))
	}
}

func TestLoadTasksInvalidJSON(t *testing.T) {
	path := setup(t)
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("writing broken file: %v", err)
	}

	if _, err := loadTasks(path); err == nil {
		t.Fatal("loadTasks = nil error, want invalid JSON error")
	}
}

func TestSaveTasksRoundTrip(t *testing.T) {
	path := setup(t)
	want := []Task{sampleTask(1, StatusTodo), sampleTask(2, StatusDone)}

	if err := saveTasks(path, want); err != nil {
		t.Fatalf("saveTasks: %v", err)
	}
	got, err := loadTasks(path)
	if err != nil {
		t.Fatalf("loadTasks: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d tasks, want %d", len(got), len(want))
	}
	if got[0].ID != want[0].ID || got[0].Status != want[0].Status ||
		!got[0].CreatedAt.Equal(want[0].CreatedAt) {
		t.Errorf("task mismatch:\n got %+v\nwant %+v", got[0], want[0])
	}
}

func TestSaveTasksWritesJSONArrayForEmptyList(t *testing.T) {
	path := setup(t)

	if err := saveTasks(path, nil); err != nil {
		t.Fatalf("saveTasks(nil): %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != "[]" {
		t.Errorf("got %q, want %q", got, "[]")
	}
}

func TestParseStatus(t *testing.T) {
	for _, in := range []string{"todo", "in-progress", "done"} {
		status, err := parseStatus(in)
		if err != nil {
			t.Errorf("parseStatus(%q) returned error: %v", in, err)
		}
		if string(status) != in {
			t.Errorf("parseStatus(%q) = %q, want %q", in, status, in)
		}
	}
	if _, err := parseStatus("archived"); err == nil {
		t.Error("parseStatus(\"archived\") = nil error, want error")
	}
}

func TestNextID(t *testing.T) {
	tests := []struct {
		name  string
		tasks []Task
		want  int64
	}{
		{"empty list", nil, 1},
		{"single task", []Task{{ID: 1}}, 2},
		{"gaps are not reused", []Task{{ID: 1}, {ID: 7}}, 8},
		{"out of order", []Task{{ID: 9}, {ID: 3}}, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextID(tt.tasks); got != tt.want {
				t.Errorf("nextID() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFindTask(t *testing.T) {
	tasks := []Task{{ID: 1}, {ID: 2}}

	if idx, err := findTask(tasks, "2"); err != nil || idx != 1 {
		t.Errorf("findTask(\"2\") = %d, %v, want 1, nil", idx, err)
	}
	for _, bad := range []string{"abc", "0", "-1", ""} {
		if _, err := findTask(tasks, bad); err == nil {
			t.Errorf("findTask(%q) = nil error, want error", bad)
		}
	}
	if _, err := findTask(tasks, "99"); err == nil {
		t.Error("findTask(\"99\") = nil error, want not-found error")
	}
}

func TestAddCreatesTodoTask(t *testing.T) {
	path := setup(t)

	out := capture(t, "add", "Buy groceries")
	if !strings.Contains(out, "Task added successfully (ID: 1)") {
		t.Errorf("unexpected output: %q", out)
	}

	tasks := storedTasks(t, path)
	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}
	got := tasks[0]
	if got.ID != 1 || got.Description != "Buy groceries" || got.Status != StatusTodo {
		t.Errorf("unexpected task: %+v", got)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Errorf("timestamps not set: %+v", got)
	}
	if !got.CreatedAt.Equal(got.UpdatedAt) {
		t.Error("createdAt and updatedAt should match on a new task")
	}
}

func TestAddTrimsAndRejectsEmptyDescription(t *testing.T) {
	path := setup(t)

	if out := capture(t, "add", "  spaced  "); !strings.Contains(out, "(ID: 1)") {
		t.Errorf("unexpected output: %q", out)
	}
	if got := storedTasks(t, path)[0].Description; got != "spaced" {
		t.Errorf("description = %q, want %q", got, "spaced")
	}

	for _, args := range [][]string{{"add"}, {"add", ""}, {"add", "   "}, {"add", "a", "b"}} {
		if err := fails(t, args...); !errors.As(err, new(*usageError)) {
			t.Errorf("%q error = %v, want a usage error", args, err)
		}
	}
	if got := len(storedTasks(t, path)); got != 1 {
		t.Errorf("stored %d tasks, want 1", got)
	}
}

func TestAddNeverReusesDeletedIDs(t *testing.T) {
	path := setup(t)

	capture(t, "add", "first")
	capture(t, "add", "second")
	capture(t, "delete", "1")

	if out := capture(t, "add", "third"); !strings.Contains(out, "(ID: 3)") {
		t.Errorf("got %q, want ID 3", out)
	}
	tasks := storedTasks(t, path)
	if len(tasks) != 2 || tasks[0].ID != 2 || tasks[1].ID != 3 {
		t.Errorf("unexpected task list: %+v", tasks)
	}
}

func TestUpdateChangesDescription(t *testing.T) {
	path := setup(t)
	capture(t, "add", "Buy groceries")

	if out := capture(t, "update", "1", "Buy groceries and cook dinner"); !strings.Contains(out, "Task 1 updated") {
		t.Errorf("unexpected output: %q", out)
	}

	tasks := storedTasks(t, path)
	if tasks[0].Description != "Buy groceries and cook dinner" {
		t.Errorf("description = %q", tasks[0].Description)
	}
	if tasks[0].Status != StatusTodo {
		t.Errorf("status = %q, want unchanged todo", tasks[0].Status)
	}
	if !tasks[0].UpdatedAt.After(tasks[0].CreatedAt) && !tasks[0].UpdatedAt.Equal(tasks[0].CreatedAt) {
		t.Error("updatedAt should not be older than createdAt")
	}
}

func TestUpdateRejectsBadInput(t *testing.T) {
	setup(t)
	capture(t, "add", "task")

	for _, args := range [][]string{
		{"update"},
		{"update", "1"},
		{"update", "1", ""},
		{"update", "abc", "new"},
		{"update", "42", "new"},
	} {
		if err := fails(t, args...); err == nil {
			t.Errorf("%q should have failed", args)
		}
	}
}

func TestDeleteRemovesTask(t *testing.T) {
	path := setup(t)
	capture(t, "add", "first")
	capture(t, "add", "second")

	if out := capture(t, "delete", "1"); !strings.Contains(out, "Task 1 deleted") {
		t.Errorf("unexpected output: %q", out)
	}

	tasks := storedTasks(t, path)
	if len(tasks) != 1 || tasks[0].ID != 2 {
		t.Fatalf("unexpected task list: %+v", tasks)
	}
}

func TestMarkStatus(t *testing.T) {
	tests := []struct {
		cmd    string
		status Status
	}{
		{"mark-in-progress", StatusInProgress},
		{"mark-done", StatusDone},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			path := setup(t)
			capture(t, "add", "task")

			if out := capture(t, tt.cmd, "1"); !strings.Contains(out, string(tt.status)) {
				t.Errorf("unexpected output: %q", out)
			}
			if got := storedTasks(t, path)[0].Status; got != tt.status {
				t.Errorf("status = %q, want %q", got, tt.status)
			}
			if out := capture(t, tt.cmd, "1"); !strings.Contains(out, "already") {
				t.Errorf("second call should report no change, got %q", out)
			}
		})
	}
}

func TestMarkRejectsBadInput(t *testing.T) {
	setup(t)
	capture(t, "add", "task")

	for _, args := range [][]string{
		{"mark-done"},
		{"mark-done", "1", "2"},
		{"mark-in-progress", "xyz"},
		{"mark-done", "7"},
	} {
		if err := fails(t, args...); err == nil {
			t.Errorf("%q should have failed", args)
		}
	}
}

func TestListFiltersByStatus(t *testing.T) {
	setup(t)
	capture(t, "add", "todo one")
	capture(t, "add", "working")
	capture(t, "add", "finished")
	capture(t, "mark-in-progress", "2")
	capture(t, "mark-done", "3")

	all := capture(t, "list")
	for _, want := range []string{"todo one", "working", "finished"} {
		if !strings.Contains(all, want) {
			t.Errorf("list is missing %q:\n%s", want, all)
		}
	}
	for _, want := range []string{"ID", "STATUS", "DESCRIPTION"} {
		if !strings.Contains(all, want) {
			t.Errorf("list header is missing %q:\n%s", want, all)
		}
	}

	tests := []struct {
		filter string
		want   []string
		absent []string
	}{
		{"todo", []string{"todo one"}, []string{"working", "finished"}},
		{"in-progress", []string{"working"}, []string{"todo one", "finished"}},
		{"done", []string{"finished"}, []string{"todo one", "working"}},
	}
	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			out := capture(t, "list", tt.filter)
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("list %s is missing %q:\n%s", tt.filter, want, out)
				}
			}
			for _, absent := range tt.absent {
				if strings.Contains(out, absent) {
					t.Errorf("list %s should not contain %q:\n%s", tt.filter, absent, out)
				}
			}
		})
	}
}

func TestListWithoutTasks(t *testing.T) {
	setup(t)

	if out := capture(t, "list"); !strings.Contains(out, "No tasks found") {
		t.Errorf("unexpected output: %q", out)
	}
	capture(t, "add", "done one")
	capture(t, "mark-done", "1")
	if out := capture(t, "list", "todo"); !strings.Contains(out, "No tasks found") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestListRejectsBadFilter(t *testing.T) {
	setup(t)
	capture(t, "add", "task")

	for _, args := range [][]string{{"list", "archived"}, {"list", "todo", "done"}} {
		if err := fails(t, args...); err == nil {
			t.Errorf("%q should have failed", args)
		}
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	setup(t)

	err := fails(t, "deletee", "1")
	if !errors.As(err, new(*usageError)) {
		t.Errorf("error = %v, want a usage error", err)
	}
}

func TestRunWithoutArgsPrintsUsage(t *testing.T) {
	setup(t)

	for _, args := range [][]string{nil, {"help"}, {"-h"}, {"--help"}} {
		if out := capture(t, args...); !strings.Contains(out, "task-cli add") {
			t.Errorf("%q did not print usage, got %q", args, out)
		}
	}
}

func TestHelpDoesNotCreateStoreFile(t *testing.T) {
	path := setup(t)

	capture(t, "--help")
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("help should not create %s", path)
	}
}

func TestRunDoesNotOverwriteCorruptedStore(t *testing.T) {
	path := setup(t)
	const broken = "{not json"
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatalf("writing broken file: %v", err)
	}

	if err := fails(t, "add", "task"); err == nil {
		t.Fatal("add on corrupted store should fail")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(data) != broken {
		t.Errorf("store was modified: %q", data)
	}
}

func TestStoreFilePermissions(t *testing.T) {
	path := setup(t)
	capture(t, "add", "verify file mode")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("expected file mode 0644, got %o", info.Mode().Perm())
	}
}
