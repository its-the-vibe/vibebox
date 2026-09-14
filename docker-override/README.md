# docker-override

Generate a Docker Compose override file that pins every service to the same image.

## Usage

```sh
docker-override create <img> [docker-compose.yml] [docker-compose.override.yml]
```

Examples:

```sh
docker-override create ghcr.io/its-the-vibe/app:latest
docker-override create ghcr.io/its-the-vibe/app:latest compose.yml compose.override.yml
```
