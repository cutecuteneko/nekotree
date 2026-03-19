# 🌿 Git Worktree API

Provides the logic for managing physical Git worktrees on the host filesystem.

## Constants & Variables

| Constant | Value | Description |
| :--- | :--- | :--- |
| `WORKTREE_PREFIX` | `nekotree-` | The prefix used for all directory names created by the tool. |
| `GIT_BIN` | `git` | The path to the git executable (must be in system PATH). |

## API Reference

::: internal.gitworktree
    options:
      show_source: true
      show_root_heading: false
