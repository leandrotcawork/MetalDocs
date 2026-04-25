#!/usr/bin/env bash
set -euo pipefail
go test ./internal/modules/iam/...   -tags=integration   -bench=BenchmarkAuthzCheck   -benchtime=1000x   -count=3   -benchmem
