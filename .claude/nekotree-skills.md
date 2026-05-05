# Nekotree Skills Reference

These skills allow Claude Code to interact with Nekotree's isolated development environments.

---

## Binary Location

The Nekotree binary is located at:
- **Default**: `build/nekotree`
- **Alternative paths**: `site/nekotree`, `./nekotree`, `~/bin/nekotree`

The binary is built during `./scripts/build.go build` and placed in `build/nekotree`.

---

## Skill: `build-dev-env`

Creates an isolated development environment for a specific branch using Nekotree's worktree capabilities.

### Description

When invoked, this skill:
1. Takes a branch name as input
2. Creates an isolated Git worktree at `build/worktrees/<branch-name>/`
3. Optionally spins up a Docker container with the specified base image
4. Returns the path to the worktree environment
5. Allows Claude to run commands within that isolated context

### Input Format

```yaml
branch: string   # Required - The branch name for the isolated environment
repo: string     # Optional - Path to the repository (defaults to current directory)
image: string    # Optional - Docker base image for container
ports: list      # Optional - Port mappings in "host:container" format
```

### Example Invocation

```yaml
branch: "feature-login"
repo: "./my-repo"
image: "node:18"
ports: ["8080:3000"]
```

### Output

Returns a context containing:
- `path`: The absolute path to the worktree environment
- `branch`: The sanitized branch name
- `cleanup`: A function to remove the environment when done

### Input Sanitization

Branch names are sanitized to prevent injection attacks:
- **Removed:** Backslashes, quotes, semicolons, pipes, ampersands, angle brackets, dollar signs, parentheses, curly braces, brackets
- **Limited:** Maximum 64 characters

---

## Skill: `build-list-environments`

Lists all created worktree environments and their status.

### Input Format

```yaml
filter: string   # Optional - Filter by branch name prefix
show-containers: bool  # Optional - Show container status
```

### Example Invocation

```yaml
filter: "feature"
show-containers: true
```

### Output

Returns a list of environments with:
- `name`: Branch name
- `path`: Worktree location
- `active`: Whether the environment is currently active
- `container`: Container status (running/stopped/none)

---

## Skill: `build-shell`

Runs a command within a specific worktree environment.

### Input Format

```yaml
branch: string   # Required - The branch name
command: string  # Required - Command to execute
container: bool  # Optional - Run inside container (default: false)
```

### Example Invocation

```yaml
branch: "feature-login"
command: "npm run build"
container: true
```

### Output

Returns:
- `stdout`: Command output
- `stderr`: Error output
- `exit_code`: Exit status
- `duration`: Time taken to execute

---

## Skill: `build-cleanup`

Removes an environment and its associated containers.

### Input Format

```yaml
branch: string   # Required - The branch name to remove
force: bool      # Optional - Force removal even if active
```

### Example Invocation

```yaml
branch: "feature-login"
force: true
```

### Output

Returns:
- `success`: Boolean indicating if cleanup succeeded
- `message`: Status message

---

## Skill: `build-ensure-binary`

Ensures the Nekotree binary is available before running commands.

### Description

This skill checks for and prepares the Nekotree binary:
1. Searches common binary locations
2. Builds the binary if not found
3. Validates binary integrity
4. Returns the binary path

### Binary Search Order

1. `build/nekotree` (default build location)
2. `site/nekotree`
3. `./nekotree` (current directory)
4. `~/bin/nekotree`
5. `/usr/local/bin/nekotree`
6. First match in `$PATH`

### Input Format

```yaml
force-build: bool  # Optional - Force rebuild even if binary exists
clean-cache: bool  # Optional - Clean build cache before building
```

### Example Invocation

```yaml
force-build: false
clean-cache: true
```

### Output

Returns:
- `path`: Absolute path to the binary
- `exists`: Boolean indicating if binary was found
- `built`: Boolean indicating if binary was built
- `error`: Error message if not found

### Output Messages

**Binary found:**
```
✓ Binary found: /path/to/nekotree
```

**Binary missing, building:**
```
✗ Binary not found. Building...
🔨 Building nekotree...
📏 Binary Size: 5821266 bytes (Expected: 5662310 - 6920601)
✅ Binary size integrity check passed.
✓ Binary ready: /path/to/build/nekotree
```

**Build failed:**
```
✗ Build failed: [error details]
```

---

## Quick Reference Table

| Skill | Purpose | Required Input |
|-------|---------|---------------|
| `build-dev-env` | Create isolated environment | `branch` |
| `build-list-environments` | List all environments | None |
| `build-shell` | Run command in environment | `branch`, `command` |
| `build-cleanup` | Remove environment | `branch` |
| `build-ensure-binary` | Ensure binary exists | None |

---

## Best Practices

1. **Always clean up** environments when done to free resources
2. **Use specific base images** for reproducible environments
3. **Filter by prefix** when listing environments to reduce output
4. **Run in containers** for true isolation when possible
5. **Sanitize inputs** before passing to any skill
6. **Use `build-ensure-binary`** before running other skills
7. **Check binary location** in build scripts

---

## Notes

- All environments are stored under `build/worktrees/`
- Container integration is optional
- Cleanup is required to prevent resource leaks
- Use `--force` flag for cleanup when environment is in unknown state
- Binary is built with `CGO_ENABLED=0` for static linking
- Binary size validation ensures consistent builds
