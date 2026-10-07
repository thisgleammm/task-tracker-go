package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// loadTasks reads the task list from path. A missing file is created as an
// empty list so a fresh checkout works without any setup step.
func loadTasks(path string) ([]Task, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		tasks := []Task{}
		if err := saveTasks(path, tasks); err != nil {
			return nil, fmt.Errorf("could not create %s: %w", path, err)
		}
		return tasks, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}
	if len(data) == 0 {
		return []Task{}, nil
	}

	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("%s contains invalid JSON: %w", path, err)
	}
	return tasks, nil
}

// saveTasks writes tasks to path atomically: the new content lands in a
// temporary file that is renamed over the target, so an interrupted write can
// never truncate the existing list.
func saveTasks(path string, tasks []Task) error {
	if tasks == nil {
		tasks = []Task{}
	}
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode tasks: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), ".tasks-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
