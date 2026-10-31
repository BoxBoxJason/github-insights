---
title: Usage
description: "Run github-insights for a date range: every command-line flag, with examples, and where the per-repository JSON reports are written."
weight: 20
---

Collect activity for a date range:

```bash
github-insights --start 2024-01-01 --end 2024-03-31
```

The tool creates an `out/` directory with one JSON file per repository (see [Output]({{< relref "output" >}})).

## Command-line flags

| Flag | Description | Required |
| ---- | ----------- | -------- |
| `--start` | Start date (RFC3339 or YYYY-MM-DD) | Yes |
| `--end` | End date (RFC3339 or YYYY-MM-DD), defaults to now | No |
| `--username` | GitHub username to query | No* |
| `--token` | GitHub personal access token | No* |
| `--out` | Output directory for JSON files (default: `out`) | No |
| `--maintained` | Comma-separated list of repos you maintain, can be repeated. An empty value disables maintainer tracking | No |
| `--config` | Path to YAML config file (default: `config.yaml` or `config.yml` if present) | No |
| `--verbose` | Enable verbose (debug-level) logging | No |

\*Resolved from flag, then environment variables, then config file (see [Configuration]({{< relref "configuration" >}})). If no token is provided, the username becomes required.

A date-only `--end` covers the whole day (until 23:59:59 UTC).

{{< callout type="info" >}}
The generated [CLI reference]({{< relref "cli-reference" >}}) lists the flags exactly as the binary defines them.
{{< /callout >}}

## Examples

### Analyze your activity for Q1

```bash
github-insights \
  --start 2024-01-01 \
  --end 2024-03-31 \
  --out ./q1-report
```

### Track maintained repositories

If you maintain certain repositories, specify them to capture release and tag information:

```bash
github-insights \
  --start 2024-01-01 \
  --username myusername \
  --maintained owner/my-project,owner/my-library \
  --out ./maintainer-report
```

### Auto-discover maintained repositories

If you omit `--maintained` (and `maintained_repos` isn't set in the config file), and you're analyzing the token owner themselves, the tool automatically discovers repositories where that user has admin, maintain, or push permissions:

```bash
github-insights --start 2024-01-01
```

This discovery is always scoped to the authenticated token: the GitHub API has no way to list "repos user X can push to" for an arbitrary `X`. So if `--username`/`GITHUB_USERNAME` names someone other than the token owner, auto-discovery is skipped automatically (with a log warning) instead of silently substituting the token owner's repos into that user's report. Pass `--maintained` explicitly if you know which repos to track in that case:

```bash
GITHUB_USERNAME="someone-else" github-insights --start 2024-01-01 --maintained owner/repo,owner/other-repo
```

To explicitly opt out of maintainer/release/tag tracking even when analyzing yourself, pass an empty list rather than omitting the flag:

```bash
github-insights --start 2024-01-01 --maintained ""
```

or in the config file:

```yaml
maintained_repos: []
```
