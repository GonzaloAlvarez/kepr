#!/bin/bash
set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

echo "=== Building kepr (dev mode) ==="
make dev

echo ""
echo "=== Building fake GitHub server ==="
go build -o bin/fakeghserver ./tests/fakeghserver/cmd

TEST_DIR=$(mktemp -d)
READY_FILE="$TEST_DIR/server.ready"

echo ""
echo "=== Test directory: $TEST_DIR ==="

cleanup() {
    echo ""
    echo "=== Cleaning up ==="
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        echo "Stopping fake server (PID: $SERVER_PID)..."
        kill -TERM "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    echo "Removing test directory..."
    rm -rf "$TEST_DIR"
    echo "Cleanup complete."
}
trap cleanup EXIT

echo ""
echo "=== Starting fake GitHub server ==="
./bin/fakeghserver \
    --port=0 \
    --repos-dir="$TEST_DIR/repos" \
    --ready-file="$READY_FILE" \
    --debug &
SERVER_PID=$!

TIMEOUT=20
echo "Waiting for server to start..."
while [ ! -f "$READY_FILE" ] && [ $TIMEOUT -gt 0 ]; do
    sleep 0.5
    TIMEOUT=$((TIMEOUT - 1))
done

if [ ! -f "$READY_FILE" ]; then
    echo "FAIL: Server did not start in time"
    exit 1
fi

SERVER_URL=$(cat "$READY_FILE")
echo "Server started at: $SERVER_URL"

export KEPR_CI=true
export GITHUB_HOST="$SERVER_URL"
export KEPR_HOME="$TEST_DIR/kepr"

echo ""
echo "=== Environment ==="
echo "  KEPR_CI=$KEPR_CI"
echo "  GITHUB_HOST=$GITHUB_HOST"
echo "  KEPR_HOME=$KEPR_HOME"

TEST_SECRET="my-super-secret-value-12345"

echo ""
echo "=== Running: kepr init testuser/test-repo ==="
./kepr init testuser/test-repo

echo ""
echo "=== Running: kepr add aws/main/keys ==="
echo "$TEST_SECRET" | ./kepr add aws/main/keys

echo ""
echo "=== Running: kepr get aws/main/keys ==="
RESULT=$(./kepr get aws/main/keys 2>/dev/null)

echo ""
echo "=== Validation (get) ==="
echo "Expected: $TEST_SECRET"
echo "Got:      $RESULT"

if [ "$RESULT" != "$TEST_SECRET" ]; then
    echo ""
    echo "=========================================="
    echo "  FAIL: Secret mismatch!"
    echo "=========================================="
    exit 1
fi
echo "PASS: Secret matches"

echo ""
echo "=== Running: kepr list ==="
LIST_ROOT=$(./kepr list 2>/dev/null)
echo "List output:"
echo "$LIST_ROOT"

if echo "$LIST_ROOT" | grep -q "aws/"; then
    echo "PASS: Found aws/ directory in root"
else
    echo "FAIL: Expected aws/ in list output"
    exit 1
fi

echo ""
echo "=== Running: kepr list aws ==="
LIST_AWS=$(./kepr list aws 2>/dev/null)
echo "List output:"
echo "$LIST_AWS"

if echo "$LIST_AWS" | grep -q "main/"; then
    echo "PASS: Found main/ directory in aws"
else
    echo "FAIL: Expected main/ in list aws output"
    exit 1
fi

echo ""
echo "=== Running: kepr list aws/main ==="
LIST_AWS_MAIN=$(./kepr list aws/main 2>/dev/null)
echo "List output:"
echo "$LIST_AWS_MAIN"

if echo "$LIST_AWS_MAIN" | grep -q "keys"; then
    echo "PASS: Found keys secret in aws/main"
else
    echo "FAIL: Expected keys in list aws/main output"
    exit 1
fi

echo ""
echo "=========================================="
echo "  PASS: E2E test completed successfully!"
echo "=========================================="
exit 0
