# ⚙️ Configuration Reference

This package handles the parsing of environment variables and the internal state of the Nekotree manager.

## Environment Variables

| Variable | Type | Description |
| :--- | :--- | :--- |
| `DEVENV_MOUNTS` | `string` | Comma-separated list of host:container paths (e.g., `/data:/mnt/data`). |
| `NEKOTREE_HOST_PATH` | `string` | Translation mapping for Docker socket calls (e.g., `/workspace:/home/yunimoo/Gitea`). |
| `DEBUG` | `bool` | Enables verbose logging for worktree and container operations. |

## API Reference

::: internal.config
    options:
      show_source: true
      show_root_heading: false
