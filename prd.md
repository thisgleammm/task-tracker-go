# PRD — Task Tracker CLI

> Product Requirements Document
> Source project page: https://roadmap.sh/projects/task-tracker
> Language: Go · Type: CLI · Level: beginner

## 1. Overview

Build a CLI app to track tasks and manage your to-do list.

Task tracker is a project used to track and manage your tasks. In this project you build a simple
command line interface to track what you need to do, what you have done, and what you are currently
working on. It practices filesystem access, handling user input, and building a small CLI app.

## 2. Goals / Non-Goals

**Goals**

- A `task-cli` binary driven entirely by positional command line arguments.
- Durable storage in a JSON file in the current directory, created on first use.
- Full lifecycle of a task: create, update, delete, change status, list.
- Graceful handling of errors and edge cases (bad input, missing file, corrupted JSON).

**Non-Goals**

- No GUI, no HTTP API, no database.
- No third-party dependencies — Go standard library only.
- No global flags or config files; arguments only.

## 3. Functional Requirements

The user must be able to:

1. Add tasks.
2. Update tasks.
3. Delete tasks.
4. Mark a task as in progress.
5. Mark a task as done.
6. List all tasks.
7. List all tasks that are done.
8. List all tasks that are not done (todo).
9. List all tasks that are in progress.

## 4. Constraints

- Any programming language may be used; this project uses Go.
- Use positional arguments to accept user input.
- Store tasks in a JSON file in the current directory.
- The JSON file must be created if it does not exist.
- Use the native file system module of the language to interact with the JSON file.
- Do not use external libraries or frameworks.
- Handle errors and edge cases gracefully.

## 5. Command Specification

```bash
# Adding a new task
task-cli add "Buy groceries"
# Output: Task added successfully (ID: 1)

# Updating and deleting tasks
task-cli update 1 "Buy groceries and cook dinner"
task-cli delete 1

# Marking a task as in progress or done
task-cli mark-in-progress 1
task-cli mark-done 1

# Listing all tasks
task-cli list

# Listing tasks by status
task-cli list done
task-cli list todo
task-cli list in-progress
```

### 5.1 Command Semantics

| Command             | Args              | Effect                                                      |
| ------------------- | ----------------- | ----------------------------------------------------------- |
| `add`               | `<description>`   | Appends a task with status `todo`, prints the assigned ID.   |
| `update`            | `<id> <desc>`     | Replaces the description, refreshes `updatedAt`.             |
| `delete`            | `<id>`            | Removes the task from the list.                              |
| `mark-in-progress`  | `<id>`            | Sets status to `in-progress`, refreshes `updatedAt`.         |
| `mark-done`         | `<id>`            | Sets status to `done`, refreshes `updatedAt`.                |
| `list`              | `[status]`        | Prints all tasks, or only those matching the status filter.  |

### 5.2 Error Behaviour

Every command exits with status `0` on success and non-zero on failure, printing a message to
stderr. Error cases handled explicitly:

- Missing, extra, or empty arguments for any command.
- Non-numeric or non-positive task IDs.
- IDs that do not exist.
- Unknown command name.
- Unknown status filter (`task-cli list archived`).
- Unreadable or corrupted `tasks.json` (invalid JSON) — the file is never overwritten in this case.

## 6. Data Model

Each task has the following properties, all persisted to the JSON file on create and refreshed on
update:

| Property      | Type     | Description                                        |
| ------------- | -------- | -------------------------------------------------- |
| `id`          | number   | Unique identifier for the task.                    |
| `description` | string   | A short description of the task.                   |
| `status`      | string   | One of `todo`, `in-progress`, `done`.              |
| `createdAt`   | RFC 3339 | Date and time when the task was created.           |
| `updatedAt`   | RFC 3339 | Date and time when the task was last updated.      |

Example `tasks.json`:

```json
[
  {
    "id": 1,
    "description": "Buy groceries",
    "status": "todo",
    "createdAt": "2026-10-07T14:20:31Z",
    "updatedAt": "2026-10-07T14:20:31Z"
  }
]
```

## 7. Technical Implementation

| Concern     | Decision                                                                 |
| ----------- | ------------------------------------------------------------------------ |
| Storage file | `./tasks.json`, created as `[]` on first run                             |
| IDs          | `max(existing id) + 1`, never reused                                      |
| Timestamps   | `time.Time` marshalled as RFC 3339 by `encoding/json`                    |
| Writes       | Temp file in the same directory + `os.Rename` (atomic, crash-safe)       |
| Code layout  | `main.go` (CLI/dispatch), `task.go` (model), `store.go` (persistence)     |
| Output       | Aligned table via `text/tabwriter`                                       |
| Dependencies  | Standard library only                                                    |

## 8. Acceptance Criteria

- [x] All nine functional requirements are reachable from the command line.
- [x] `tasks.json` is created automatically in the current directory.
- [x] Every property in the data model is written on create and kept current on update.
- [x] `task-cli add "Buy groceries"` prints `Task added successfully (ID: 1)`.
- [x] `list` filters correctly for `todo`, `in-progress`, and `done`.
- [x] Invalid arguments produce a clear message and a non-zero exit code.
- [x] No external libraries are used.
- [x] Unit tests cover storage round-trips, status parsing, ID allocation, and every command.

## 9. Getting Started

1. **Environment** — install Go 1.25+ and a code editor (VS Code / GoLand).
2. **Initialisation** — clone the repository; `go.mod` and dependencies are standard library only.
3. **Implement features** — start with argument parsing, then `add`, `list`, `update`, status
   changes, and `delete`, testing each before moving on.
4. **Test and debug** — run `go test ./...` and inspect `tasks.json` to verify persistence.
5. **Finalise** — keep code formatted with `gofmt`, comment non-obvious logic, and document usage
   in `README.md`.

Happy coding!
