---
title: Output
weight: 35
---

The tool writes one JSON file per repository with activity in the date range, in the output directory (`out` by default). Files are named after the repository, with the slash replaced by an underscore:

```plaintext
out/
  octocat_hello-world.json
  my-org_my-library.json
```

Repositories without any tracked activity get no file. Running the tool again with the same output directory overwrites the files of the repositories it reports on.

Each file holds everything collected for that repository:

| Section | Content |
| ------- | ------- |
| `prs_authored` | Pull requests you authored, with their reviews, comments and size |
| `prs_reviewed` | Pull requests you reviewed, with your reviews and all the others |
| `issues_created` | Issues you created |
| `issues_commented` | Issues you commented on |
| `mentions` | Issues, pull requests and comments mentioning your username |
| `maintainer_new_issues` | New issues opened in repositories you maintain |
| `releases` | Releases published in repositories you maintain |
| `tags` | Tags created in repositories you maintain |

Every field of the files is described in the generated [output schema]({{< relref "output-schema" >}}).

The files can be browsed with the [viewer]({{< relref "viewer" >}}).
