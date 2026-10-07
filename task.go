package main

import "time"

// Status is the lifecycle state of a task.
type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in-progress"
	StatusDone       Status = "done"
)

// Task is a single entry of the to-do list.
type Task struct {
	ID          int64     `json:"id"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// parseStatus converts a user supplied string into a Status.
func parseStatus(s string) (Status, error) {
	switch Status(s) {
	case StatusTodo:
		return StatusTodo, nil
	case StatusInProgress:
		return StatusInProgress, nil
	case StatusDone:
		return StatusDone, nil
	default:
		return "", &usageError{s: "invalid status " + quote(s) + ", expected todo, in-progress or done"}
	}
}
