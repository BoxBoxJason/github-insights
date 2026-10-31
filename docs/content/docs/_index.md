---
title: Documentation
description: "Overview of github-insights: track your GitHub pull requests, code reviews, issues, mentions and releases over a date range, exported as JSON."
weight: 1
---

github-insights collects comprehensive activity insights from GitHub. It tracks your contributions (pull requests, code reviews, issues, mentions, and releases) across a date range and outputs structured data for reporting, analysis, or documentation.

![Example visualization of collected insights in the Steam-themed viewer](../images/example.png)

## Features

- **Pull Request Tracking**: capture all PRs you authored and reviewed
- **Issue Analytics**: track issues created and commented on
- **Mention Detection**: find all mentions of your username across repositories
- **Release & Tag Monitoring**: track releases and tags in repositories you maintain
- **Flexible Configuration**: use a YAML config file, command-line flags, or environment variables
- **Structured Output**: JSON files organized by repository for easy analysis
- **Rate Limit Aware**: efficient GitHub API usage with automatic rate limit handling

## Where to go next

{{< cards >}}
  {{< card link="installation" title="Installation" subtitle="go install, release binaries, container image, or build from source." >}}
  {{< card link="usage" title="Usage" subtitle="Command-line flags and common examples." >}}
  {{< card link="configuration" title="Configuration" subtitle="Config file, environment variables, and how they combine." >}}
  {{< card link="output" title="Output" subtitle="The report files and what they contain." >}}
  {{< card link="viewer" title="Viewer" subtitle="Browse the reports in a static web page." >}}
  {{< card link="troubleshooting" title="Troubleshooting" subtitle="Common errors and how to fix them." >}}
{{< /cards >}}
