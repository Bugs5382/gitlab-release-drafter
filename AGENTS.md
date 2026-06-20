# AGENTS.md - gitlab-release-drafter

Guide for AI agents working in this repository. Pair with `CLAUDE.md` (the working agreement and
hook-enforced rules). Keep this file current when the build, layout, or public API changes.

## What this is

A GitLab-native release-note drafter, semantic-version resolver, and changelog maintainer, shipped
as a single static Go CLI and a distroless container image. It is driven entirely by GitLab CI: a
project adopts it by setting one CI/CD variable (`GLRD_CONFIG`) and adding a few jobs to
`.gitlab-ci.yml`. There is no config file committed to the target repository.

The binary has four subcommands: `preview`, `label`, `release`, and `changelog`. Pure logic lives in
focused `internal/*` packages; the GitLab API is hidden behind an interface (`internal/gitlab`) so
all logic is unit-testable without network access via the in-memory fake.

## Layout

- `cmd/gitlab-release-drafter` -- entrypoint and subcommand dispatch (Cobra).
- `internal/config` -- `GLRD_CONFIG` load (raw or base64 YAML), parse, defaults, validate.
- `internal/version` -- semver type, base detection, label->increment, bump, templating.
- `internal/model` -- shared domain types (MR, Milestone, Author, Release).
- `internal/categorize` -- assign MRs to categories; exclude; labels mode.
- `internal/render` -- notes templating (body/change/category) and contributors.
- `internal/changelog` -- Keep a Changelog read/modify/write.
- `internal/autolabel` -- files-glob / branch-regex / title-regex to label set.
- `internal/gitlab` -- `Client` interface, real HTTP impl (GraphQL + REST), and fake.
- `internal/output` -- stdout, dotenv artifact, notes artifact.
- `internal/app` -- command implementations wiring the pure packages and the client.

## Build and test

- `task build` / `go build ./...`
- `task test` / `go test ./...`
- `task lint` -- gofmt, golangci-lint, yamllint.
- `task license` -- verify MIT headers via golic (CI; never writes). `task license:fix` injects.

## Conventions

Conventional Commits, squash merges, human-authored voice (no AI tells, no emoji in source or
commit messages). See `CLAUDE.md` for the full working agreement.
