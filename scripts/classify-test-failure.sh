#!/usr/bin/env bash
# Classifies go test -json output as: infra | flake | regression
# Usage: go test -tags integration -json ./... | classify-test-failure.sh
set -euo pipefail
input=$(cat)
if echo "$input" | grep -q '"Action":"fail"'; then
  if echo "$input" | grep -qE '"Output":".*(connection refused|cannot connect|dial tcp|context deadline exceeded|container)'; then
    echo "INFRA"
    exit 0
  fi
  if echo "$input" | grep -qE '"Output":".*(flaky|timing|race detected)'; then
    echo "FLAKE"
    exit 1
  fi
  echo "REGRESSION"
  exit 2
fi
echo "PASS"
exit 0
