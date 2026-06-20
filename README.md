# 🦊 gitlab-release-drafter

> GitLab-native release notes, semantic versioning, and changelog automation — driven entirely by GitLab CI, with **no config file committed to your repository**.

Inspired by [release-drafter/release-drafter](https://github.com/release-drafter/release-drafter), but **not** a port. This tool speaks GitLab: **Merge Requests**, **scoped labels**, **milestones**, and **GitLab Releases**.

---

## ✨ What it does

- 📝 **Drafts release notes** from a milestone's merged MRs, grouped into categories by label.
- 🔢 **Resolves the next semver** from MR labels (`type::feature` → minor bump, `type::breaking` → major, …).
- 🏷️ **Autolabels MRs** from branch names, MR titles, and changed files.
- 👥 **Builds a contributors list** from MR authors (with an exclude list for bots).
- 📜 **Maintains `CHANGELOG.md`** in [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) format.
- 🚀 **Creates GitLab Releases** when a milestone is done.
- 🔮 **Previews the next version** on any branch and emits it as a `dotenv` artifact so your build jobs can align image tags before the release is even cut.

All configuration lives in **one CI/CD variable** (`GLRD_CONFIG`). Adopting the tool means setting that variable (group-level works for many repos) and adding a few jobs to `.gitlab-ci.yml`. No bootstrapping, no committed YAML, no scaffolding.

---

## 📦 Installation

### As a CI job image (recommended)

```yaml
preview:
  image: ghcr.io/bugs5382/gitlab-release-drafter:latest
  script:
    - gitlab-release-drafter preview
```

### From source

```bash
go install github.com/Bugs5382/gitlab-release-drafter/cmd/gitlab-release-drafter@latest
```

### Build locally

```bash
go build -o gitlab-release-drafter ./cmd/gitlab-release-drafter
```

---

## 🚀 Quick start

1. **Create a token** with `api` scope (Project or Group Access Token, or a PAT) and store it as a **masked** CI/CD variable named `GLRD_TOKEN`.
2. **Write a config** (see [Configuration](#%EF%B8%8F-configuration)) and store it as a CI/CD variable named `GLRD_CONFIG` (raw YAML or base64). Do **not** mask `GLRD_CONFIG` — masking breaks emojis and long YAML.
3. **Label your MRs** (e.g. `type::feature`, `type::fix`) and assign them to a **milestone**.
4. **Add jobs** to `.gitlab-ci.yml` — start from [`examples/ci/01-basic.gitlab-ci.yml`](examples/ci/01-basic.gitlab-ci.yml).

```yaml
stages: [preview, release]

preview:
  stage: preview
  image: $GLRD_IMAGE
  script: [ "gitlab-release-drafter preview" ]
  artifacts:
    reports:
      dotenv: release.env
    paths: [ release-notes.md ]
```

---

## 🧠 Core model

| Concept | How the tool uses it |
|---|---|
| **Milestone** | The release unit. Its **merged MRs** are the inputs to a release. |
| **MR labels** | Drive both **categorization** (which section of the notes) and the **version bump**. |
| **GitLab Release** | The output, created when a milestone is done. |
| **`GLRD_CONFIG`** | The only configuration source. Never a file in the repo. |
| **`GLRD_TOKEN`** | Auth for reading MRs and writing labels/releases (`CI_JOB_TOKEN` can't read the MR API). |

---

## 🧩 Commands

The binary is a single CLI with four subcommands, each wired as its own CI job so they compose independently.

| Command | Typical trigger | Writes? | Behavior |
|---|---|---|---|
| `preview` | any branch | ❌ none | Resolve version + render notes for the relevant milestone. Pure read. |
| `label` | MR pipelines | ✏️ MR labels | Apply autolabeler rules to the pipeline's MR. |
| `release` | milestone done / tag push | 🚀 tag (opt), Release, CHANGELOG | Resolve, optionally tag, create/upsert the Release, update `CHANGELOG.md` (working tree only). |
| `changelog` | optional standalone | 📜 CHANGELOG file | Update the changelog without touching releases/tags. |

```bash
gitlab-release-drafter --help
gitlab-release-drafter release --help
```

### Global flags

| Flag | Env default | Description |
|---|---|---|
| `--milestone` | `GLRD_MILESTONE` | Force a specific milestone (title or IID). |
| `--out-dir` | `.` | Directory for artifacts and `CHANGELOG.md`. |
| `--config-var` | `GLRD_CONFIG` | Name of the env var holding the YAML config. |

`release` also accepts `--auto-tag` (env: `GLRD_AUTO_TAG`) to create the resolved tag itself.

---

## 📤 Outputs & artifacts

| File / output | Produced by | Contents |
|---|---|---|
| `release.env` | `preview`, `release` | dotenv: `RELEASE_VERSION`, `RELEASE_TAG`, `RELEASE_MILESTONE`, `RELEASE_INCREMENT` |
| `release-notes.md` | `preview`, `release` | The rendered release body. |
| `CHANGELOG.md` | `release`, `changelog` | Keep a Changelog file, **written to the working tree** (committing is your pipeline's job). |
| GitLab Release | `release` | Created/updated via the Releases API. |
| MR labels | `label` | Applied via the MRs API. |

> 💡 The `dotenv` report means downstream jobs can read `$RELEASE_VERSION` directly — perfect for tagging container images before the release is cut. See [`examples/ci/04-version-in-build.gitlab-ci.yml`](examples/ci/04-version-in-build.gitlab-ci.yml).

---

## ⚙️ Configuration

Config comes from the `GLRD_CONFIG` variable as **YAML**, optionally **base64-encoded** (auto-detected). Encode with [`examples/scripts/encode-config.sh`](examples/scripts/encode-config.sh).

### Minimal

```yaml
categories:
  - title: "Changes"
    labels: ["feature", "fix", "chore"]
```

### Full schema

```yaml
version:
  tag-template: "v{version}"          # tag name        (default)
  name-template: "v{version}"         # Release title   (default)
  initial: "0.1.0"                    # used when no prior release exists
  default-increment: patch            # for MRs matching no increment label
  increments:                         # label -> semver bump
    major: ["type::breaking", "breaking-change"]
    minor: ["type::feature", "enhancement"]
    patch: ["type::fix", "bug", "type::chore"]

categories:                           # render order = list order
  - title: "⚠️ Breaking Changes"      # list first so it leads the notes;
    labels: ["type::breaking", "breaking-change"]   # mirror version.increments.major
  - title: "🚀 Features"
    labels: ["type::feature", "enhancement"]
  - title: "🐛 Bug Fixes"
    labels: ["type::fix", "bug"]
  - title: "🧰 Maintenance"
    labels: ["type::chore", "dependencies"]
uncategorized-title: ""               # e.g. "Other Changes"; empty = drop unmatched MRs

labels-mode: any                      # any | all  (how an MR matches a category)
exclude-labels: ["skip-changelog", "type::ci"]   # MRs with these are excluded entirely

template: |                           # Release body
  {changes}

  ## Contributors
  {contributors}
change-template: "- {title} (!{iid}) by @{author}"
no-changes-template: "No notable changes."
category-template: "### {title}"
sort: merged_at                       # merged_at | title
sort-direction: desc                  # asc | desc

contributors:
  exclude: ["renovate-bot", "dependabot"]
  template: "@{username}"
  separator: ", "

autolabeler:                          # applied by the `label` command
  - label: "type::fix"
    branch: ["^fix/", "^bugfix/"]
    title: ['^fix(\(.+\))?:']
  - label: "type::feature"
    branch: ["^feat/", "^feature/"]
    title: ['^feat(\(.+\))?:']
  - label: "documentation"
    files: ["docs/**", "**/*.md"]

changelog:
  file: "CHANGELOG.md"
  keep-a-changelog: true
  section-map:                        # map your categories -> canonical KAC sections
    "⚠️ Breaking Changes": "Changed"
    "🚀 Features": "Added"
    "🐛 Bug Fixes": "Fixed"
    "🧰 Maintenance": "Changed"
```

### 🔤 Template placeholders

| Scope | Placeholders |
|---|---|
| **Body** (`template`) | `{changes}`, `{contributors}`, `{version}`, `{previous_tag}`, `{date}`, `{milestone}` |
| **Change** (`change-template`) | `{title}`, `{iid}`, `{author}`, `{url}`, `{labels}`, `{milestone}` |
| **Contributors** (`contributors.template`) | `{username}`, `{name}` |
| **Tag / Name** (`tag-template`, `name-template`) | `{version}`, `{major}`, `{minor}`, `{patch}` |

---

## 🔢 Versioning

```
next version = (highest existing tag matching tag-template, else `initial`)
               bumped by
               (highest increment implied across the milestone's MR labels)
```

- An MR whose labels match no `increments` entry contributes `default-increment`.
- The **highest** increment across all MRs wins (one `type::breaking` MR ⇒ major bump).
- On a feature branch, `preview` reports the version the branch's milestone *would* cut, emitted as `$RELEASE_VERSION`.

---

## 🎯 Milestone selection

| Situation | Milestone chosen |
|---|---|
| `--milestone` / `GLRD_MILESTONE` set | That milestone (title or IID). |
| `preview` on a feature branch | The milestone of the branch's open MR (no MR milestone ⇒ "no milestone", exits 0). |
| `preview` / default branch / scheduled | The project's nearest-due open milestone. |
| `release`, scheduled | All **closed-but-unreleased** milestones (closed, with no Release yet). |

---

## ⏱️ Triggers — how a milestone becomes a Release

GitLab can't fire a pipeline on "milestone closed", so `release` is driven by:

1. **Scheduled pipeline** (primary) 🔁 — scans for closed-but-unreleased milestones and releases each.
2. **Manual job** 👆 — release on demand (`when: manual`).
3. **Tag pipeline** 🏷️ — react to a pushed version tag (`if: $CI_COMMIT_TAG`).

`--auto-tag` lets `release` create the tag itself instead of waiting for one.

---

## 🔐 Auth & permissions

| Variable | Scope | Used for | Masked? |
|---|---|---|---|
| `GLRD_TOKEN` | `api` | Read MRs (GraphQL/REST), write MR labels, create tags/releases | ✅ yes (and protected if releasing only from protected refs) |
| `GLRD_CONFIG` | — | The YAML config | ❌ **no** (masking breaks emojis/long YAML) |

`CI_JOB_TOKEN` can create Releases/tags but **cannot** read MRs or write labels, so a single `GLRD_TOKEN` is used everywhere for simplicity.

---

## 📁 Examples

See the [`examples/`](examples/) folder:

| Path | Shows |
|---|---|
| [`configs/minimal.yml`](examples/configs/minimal.yml) | Smallest viable config. |
| [`configs/scoped-labels.yml`](examples/configs/scoped-labels.yml) | GitLab scoped labels + emoji categories. |
| [`configs/conventional-commits.yml`](examples/configs/conventional-commits.yml) | Autolabeler tuned for conventional-commit branches/titles. |
| [`configs/full.yml`](examples/configs/full.yml) | Every option, documented inline. |
| [`ci/01-basic.gitlab-ci.yml`](examples/ci/01-basic.gitlab-ci.yml) | `preview` + `release`. |
| [`ci/02-autolabel.gitlab-ci.yml`](examples/ci/02-autolabel.gitlab-ci.yml) | MR autolabeling. |
| [`ci/03-auto-tag.gitlab-ci.yml`](examples/ci/03-auto-tag.gitlab-ci.yml) | Tool creates the tag itself. |
| [`ci/04-version-in-build.gitlab-ci.yml`](examples/ci/04-version-in-build.gitlab-ci.yml) | Consume `$RELEASE_VERSION` in a build job. |
| [`ci/05-shared-template.gitlab-ci.yml`](examples/ci/05-shared-template.gitlab-ci.yml) | `include:` a shared template across many repos. |
| [`scripts/encode-config.sh`](examples/scripts/encode-config.sh) | Base64-encode a config for the CI variable. |

---

## 🛠️ Development

```bash
go test ./...     # run the unit suite
go vet ./...      # static checks
go build ./...    # compile everything
```

The codebase is split into small, independently testable packages under `internal/`:

| Package | Responsibility |
|---|---|
| `config` | Load `GLRD_CONFIG` (raw/base64), defaults, validation. |
| `version` | Semver, base detection, label→increment, bump, templating. |
| `categorize` | Assign MRs to categories; exclude rules; labels-mode. |
| `render` | Notes templating + contributors. |
| `changelog` | Keep a Changelog 1.1.0 read/modify/write. |
| `autolabel` | branch / title / file-glob rule matching. |
| `gitlab` | API client (`Client` interface + REST impl + test fake). |
| `output` | stdout, dotenv artifact, file writers. |
| `app` | Command implementations wiring everything together. |

> ℹ️ The GitLab client is implemented against the **REST** API v4. The `Client` interface keeps all business logic testable without a network.

---

## 📄 License

[MIT](LICENSE) © Bugs5382
