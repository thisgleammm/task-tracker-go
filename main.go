package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// storeFile is the JSON database, kept in the current directory. Tests override it.
var storeFile = "tasks.json"

// stdout and stderr are indirected so tests can capture output.
var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

const timeLayout = "2006-01-02 15:04"

const usage = `task-cli - a simple task tracker

Usage:
  task-cli add "Buy groceries"
  task-cli update <id> "New description"
  task-cli delete <id>
  task-cli mark-in-progress <id>
  task-cli mark-done <id>
  task-cli list [todo|in-progress|done]

Tasks are stored in ./tasks.json`

// usageError marks a mistake in the command line, which is worth echoing usage for.
type usageError struct{ s string }

func (e *usageError) Error() string { return e.s }

// commands maps every supported command to its handler.
var commands = map[string]func(tasks []Task, args []string) error{
	"add":    addTask,
	"update": updateTask,
	"delete": deleteTask,
	"list":   listTasks,
	"mark-in-progress": func(tasks []Task, args []string) error {
		return markTask(tasks, args, "mark-in-progress", StatusInProgress)
	},
	"mark-done": func(tasks []Task, args []string) error {
		return markTask(tasks, args, "mark-done", StatusDone)
	},
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		var ue *usageError
		if errors.As(err, &ue) {
			fmt.Fprintf(stderr, "\n%s\n", usage)
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(stdout, usage)
		return nil
	}

	cmd, rest := args[0], args[1:]
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		fmt.Fprintln(stdout, usage)
		return nil
	}

	handler, ok := commands[cmd]
	if !ok {
		return &usageError{s: "unknown command " + quote(cmd)}
	}

	tasks, err := loadTasks(storeFile)
	if err != nil {
		return err
	}
	return handler(tasks, rest)
}

// addTask appends a new todo task.
func addTask(tasks []Task, args []string) error {
	if len(args) != 1 {
		return &usageError{s: `add requires exactly one description: task-cli add "Buy groceries"`}
	}
	description := strings.TrimSpace(args[0])
	if description == "" {
		return &usageError{s: "add requires a non-empty description"}
	}

	t := now()
	task := Task{
		ID:          nextID(tasks),
		Description: description,
		Status:      StatusTodo,
		CreatedAt:   t,
		UpdatedAt:   t,
	}
	if err := saveTasks(storeFile, append(tasks, task)); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Task added successfully (ID: %d)\n", task.ID)
	return nil
}

// updateTask replaces the description of an existing task.
func updateTask(tasks []Task, args []string) error {
	if len(args) != 2 {
		return &usageError{s: `update requires an id and a description: task-cli update 1 "New description"`}
	}
	description := strings.TrimSpace(args[1])
	if description == "" {
		return &usageError{s: "update requires a non-empty description"}
	}

	idx, err := findTask(tasks, args[0])
	if err != nil {
		return err
	}
	tasks[idx].Description = description
	tasks[idx].UpdatedAt = now()

	if err := saveTasks(storeFile, tasks); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Task %d updated successfully\n", tasks[idx].ID)
	return nil
}

// deleteTask removes a task from the list.
func deleteTask(tasks []Task, args []string) error {
	if len(args) != 1 {
		return &usageError{s: "delete requires exactly one task id: task-cli delete 1"}
	}

	idx, err := findTask(tasks, args[0])
	if err != nil {
		return err
	}
	deleted := tasks[idx]
	if err := saveTasks(storeFile, append(tasks[:idx:idx], tasks[idx+1:]...)); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Task %d deleted successfully\n", deleted.ID)
	return nil
}

// markTask moves a task to a new status.
func markTask(tasks []Task, args []string, cmd string, status Status) error {
	if len(args) != 1 {
		return &usageError{s: fmt.Sprintf("%s requires exactly one task id: task-cli %s 1", cmd, cmd)}
	}

	idx, err := findTask(tasks, args[0])
	if err != nil {
		return err
	}
	if tasks[idx].Status == status {
		fmt.Fprintf(stdout, "Task %d is already %s\n", tasks[idx].ID, status)
		return nil
	}
	tasks[idx].Status = status
	tasks[idx].UpdatedAt = now()

	if err := saveTasks(storeFile, tasks); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Task %d marked as %s\n", tasks[idx].ID, status)
	return nil
}

// listTasks prints every task, or only those matching an optional status filter.
func listTasks(tasks []Task, args []string) error {
	if len(args) > 1 {
		return &usageError{s: "list accepts at most one status filter: todo, in-progress or done"}
	}

	filter := ""
	if len(args) == 1 {
		status, err := parseStatus(args[0])
		if err != nil {
			return err
		}
		filter = string(status)
	}

	shown := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if filter == "" || string(task.Status) == filter {
			shown = append(shown, task)
		}
	}
	if len(shown) == 0 {
		fmt.Fprintln(stdout, "No tasks found.")
		return nil
	}

	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tDESCRIPTION\tCREATED AT\tUPDATED AT")
	for _, task := range shown {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			task.ID, task.Status, task.Description,
			task.CreatedAt.Local().Format(timeLayout), task.UpdatedAt.Local().Format(timeLayout))
	}
	return w.Flush()
}

// nextID returns the smallest unused id that is greater than every existing one.
func nextID(tasks []Task) int64 {
	var max int64
	for _, task := range tasks {
		if task.ID > max {
			max = task.ID
		}
	}
	return max + 1
}

// findTask resolves a command line id into an index into tasks.
func findTask(tasks []Task, rawID string) (int, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
	if err != nil || id <= 0 {
		return -1, &usageError{s: "invalid task id " + quote(rawID) + ", expected a positive integer"}
	}
	for i, task := range tasks {
		if task.ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no task found with id %d", id)
}

func quote(s string) string { return strconv.Quote(s) }

// now is the timestamp source for createdAt and updatedAt. Values are stored in
// UTC at second precision, which keeps tasks.json readable and RFC 3339 valid.
func now() time.Time { return time.Now().UTC().Truncate(time.Second) }
