# Copywrite — Copilot Agent Instructions

Trust these instructions first. Only fall back to searching the codebase if the
information here is incomplete or proven incorrect by an error you actually hit.

## What this repository is

`copywrite` is a HashiCorp/IBM command-line tool (single Go module/binary) that
automates copyright headers and license files across repositories at scale. It
can add/validate copyright headers on source files, manage `LICENSE` files with
git-aware copyright-year detection, report on licenses across many repos, and
run compliance checks in CI (via the `--plan` dry-run flag, which exits non-zero
when changes are needed).

- **Type:** CLI application built with [Cobra](https://github.com/spf13/cobra).
- **Language/runtime:** Go (**module declares `go 1.25.0`; `.go-version` pins
  `1.25.0`**). ~99% Go. Always use a Go toolchain that satisfies `go.mod`.
- **Config & parsing:** uses `koanf` with HCL/YAML/dotenv/env providers. Project
  config lives in a per-repo `.copywrite.hcl` file (see `.copywrite.hcl` and the
  README "Config Structure" section).
- **Size:** small-to-medium repo, a handful of top-level packages.

## Build, test, lint, run (validated against CI)

There is **no Makefile and no bootstrap script**. Use standard Go commands from
the repo root. **Always run `go mod download` (or let the first `go build` fetch
modules) before building offline.**

Run these in this order; this mirrors the `golangci.yml` CI workflow:

1. **Build the binary:** `go build -o copywrite .`
   (GoReleaser uses `main: main.go` with `CGO_ENABLED=0`; CGO is not required.)
2. **Run all tests:** `go test ./...`
   - CI uses: `go test -v -coverprofile=coverage.out ./...`
   - **Important:** some tests (and the tool itself) depend on **git history**.
     The CI test job checks out with `fetch-depth: 0`. When working in a shallow
     clone, copyright-year auto-detection or git-dependent tests may behave
     differently. Ensure full git history is available before running tests that
     touch year detection.
   - Tests in `licensecheck/` and `addlicense/` mutate the filesystem; they use
     `t.TempDir()` and `testdata/` fixtures. Do not edit files under any
     `testdata/` directory unless that is the intent of your change.
3. **Lint:** CI runs **`golangci-lint`** (action `golangci-lint-action`,
   `version: latest`). There is **no committed `.golangci.yml`**, so the linter
   runs with its default rule set. Locally: `golangci-lint run ./...`
   (install with `brew install golangci-lint` or `go install
   github.com/golangci/golangci-lint/cmd/golangci-lint@latest`).
4. **Tidy modules:** CI has a dedicated `go mod tidy` job that fails if `go.mod`/
   `go.sum` change. **Always run `go mod tidy` and commit the result whenever you
   add, remove, or change a dependency.**
5. **Run formatting/vet (pre-commit parity):** `gofmt -l .` (should print
   nothing) and `go vet ./...`.
6. **Run the CLI:** `go run . --help`, or after building, `./copywrite --help`.
   Common subcommands: `headers`, `license`, `init`, `report`, `dispatch`,
   `debug`. Add `--plan` to `headers`/`license` for a non-mutating dry run.

### Pre-commit hooks
`.pre-commit-config.yaml` defines the local hook suite (gofmt, go-vet,
go-imports, go-cyclo `-over=20`, golangci-lint, go-unit-tests, go-build,
go-mod-tidy, header/whitespace fixers). To replicate: `brew install pre-commit
golangci-lint go-critic`, then `pre-commit install`, then `pre-commit run
--all-files`. The `go-critic` hook is intentionally disabled in config.

### GoReleaser (build/release only — not normally needed for code changes)
`go build` is enough for development. The release/build CI job runs
`goreleaser release --clean --skip=publish --snapshot`. Only invoke GoReleaser
if your change affects `.goreleaser.yaml`.

### Things that commonly cause CI rejection — avoid them
- **Missing or out-of-date copyright headers.** Every Go source file begins with
  `// Copyright IBM Corp. <years>` and `// SPDX-License-Identifier: MPL-2.0`.
  **Always add this two-line header to new `.go` files** (the project enforces
  its own headers). Files under `addlicense/` are a forked Google package and
  keep the Apache header instead.
- **Untidy modules** → run `go mod tidy`.
- **Lint/format failures** → run `gofmt -w .`, `go vet ./...`,
  `golangci-lint run ./...` before pushing.
- **Cyclomatic complexity over 20** (go-cyclo hook) on functions you add.

## Project layout

Repo root files: `main.go` (entrypoint → calls `cmd.Execute()`), `go.mod`,
`go.sum`, `.go-version`, `.goreleaser.yaml`, `.pre-commit-config.yaml`,
`.pre-commit-hooks.yaml`, `.copywrite.hcl`, `README.md`, `CHANGELOG.md`,
`LICENSE`, `.gitignore`, `.vscode/`, `META.d/`. (No `.golangci.yml` is
committed.) Under `.github/`: `CODEOWNERS`, `dependabot.yml`,
`pull_request_template.md`, `workflows/`.

Top-level Go packages (where to make changes):
- **`cmd/`** — all Cobra commands/CLI surface. Key files: `root.go` (root
  command, global flags, version), `headers.go`, `license.go`, `init.go`,
  `report.go`, `report_prs.go`, `report_repos.go`, `dispatch.go`, `debug.go`,
  `utils.go` (+ `utils_test.go`). **Add or modify CLI behavior here.**
- **`config/`** — `.copywrite.hcl` schema loading/merging via koanf
  (`config.go`, `config_test.go`, `testdata/`). Change config schema here.
- **`addlicense/`** — internal fork of `google/addlicense`; core header
  add/validate/update logic (`main.go` holds `Run`, `addLicense`,
  `updateLicenseHolder`, `licenseHeader` file-extension→comment-style map).
  Apache-licensed; "not supported" upstream-style fork.
- **`licensecheck/`** — `LICENSE` file detection/remediation (`Entry`).
- **`github/`** — GitHub API client/auth helpers (App creds, `GITHUB_TOKEN`,
  `gh` CLI fallback; see README "GitHub Authentication").
- **`dispatch/`**, **`repodata/`** — audit-job dispatch and repo data helpers.

### CI / checks that gate merges
GitHub Actions workflows in `.github/workflows/`:
- **`golangci.yml`** (on push to `main` and all PRs): jobs `lint`
  (golangci-lint), `test` (`go test -v -coverprofile=coverage.out ./...` with
  `fetch-depth: 0`), `tidy` (`go mod tidy`), `build` (GoReleaser snapshot).
  **Replicate locally with the build/test/lint/tidy commands above before
  finishing.**
- **`build-and-release.yml`** (on `v*` tags only): GoReleaser release + Homebrew
  publish. Not triggered by normal PRs.

### Environment / runtime notes
- Some commands (`report`, `dispatch`, header logic without a `.copywrite.hcl`)
  call the GitHub API and/or rely on git history. For local runs set
  `GITHUB_TOKEN`, or rely on the `gh` CLI; otherwise an unauthenticated client
  is used and may be rate-limited.
- Logging verbosity is controlled by `COPYWRITE_LOG_LEVEL`
  (`trace|debug|info|warn|error|off`); `RUNNER_DEBUG=1` enables debug logging.
- Use `copywrite debug` to inspect loaded config and auth state.

If anything above fails or is missing, then (and only then) search the relevant
package directory listed under "Project layout".
