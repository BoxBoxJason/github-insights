---
title: Viewer
weight: 50
---

A static, Steam-themed web viewer lives in the [`frontend/`](https://github.com/boxboxjason/github-insights/tree/main/frontend) directory of the repository. It presents your collected reports as a "library" of repositories (game-capsule cards) with a per-repository detail view for PRs, reviews, issues, mentions, releases, and tags. No backend or build step is required: it runs from a single HTML file.

![Example visualization of collected insights in the Steam-themed viewer](../../images/example.png)

Because browsers cannot list a directory, a small script bundles your JSON reports into `frontend/data.js` first:

```bash
make frontend-data        # bundles results/*.json -> frontend/data.js
xdg-open frontend/index.html
```

The viewer reads from the `results/` directory by default. Re-run `make frontend-data` whenever you collect new reports, since the bundle is a snapshot.

{{< callout type="info" >}}
If your reports are in a different output directory, collect them with `--out results`, or copy them into `results/` before bundling.
{{< /callout >}}

{{< callout type="warning" >}}
The generated `frontend/data.js` embeds the same private data as `results/`. Both are gitignored: do not publish the viewer with your data in it.
{{< /callout >}}
