#!/usr/bin/env bash
#
# code_quality.sh - Run all code quality checks for cccc
#
# This script runs the same checks that are executed in the GitHub Actions CI workflow.
# Run this before committing to catch issues early.

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}Running Code Quality Checks (cccc)${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

###########################################
# 1. Code Formatting Check (gofmt)
###########################################
echo -e "${BLUE}[1/7] Checking code formatting with gofmt...${NC}"
UNFORMATTED=$(gofmt -s -l . 2>&1)
if [ -n "$UNFORMATTED" ]; then
    echo -e "${RED}✗ The following files are not formatted:${NC}"
    echo "$UNFORMATTED"
    echo -e "${YELLOW}Run: gofmt -s -w .${NC}"
    exit 1
else
    echo -e "${GREEN}✓ All files are properly formatted${NC}"
fi
echo ""


###########################################
# 2. Build Check
###########################################
echo -e "${BLUE}[2/7] Building binary...${NC}"
if go build -v -o cccc ./cmd/ccc 2>&1 > /dev/null; then
    echo -e "${GREEN}✓ Build successful${NC}"

    # Verify binary works
    if ./cccc --help > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Binary verification successful${NC}"
    else
        echo -e "${YELLOW}⚠ Binary runs but --help returned non-zero (expected during development)${NC}"
    fi

    # Clean up
    rm -f cccc
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi
echo ""


###########################################
# 3. Static Analysis (go vet)
###########################################
echo -e "${BLUE}[3/7] Running static analysis with go vet...${NC}"
if go vet ./... 2>&1; then
    echo -e "${GREEN}✓ go vet passed${NC}"
else
    echo -e "${RED}✗ go vet found issues${NC}"
    exit 1
fi
echo ""


###########################################
# 4. Tests with Race Detection and SKIP detection
###########################################
echo -e "${BLUE}[4/7] Running tests with race and skip detection...${NC}"
# Run go test and capture combined stdout+stderr
output=$(go test -json -v -race \
    -coverprofile=coverage.out \
    -covermode=atomic \
    -coverpkg=./... \
    ./... 2>&1)

test_exit_code=$?   # exit code of go test

# Fail if any tests failed
if [ "$test_exit_code" -ne 0 ]; then
    echo -e "${RED}✗ One or more tests failed${NC}"
    exit 1
fi

# Fail if any tests were skipped
if echo "$output" | grep -q '"Action":"skip"'; then
    echo -e "${RED}✗ One or more tests were SKIPPED${NC}"
    exit 1
fi

echo -e "${GREEN}✓ All tests passed${NC}"

echo ""

###########################################
# 5. Security Scans (govulncheck, gosec, trivy)
###########################################
echo -e "${BLUE}[5/7] Running security scans...${NC}"

# govulncheck
echo -e "${BLUE}  Running govulncheck...${NC}"
if command -v govulncheck &> /dev/null; then
    if govulncheck ./... 2>&1; then
        echo -e "${GREEN}  ✓ govulncheck passed${NC}"
    else
        echo -e "${RED}  ✗ govulncheck found vulnerabilities${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}  ⚠ govulncheck not installed, skipping${NC}"
    echo -e "${YELLOW}    Install with: go install golang.org/x/vuln/cmd/govulncheck@latest${NC}"
fi

# gosec
echo -e "${BLUE}  Running gosec...${NC}"
if command -v gosec &> /dev/null; then
    if gosec -quiet ./... 2>&1; then
        echo -e "${GREEN}  ✓ gosec passed${NC}"
    else
        echo -e "${RED}  ✗ gosec found security issues${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}  ⚠ gosec not installed, skipping${NC}"
    echo -e "${YELLOW}    Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest${NC}"
fi

# trivy
echo -e "${BLUE}  Running trivy...${NC}"
if command -v trivy &> /dev/null; then
    if trivy fs . --scanners=vuln,misconfig,secret --exit-code 1 2>&1; then
        echo -e "${GREEN}  ✓ trivy passed${NC}"
    else
        echo -e "${RED}  ✗ trivy found issues${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}  ⚠ trivy not installed, skipping${NC}"
    echo -e "${YELLOW}    Install with: brew install trivy (macOS) or see https://trivy.dev${NC}"
fi

echo ""

###########################################
# 6. Safety Tests
###########################################
echo -e "${BLUE}[6/7] Running safety tests...${NC}"
safety_output=$(go test -json -v -tags=safety ./test/safety/... 2>&1)
safety_exit_code=$?

if [ "$safety_exit_code" -ne 0 ]; then
    echo -e "${RED}✗ Safety tests failed${NC}"
    exit 1
fi

if echo "$safety_output" | grep -q '"Action":"skip"'; then
    echo -e "${RED}✗ Safety tests were SKIPPED${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Safety tests passed${NC}"
echo ""

###########################################
# 7. Coverage Report (informational only)
###########################################
echo -e "${BLUE}[7/7] Reporting test coverage (informational)...${NC}"
if [ -f coverage.out ]; then
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    echo -e "Total coverage: ${YELLOW}${COVERAGE}%${NC}"
    echo -e "${BLUE}Note: Coverage is measured but does not fail checks${NC}"

    # Show coverage by package
    echo ""
    echo -e "${BLUE}Coverage by package:${NC}"
    go tool cover -func=coverage.out | grep -v "total:" | while read -r line; do
        if echo "$line" | grep -q "100.0%"; then
            echo -e "${GREEN}${line}${NC}"
        elif echo "$line" | awk '{if ($NF+0 >= 80) exit 0; else exit 1}' 2>/dev/null; then
            echo -e "${YELLOW}${line}${NC}"
        else
            echo -e "${line}"
        fi
    done
else
    echo -e "${YELLOW}⚠ No coverage file found${NC}"
fi
echo ""

###########################################
# Summary
###########################################
echo -e "${BLUE}================================================${NC}"
echo -e "${GREEN}✓ All checks passed! Ready to commit.${NC}"
echo -e "${BLUE}================================================${NC}"
exit 0
