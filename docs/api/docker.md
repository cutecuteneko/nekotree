# 🐳 Docker Manager API

Handles all communication with the RHEL host Docker daemon to manage development environments.

## Internal Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `DOCKER_SOCKET` | `/var/run/docker.sock` | The Unix socket path for the host Docker engine. |
| `LABEL_MANAGED` | `nekotree.managed` | The label applied to all containers created by this tool. |

## API Reference

::: internal.docker
    options:
      show_source: true
      show_root_heading: false
