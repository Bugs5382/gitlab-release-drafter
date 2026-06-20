# 📁 Examples

Ready-to-adapt configs and CI pipelines for **gitlab-release-drafter**. See the
[top-level README](../README.md) for the full reference.

## 🧾 Configs (`configs/`)

Store the chosen config as the **`GLRD_CONFIG`** CI/CD variable (raw YAML, or
base64 via [`scripts/encode-config.sh`](scripts/encode-config.sh)). Do **not**
mask it.

| File | Use it when… |
|---|---|
| [`minimal.yml`](configs/minimal.yml) | You want the absolute smallest starting point and are happy with defaults. |
| [`scoped-labels.yml`](configs/scoped-labels.yml) | You use GitLab **scoped labels** (`type::feature`) and want emoji categories + a contributors section. |
| [`conventional-commits.yml`](configs/conventional-commits.yml) | You want labels derived **automatically** from `feat/`, `fix:`-style branches/titles. |
| [`full.yml`](configs/full.yml) | You want to see **every** option documented inline, then trim it down. |

## 🏗️ CI pipelines (`ci/`)

Copy a file's jobs into your `.gitlab-ci.yml` (or `include:` it). Each assumes
`GLRD_CONFIG` and `GLRD_TOKEN` are set as CI/CD variables.

| File | Demonstrates |
|---|---|
| [`01-basic.gitlab-ci.yml`](ci/01-basic.gitlab-ci.yml) | `preview` on every pipeline + `release` on schedule/tag/manual. |
| [`02-autolabel.gitlab-ci.yml`](ci/02-autolabel.gitlab-ci.yml) | Auto-applying labels to MRs on `merge_request_event` pipelines. |
| [`03-auto-tag.gitlab-ci.yml`](ci/03-auto-tag.gitlab-ci.yml) | Letting the tool create the version **tag** itself (`--auto-tag`). |
| [`04-version-in-build.gitlab-ci.yml`](ci/04-version-in-build.gitlab-ci.yml) | Consuming `$RELEASE_VERSION` from the dotenv artifact in a **build** job. |
| [`05-shared-template.gitlab-ci.yml`](ci/05-shared-template.gitlab-ci.yml) | A central template `include:`d across **many repos** with group-level vars. |

## 🔧 Scripts (`scripts/`)

| File | Purpose |
|---|---|
| [`encode-config.sh`](scripts/encode-config.sh) | Base64-encode a config file for the `GLRD_CONFIG` variable. |

## ▶️ Try a config locally

You can validate a config end-to-end without a real GitLab by pointing the tool
at a config file via an env var. For example:

```bash
# Build the binary
go build -o gitlab-release-drafter ./cmd/gitlab-release-drafter

# Load a config from a custom variable and run preview.
# (Without CI_* vars set, the API call will fail — this just confirms the
#  config parses and the CLI wires up.)
export MY_CFG="$(cat examples/configs/minimal.yml)"
./gitlab-release-drafter preview --config-var MY_CFG
```

In real pipelines the variable is named `GLRD_CONFIG` and the `CI_*` variables
are provided by GitLab automatically.
