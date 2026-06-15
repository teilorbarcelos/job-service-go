#!/usr/bin/env bash
set -uo pipefail
PROJECT_KEY="${1:-job-service-go}"
PROJECT_NAME="${2:-job-service-go}"
SONAR_TOKEN="${SONAR_TOKEN:-squ_20f4835a20b16839fe7a52b4d43ff97224e640d1}"
SONAR_HOST="${SONAR_HOST:-http://localhost:9000}"
SCANNER_BIN="${SCANNER_BIN:-/home/teilor/.sonar/native-sonar-scanner/sonar-scanner-6.2.1.4610-linux-x64/bin/sonar-scanner}"
export LANG=C.UTF-8 LC_ALL=C.UTF-8
"$SCANNER_BIN" \
  -Dsonar.projectKey="$PROJECT_KEY" \
  -Dsonar.projectName="$PROJECT_NAME" \
  -Dsonar.sources=cmd,internal \
  -Dsonar.tests=. \
  -Dsonar.exclusions="**/*_test.go,**/vendor/**" \
  -Dsonar.go.coverage.reportPaths=coverage.out \
  -Dsonar.host.url="$SONAR_HOST" \
  -Dsonar.login="$SONAR_TOKEN"
