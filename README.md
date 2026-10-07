# Task Tracker CLI

A small command line task manager built with Go and the standard library only. Tasks are stored in a
JSON file in the current directory, so there is no server, no database, and nothing to install.

Built as a learning project from the roadmap.sh specification.

**Project page:** https://roadmap.sh/projects/task-tracker

## Features

- Add, update, and delete tasks.
- Mark a task as `in-progress` or `done`.
- List every task, or filter by `todo`, `in-progress`, and `done`.
- Persistent JSON storage that is created automatically on first run.
- Explicit errors for bad arguments, missing IDs, unknown commands, and corrupted storage.
- Zero third-party dependencies.

## Requirements

- Go 1.25 or newer ([download](https://go.dev/dl/))

Check your version:

```bash
go version
```

## How to run

Build the binary and run it from the directory where you want the task list to live:

```bash
go build -o task-cli .
./task-cli add "Buy groceries"
```

To get `task-cli` on your `PATH` instead, install the binary into your Go bin directory:

```bash
go build -o "$(go env GOPATH)/bin/task-cli" .
task-cli add "Buy groceries"
```

`go run .` works too, for a quick try without keeping a binary around:

```bash
go run . add "Buy groceries"
```

Every command works exactly the same way; only the arguments change.

## Usage

```
task-cli add "Buy groceries"
task-cli update <id> "New description"
task-cli delete <id>
task-cli mark-in-progress <id>
task-cli mark-done <id>
task-cli list [todo|in-progress|done]
```

Run `task-cli help` to print the same summary, or call it with no arguments.

### Add a task

```bash
$ task-cli add "Buy groceries"
Task added successfully (ID: 1)
```

New tasks always start with status `todo`.

### List tasks

```bash
$ task-cli list
ID  STATUS       DESCRIPTION                    CREATED AT        UPDATED AT
1   todo         Buy groceries                  2026-10-07 14:24  2026-10-07 14:24
2   in-progress  Write README                   2026-10-07 14:24  2026-10-07 14:25
3   done         Ship release                   2026-10-07 14:24  2026-10-07 14:26
```

### Filter by status

```bash
task-cli list todo
task-cli list in-progress
task-cli list done
```

Each filter shows the same table restricted to matching tasks. When nothing matches, the command
prints `No tasks found.`

### Update a task

```bash
$ task-cli update 1 "Buy groceries and cook dinner"
Task 1 updated successfully
```

### Change status

```bash
$ task-cli mark-in-progress 1
Task 1 marked as in-progress

$ task-cli mark-done 1
Task 1 marked as done
```

Marking a task with the status it already has reports that nothing changed:

```bash
$ task-cli mark-done 1
Task 1 is already done
```

### Delete a task

```bash
$ task-cli delete 1
Task 1 deleted successfully
```

Task IDs are never reused. If the highest ID is 3 and you delete task 2, the next task you add gets
ID 4.

## Task data

Tasks live in `tasks.json` in the current directory. The file is created as an empty list on the
first command that needs it, and you can read or edit it by hand.

```json
[
  {
    "id": 1,
    "description": "Buy groceries",
    "status": "todo",
    "createdAt": "2026-10-07T07:24:08Z",
    "updatedAt": "2026-10-07T07:24:08Z"
  }
]
```

| Property      | Description                                          |
| ------------- | ---------------------------------------------------- |
| `id`          | Unique identifier, assigned automatically.           |
| `description` | Short description of the task.                       |
| `status`      | One of `todo`, `in-progress`, `done`.                |
| `createdAt`   | When the task was created, RFC 3339 in UTC.          |
| `updatedAt`   | When the task last changed, RFC 3339 in UTC.         |

Timestamps are stored in UTC and printed in your local time zone by `list`.

Because the file is resolved relative to the working directory, running the CLI from a different
folder gives you a different task list. That is intentional: each project directory can keep its own
tasks.

## Error handling

Every command exits with status `0` on success and `1` on failure, and prints the reason to stderr.
Usage mistakes additionally print the help text:

```bash
$ task-cli mark-done 99
Error: no task found with id 99

$ task-cli list archived
Error: invalid status "archived", expected todo, in-progress or done
```

Handled cases include missing or extra arguments, empty descriptions, non-numeric IDs, IDs that do
not exist, unknown commands, unknown status filters, and a `tasks.json` that contains invalid JSON.
A corrupted file is reported, never overwritten. Writes go to a temporary file that is renamed over
the target, so an interrupted write cannot truncate your list.

## Tests

```bash
go test ./...
```

The test suite covers storage round-trips, automatic file creation, corrupted JSON, status parsing,
ID allocation, every command, and the error paths.

## Project structure

| File            | Responsibility                                    |
| --------------- | ------------------------------------------------- |
| `main.go`       | Argument parsing, command dispatch, output.       |
| `task.go`       | `Task` model, statuses, status parsing.           |
| `store.go`      | Loading and atomically saving `tasks.json`.       |
| `main_test.go` | Tests for all of the above.                       |
| `prd.md`        | Product requirements this project implements.     |

```bash
gofmt -l .   # check formatting
go vet ./...  # static checks
```

## Learn more

- Project specification: https://roadmap.sh/projects/task-tracker
- Community solutions and feedback: https://roadmap.sh/projects/task-tracker/solutions
- Go standard library docs: https://pkg.go.dev/std
