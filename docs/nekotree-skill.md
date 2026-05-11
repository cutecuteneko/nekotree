# Nekotree Agent Skill Reference

**Nekotree** creates isolated Git worktree + Docker container pairs for on-demand development environments. This document is intended for Claude agents operating in a repository that has nekotree available. It describes the exact CLI interface and the expected workflow for implementing features safely.

---

## Prerequisites

Before using any nekotree command, verify the binary is available:

```bash
# Check binary location (try in order)
./build/nekotree --version
nekotree --version
```

If neither works, ask the user where the binary is located before proceeding. Do not assume a path.

If not found and you are inside the nekotree repository itself, build it:

```bash
make build   # or: go run scripts/build.go build
```

The binary will be at `build/nekotree`. Reference it by its full path if it is not on `$PATH`.

---

## CLI Reference

All commands are invoked as:

```
nekotree <command> [arguments] [options]
```

### `create` — Create a new environment

```
nekotree create <branch> [image|compose-file] [command...] [-f flag] [-e env-file]
```

| Argument | Required | Description |
|---|---|---|
| `branch` | Yes | Branch name for the worktree. Must be alphanumeric with `-`, `_`, `.` only. Keep short for readability. |
| `image\|compose-file` | No | Docker image (e.g. `alpine:latest`, `node:18`) or path to a compose file |
| `command` | No | Command to run inside the container. Defaults to `tail -f /dev/null` if no compose file (keeps container alive on any POSIX image). |
| `-f`, `--flag` | No | Raw Docker flags (e.g. `-f "-p 8080:3000"`). Repeatable. |
| `-e`, `--env` | No | Path to a `.env` file forwarded as `--env-file` to Docker. When a compose file is used, defaults to `<compose-dir>/.env` if that file exists. |

**What it does:**
1. Creates a Git worktree for `branch` at `nekotree-<repo>-<branch>/` inside the current repo root
2. Starts a Docker container named `nekotree-<repo>-<branch>`
3. Mounts the worktree at `/workspace` inside the container
4. Runs the specified command, or defaults to `tail -f /dev/null` to keep the container alive indefinitely

**Examples:**

```bash
# Minimal — worktree + container with default keep-alive
nekotree create feature-login alpine:latest

# With a specific startup command
nekotree create feature-login node:18 npm start

# With port mapping
nekotree create feature-login node:18 -f "-p 8080:3000" npm start

# With a compose file (no default command)
nekotree create feature-login docker-compose.yaml

# Compose with an override command
nekotree create feature-login docker-compose.yaml npm start

# With an explicit .env file
nekotree create feature-login node:18 -e .env.local npm start

# Compose with auto-detected .env (nekotree looks for <compose-dir>/.env automatically)
nekotree create feature-login docker-compose.yaml
```

**Stdout on success:**
```
🐳 Launching environment: nekotree-<repo>-<branch>
```

---

### `run` — Execute a command in an existing environment

```
nekotree run <branch> <command...>
nekotree r <branch> <command...>
```

| Argument | Required | Description |
|---|---|---|
| `branch` | Yes | The branch name of an existing environment |
| `command` | Yes | The command to run inside the container |

If the container does not exist but the worktree does, nekotree will start a new container (`alpine:latest` with `tail -f /dev/null`) automatically before running the command.

If the container shows as `Exited` in `nekotree list`, running any `nekotree run` command will auto-restart it — no need to manually remove and recreate.

**Examples:**

```bash
nekotree run feature-login npm test
nekotree run feature-login go build ./...
```

---

### `shell` — Open an interactive shell

```
nekotree shell <branch>
nekotree sh <branch>
```

Opens an interactive shell inside the running container. Prefers `bash` if available, falls back to `sh`.

**Example:**

```bash
nekotree shell feature-login
```

> **Note:** This requires an interactive TTY. Do not use this in non-interactive agent contexts — use `run` instead.

---

### `list` — List all environments

```
nekotree list
nekotree ls
```

Lists all Docker containers whose names start with `nekotree-`, showing name, status, and image.

**Example output:**
```
NAMES                          STATUS          IMAGE
nekotree-myrepo-feature-login  Up 2 hours      node:18
nekotree-myrepo-fix-auth       Exited (0)      alpine:latest
```

If no environments exist:
```
🌳 No active nekotree environments found.
```

---

### `remove` — Remove an environment

```
nekotree remove <name-or-branch>
nekotree rm <name-or-branch>
```

Accepts either the bare branch name (e.g. `feature-login`) or the full container name (e.g. `nekotree-myrepo-feature-login`).

**What it does:**
1. Stops and removes the Docker container
2. Removes the Git worktree directory
3. The **Git branch is preserved** — only the worktree checkout and container are removed

**Example:**

```bash
nekotree remove feature-login
```

---

## Standard Feature Workflow

When a user asks you to implement a feature, follow this workflow exactly:

### Step 1 — Safety checks (REQUIRED before any work)

```bash
# 1a. Check current branch — working on main/master is fine, nekotree creates an isolated worktree
git branch --show-current
# If on a feature branch already, confirm it's not a leftover nekotree worktree

# 1b. Confirm binary is available
nekotree --version
```

**If `main`/`master` has no commits, `create` will fail.** nekotree uses `git worktree add` under the hood, which requires a valid HEAD. If the repo has no commits yet, make one before proceeding:

```bash
git commit --allow-empty -m "Initial commit"
```

### Step 2 — Create the environment

Name the branch descriptively from the user's request. Use only `a-z`, `0-9`, `-`.

```bash
nekotree create <branch> <image>
# Example: nekotree create feature-user-auth node:18
```

**After `create`, always verify the container is actually running before proceeding:**

```bash
nekotree list
# Confirm the environment shows "Up" — not "Exited" or missing entirely
```

If it shows `Exited` or is absent, check the error output from `create` and resolve before continuing.

**Port exposure must be declared at create time:**

If the feature requires any ports to be accessible on the host (e.g. to test a web server), pass `-f "-p host:container"` at `create` time. Ports cannot be added to a running container — you would need to remove and recreate it.

```bash
nekotree create feature-login node:18 -f "-p 8080:3000"
# Multiple ports: use -f once per mapping
nekotree create feature-login node:18 -f "-p 8080:3000" -f "-p 9229:9229"
```

**Check for a repo config file first:**

If the repo contains a `nekotree-config.json` at its root, nekotree will read it automatically. You do not need to pass the compose file explicitly — it's already wired in:

```json
{ "compose_file": "docker-compose.yaml" }
```

If the file is absent, nekotree falls back to plain Docker with the image you provide.

**If the environment already exists:**

If `nekotree create` is called and the worktree already exists, it will link to the existing branch and print an info message — it is **not** an error. The container will still be started. This is safe to re-run.

**Additional volume mounts (optional):**

If the feature requires access to paths outside the worktree (e.g. a shared package cache), set `DEVENV_MOUNTS` before running `create`:

```bash
export DEVENV_MOUNTS="/host/cache:/cache:ro,/host/data:/data"
nekotree create feature-login node:18
```

Format: comma-separated `host:container` or `host:container:ro` pairs.

**Path safety rules (ENFORCED):**
- Never mount `/`, `/etc`, `/home`, `/root`, `/var`, `/usr`, `/sys`, `/proc`, or any system directory
- The worktree path (auto-managed by nekotree at `nekotree-<repo>-<branch>/`) is the only mount
- If using `-f` flags with `-v`, only mount paths within the project directory

**Shell injection rules (ENFORCED):**
- Branch names must match `^[a-zA-Z0-9._-]+$` — nekotree will reject anything else
- Do not construct branch names from unsanitized user input with special characters
- Do not pass user-supplied strings directly into the `command` argument without validation

### Step 3 — Implement the feature

Work inside the worktree directory. Use `nekotree run` to execute commands inside the container:

```bash
nekotree run <branch> <build-or-test-command>
```

The command tail is joined and passed to `sh -c` inside the container, so compound commands work when quoted:

```bash
# Single command — no quoting needed
nekotree run feature-login go build ./...

# Compound command — quote the whole thing so && is passed to sh, not the local shell
nekotree run feature-login "cd /workspace && go build ./... && go test ./..."
```

**Working directory:** The default working directory inside the container is determined by the image (e.g. `/go` for `golang` images, `/app` for some node images, `/root` for `alpine`). It is NOT automatically set to `/workspace`. Always check the image's default or prefix commands with `cd /workspace &&` to ensure you are operating on the worktree.

Do not use `nekotree shell` in automated agent contexts — use `run` for all non-interactive commands. Only fall back to `docker exec` directly if `nekotree run` itself is unavailable or broken.

### Step 4 — Build and test (REQUIRED before committing)

Detect the target repo's build/test convention and run it:

```bash
# Check what's available
ls Makefile package.json go.mod Cargo.toml pyproject.toml 2>/dev/null

# Run tests via nekotree run
nekotree run <branch> make test         # if Makefile present
nekotree run <branch> go test ./...     # if go.mod present
nekotree run <branch> npm test          # if package.json present
```

**If tests fail:**
- Do NOT commit
- Do NOT clean up the environment
- Report the failure output to the user and wait for instructions

### Step 5 — Commit inside the worktree (REQUIRED before cleanup)

The commit goes inside the feature branch — not on main. Use `git -C` to avoid relying on shell `cd`, which does not persist between agent tool calls:

```bash
# Check what will be staged before adding
git -C nekotree-<repo>-<branch>/ status

# Stage specific files rather than -A to avoid committing build artifacts or binaries
git -C nekotree-<repo>-<branch>/ add <file1> <file2>

git -C nekotree-<repo>-<branch>/ commit -m "<descriptive message summarizing the change>"
```

Rules:
- Commit message must be non-empty and descriptive
- Do not use `--no-verify` to skip hooks
- Do not amend a previous commit unless explicitly requested

### Step 6 — Confirm with user before teardown (REQUIRED)

Before running `remove`, confirm:

> "Feature branch `<branch>` is committed. Ready to remove the worktree and container? The branch will be preserved and can be checked out or PR'd at any time."

Only proceed with `remove` after the user confirms.

### Step 7 — Clean up

```bash
nekotree remove <branch>
```

**Cleanup rules:**
- Only clean up on success (tests passed, commit made, user confirmed)
- If any step failed, leave the environment intact for debugging
- After `remove`, verify with `nekotree list` that the environment is gone

---

## Quick Reference

| Command | What it does |
|---|---|
| `nekotree create <branch> <image> [cmd]` | Create worktree + container (use `-e` for .env file, `-f` for raw Docker flags) |
| `nekotree run <branch> <cmd>` | Execute command in container (alias: `r`) |
| `nekotree shell <branch>` | Interactive shell (TTY required) (alias: `sh`, `s`) |
| `nekotree list` | List all nekotree containers (alias: `ls`) |
| `nekotree remove <branch>` | Tear down container + worktree (branch preserved) (alias: `rm`) |

---

## Error Handling

### Common failures and what to do

| Situation | What you see | Action |
|---|---|---|
| Docker not running | `docker run failed: ...` | Report to user — Docker must be started manually |
| Branch name invalid | `invalid name: ...` | Rename the branch using only `a-z`, `0-9`, `-` |
| Worktree already exists | Info message, continues | Safe — nekotree links to the existing branch and proceeds |
| Container shows `Exited` in `list` | Status column: `Exited (0)` | Run `nekotree run <branch> <cmd>` — it auto-restarts the container |
| No worktree and no container | `worktree not found for branch: ...` | The environment was removed; use `create` again |
| `create` fails partway | Partial state (worktree created, container failed) | Run `nekotree remove <branch>` to clean up, then retry |

### Never silently continue after an error

If any `nekotree` command exits non-zero:
- Stop immediately
- Print the full error output to the user
- Do **not** commit or clean up
- Wait for user instructions

---

## Enforced Rules Summary

| Rule | Description |
|---|---|
| Never work on main | Always create a worktree branch — never implement directly on `main` or `master` |
| Require commit before cleanup | Block teardown if no commit has been made in the session |
| Confirm before teardown | Ask the user before running `nekotree remove` |
| Warn on test failure | If build/tests fail, stop and report — do not commit, do not clean up |
| Safe mount paths only | Never mount system directories; worktree path is auto-managed by nekotree |
| No shell injection | Branch names and commands must be validated — reject special characters |
