#!/usr/bin/env bash
# encode-config.sh — base64-encode a config file for the GLRD_CONFIG CI variable.
#
# gitlab-release-drafter accepts GLRD_CONFIG as raw YAML OR base64-encoded YAML
# (it auto-detects). Base64 is the lossless option when your config contains
# emojis or when a pipeline UI mangles multiline values.
#
# Usage:
#   ./encode-config.sh ../configs/scoped-labels.yml
#   ./encode-config.sh ../configs/scoped-labels.yml | pbcopy      # macOS clipboard
#
# Then paste the output as the value of the GLRD_CONFIG CI/CD variable
# (Settings > CI/CD > Variables). Leave "Mask variable" UNCHECKED.

set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <config.yml>" >&2
  exit 2
fi

file="$1"
if [ ! -f "$file" ]; then
  echo "error: no such file: $file" >&2
  exit 1
fi

# `base64 -w0` (GNU) keeps it on one line; fall back to stripping newlines for
# BSD/macOS base64 which has no -w flag.
if base64 -w0 </dev/null >/dev/null 2>&1; then
  base64 -w0 "$file"
else
  base64 "$file" | tr -d '\n'
fi
echo
