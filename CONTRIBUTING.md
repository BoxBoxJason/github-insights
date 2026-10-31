# Contributing to github-insights

Contributions are welcome: open an issue to discuss a change, or submit a pull request.

## Development Workflow

1. Install [Go](https://go.dev/doc/install) (see [go.mod](./go.mod) for the required version) and [Make](https://www.gnu.org/software/make/).
2. Create a branch for your changes.
3. Run the tests: `make test`
4. Run the linter: `make lint`
5. Submit a pull request with a clear description of your changes and why they are needed.

## Documentation

The documentation lives in the [`docs`](./docs) directory, as Markdown rendered by [Hugo](https://gohugo.io/) (extended edition) with the shared [hugo-theme](https://github.com/boxboxjason/hugo-theme), and is published to <https://boxboxjason.github.io/github-insights/> from `main`.

- Change the documentation in the same pull request as the code it describes.
- `make docs-serve` serves the site locally with live reload, `make docs` builds it into `docs/public`.
- The [CLI reference](./docs/content/docs/cli-reference.md) and [output schema](./docs/content/docs/output-schema.md) pages are generated from the code: run `make docs-generate` after changing a flag or an output type, and commit the result.
- `make test` fails when a flag is not documented in the [usage](./docs/content/docs/usage.md) page, when an environment variable or a config file key is not documented in the [configuration](./docs/content/docs/configuration.md) page, or when a generated page is stale.
- Look and feel (colors, footer, ...) belong to the shared theme, do not restyle a single site.
