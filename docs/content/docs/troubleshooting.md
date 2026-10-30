---
title: Troubleshooting
weight: 60
---

## Missing GitHub token or username

If no token is provided, you must specify a username:

```plaintext
error: missing GitHub username: set --username, GITHUB_USERNAME env var, or config username (required when no token is provided)
```

Provide either a GitHub token or a username, through a flag, an environment variable or the config file (see [Configuration]({{< relref "configuration" >}})).

## Missing start date

```plaintext
error: missing start date: set --start or config start
```

The start of the date range has no default: pass `--start` or set `start` in the config file.

## Invalid date format

Dates must be in `RFC3339` or `YYYY-MM-DD` format:

- ✅ `2024-01-15`
- ✅ `2024-01-15T10:30:00Z`
- ❌ `01/15/2024`

## No maintained repos discovered

If you don't specify repositories and the tool finds none, it skips release/tag and maintainer insights:

```plaintext
no maintained repos found; release, tag, and maintainer-issue collection skipped
```

Explicitly provide repos with `--maintained`, or make sure your token has permission to list repositories. Discovery is also skipped when analyzing another user than the token owner (see [auto-discovery]({{< relref "usage#auto-discover-maintained-repositories" >}})).
