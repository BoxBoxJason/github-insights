---
title: Installation
description: "Install github-insights with go install, a release binary, the container image or from source, and create the GitHub token it uses."
weight: 10
---

## Install with go install

```bash
go install github.com/boxboxjason/github-insights@latest
```

To install a specific version, use a semantic version tag instead of `latest`.
Building requires the Go version declared in [`go.mod`](https://github.com/boxboxjason/github-insights/blob/main/go.mod) or later.

## Release binaries

Every [release](https://github.com/boxboxjason/github-insights/releases) ships prebuilt binaries for Linux, macOS and Windows, on `amd64` and `arm64`, named `github-insights_<version>_<os>_<arch>`.

## Container image

The CLI is also published to GitHub Container Registry, as `ghcr.io/boxboxjason/github-insights`, tagged `latest`, `<major>`, `<major>.<minor>` and `<version>`.
The image runs as a non-root user, so mount a directory it can write to for the reports, and point `--out` at it:

```bash
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -e GITHUB_TOKEN \
  -v "$PWD/out:/out" \
  ghcr.io/boxboxjason/github-insights:latest \
  --start 2024-01-01 --out /out
```

## Build from source

```bash
git clone https://github.com/boxboxjason/github-insights.git
cd github-insights
make build
```

The binary is written to `bin/github-insights`.

## GitHub token

A token is optional, but without one GitHub applies much lower rate limits and you must pass a username. For full functionality, the token should have access to:

- `public_repo`: access to public repositories
- `read:user`: read user profile information
- `repo` (optional): for private repositories

See [Configuration]({{< relref "configuration" >}}) for the ways to provide it.
