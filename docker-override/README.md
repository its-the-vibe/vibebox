# docker-override

Generate a Docker Compose override file that pins every service to the same image.

## Usage

```sh
docker-override create <img> [docker-compose.yml] [docker-compose.override.yml]
docker-override create --override-tag <tag> [docker-compose.yml] [docker-compose.override.yml]
docker-override view [--base | --override | --current]
docker-override delete [docker-compose.override.yml]
```

Examples:

```sh
docker-override create ghcr.io/its-the-vibe/app:latest
docker-override create ghcr.io/its-the-vibe/app:latest compose.yml compose.override.yml
docker-override create --override-tag feature
docker-override create --override-tag feature compose.yml compose.override.yml
docker-override view
docker-override view --base
docker-override delete
```
