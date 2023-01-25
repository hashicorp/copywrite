# copywrite

This repo provides utilities for managing copyright headers and license files
across many repos at scale.

You can use it to add or validate copyright headers on source code files, add a
LICENSE file to a repo, report on what licenses repos are using, and more.

## Getting Started

The easiest way to get started is to use Homebrew:

```sh
brew tap hashicorp/tap
brew install hashicorp/tap/copywrite
```

Installers for Windows, Linux, and MacOS are also available on the [releases](https://github.com/hashicorp/copywrite/releases) page.

## CLI Usage

This Go app is consumable as a command-line tool. Currently, the following subcommands are available:

```none
❯ copywrite
Copywrite provides utilities for managing copyright headers and license
files in HashiCorp repos.

You can use it to report on what licenses repos are using, add LICENSE files,
and add or validate the presence of copyright headers on source code files.

Usage:
  copywrite [command]

Common Commands:
  headers     Adds missing copyright headers to all source code files
  init        Generates a .copywrite.hcl config for a new project
  license     Validates that a LICENSE file is present and remediates any issues if found

Additional Commands:
  completion  Generate the autocompletion script for the specified shell
  debug       Prints env-specific debug information about copywrite
  dispatch    Dispatches audit jobs for a list of repos
  help        Help about any command
  report      Performs a variety of reporting tasks

Flags:
      --config string   config file (default is .copywrite.hcl in current directory)
  -h, --help            help for copywrite
  -v, --version         version for copywrite

Use "copywrite [command] --help" for more information about a command.
```

To get started with Copywrite on a new project, run `copywrite init`, which will
interactively help generate a `.copywrite.hcl` config file to add to Git.

The most common command you will use is `copywrite headers`, which will automatically
scan all files in your repo and copyright headers to any that are missing:

```sh
copywrite headers --spdx "MPL-2.0"
```

You may omit the `--spdx` flag if you add a `.copywrite.hcl` config, as outlined
[here](#config-structure).

### `--plan` Flag

Both the `headers` and `license` commands allow you to use a `--plan` flag, which
performs a dry-run and will outline what changes would be made. This flag also
returns a non-zero exit code if any changes are needed. As such, it can be used
to validate if a repo is in compliance or not.

## Config Structure

> :bulb: You can automatically generate a new `.copywrite.hcl` config with the
`copywrite init` command.

A `.copywrite.hcl` file can be referenced to provide configuration information
for a given project. This file should be specific to each repo and checked into
git. If no configuration file is present, default values will be used throughout
the `copywrite` application. An example config structure is shown below:

```hcl
# (OPTIONAL) Overrides the copywrite config schema version
# Default: 1
schema_version = 1

project {
  # (OPTIONAL) SPDX-compatible license identifier
  # Leave blank if you don't wish to license the project
  # Default: "MPL-2.0"
  license = "MPL-2.0"

  # (OPTIONAL) Represents the year that the project initially began
  # Default: <the year the repo was first created>
  # copyright_year = 0

  # (OPTIONAL) A list of globs that should not have copyright or license headers .
  # Supports doublestar glob patterns for more flexibility in defining which
  # files or folders should be ignored
  # Default: []
  header_ignore = [
    # "vendors/**",
    # "**autogen**",
  ]

  # (OPTIONAL) Links to an upstream repo for determining repo relationships
  # This is for special cases and should not normally be set.
  # Default: ""
  # upstream = "hashicorp/<REPONAME>"
}

```

## GitHub Authentication

Some commands interact directly with GitHub's API (especially when a
`.copywrite.hcl` config is not present for the project). In order to use these
commands successfully, multiple mechanisms are available to provide GitHub
credentials and are prioritized in the following order:

- GitHub App credentials can be supplied via the `APP_ID`, `INSTALLATION_ID`, and `APP_PEM` environment variables or a `.env` file.
- A `GITHUB_TOKEN` environment variable can be used with a Personal Access Token
- If you use the [GitHub CLI](https://cli.github.com/), auth information can automatically be used

If none of the above methods work, `copywrite` will default to using an **unauthenticated** client.

GitHub credentials are purposely excluded from the `.copywrite.hcl` config, as
that file is meant to be specific to each project and checked in to its repo.

## GitHub Action

To make it easier to use `copywrite` in your own CI jobs (e.g., to add a PR check),
you can make use of the `hashicorp/setup-copywrite` GitHub Action. It
automatically installs the binary and adds it to your `$PATH` so you can call it
freely in later steps.

```yaml
  - name: Setup Copywrite
    uses: hashicorp/setup-copywrite@main

  - name: Check Header Compliance
    run: copywrite headers --plan
```

:bulb: Running the copywrite command with the `--plan` flag will return a non-zero exit code if the repo is out of compliance.

## Development

### IDE Settings

To maintain a consistent developer experience, this repo comes bundled with VS Code settings. When opening the repo for the first time, you will be asked if you want to install [suggested extensions](./.vscode/extensions.json) and your workspace will be pre-configured with consistent format-on-save [settings](./.vscode/settings.json).

### Pre-Commit Hooks

Before committing code, this repo has been setup to check Go files using [pre-commit git hooks](https://pre-commit.com/). To leverage pre-commit, developers must install pre-commit and associated tools locally:

```bash
brew install pre-commit golangci-lint go-critic
```

Verify install went successfully with:

```bash
pre-commit --version
```

Once you verify `pre-commit` is installed locally, you can use pre-commit git hooks by installing them in this repo:

```bash
pre-commit install
```
