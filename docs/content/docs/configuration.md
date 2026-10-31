---
title: Configuration
weight: 30
---

Settings can come from a YAML config file, environment variables and [command-line flags]({{< relref "usage#command-line-flags" >}}).

## Resolution order

Settings are resolved in this order (later values override earlier ones):

1. Config file (`config.yaml` or `config.yml` in the working directory, or the file given with `--config`)
2. Environment variables
3. Command-line flags

## Config file

Create a `config.yaml` (or `config.yml`) in your working directory. It is loaded automatically when no `--config` flag is provided.

```yaml
username: octocat
start: 2024-01-01
end: 2024-03-31
token: ghp_xxxxxx  # Optional; if not provided, username becomes required
output_dir: ./results
maintained_repos:
  - owner/repo1
  - owner/repo2
```

| Key | Flag equivalent | Description |
| --- | --------------- | ----------- |
| `username` | `--username` | GitHub username to query |
| `start` | `--start` | Start date (RFC3339 or YYYY-MM-DD) |
| `end` | `--end` | End date (RFC3339 or YYYY-MM-DD), defaults to now |
| `output_dir` | `--out` | Output directory for JSON files (default: `out`) |
| `maintained_repos` | `--maintained` | Repositories you maintain. An empty list (`[]`) disables maintainer tracking, leaving the key out enables auto-discovery |
| `token` | `--token` | GitHub personal access token |

{{< callout type="warning" >}}
Prefer the `GITHUB_TOKEN` environment variable to a `token` in the config file, so the token is never committed or shared along with the file.
{{< /callout >}}

## Environment variables

| Variable | Flag equivalent | Description |
| -------- | --------------- | ----------- |
| `GITHUB_TOKEN` | `--token` | GitHub personal access token |
| `GITHUB_USERNAME` | `--username` | GitHub username to query |

```bash
export GITHUB_TOKEN="ghp_xxxxxx"
export GITHUB_USERNAME="octocat"
github-insights --start 2024-01-01
```

When only a token is provided, the username defaults to the owner of the token.
