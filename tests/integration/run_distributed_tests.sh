#!/bin/bash

# Script to run distributed worker integration tests

set -e

echo "🧪 Running Distributed Worker Integration Tests"
echo "================================================"

# Check if PostgreSQL is running
echo "📊 Checking PostgreSQL connection..."
export TEST_POSTGRES_HOST=${TEST_POSTGRES_HOST:-localhost}
export TEST_POSTGRES_USER=${TEST_POSTGRES_USER:-aetherium}
export TEST_POSTGRES_PASSWORD=${TEST_POSTGRES_PASSWORD:-aetherium}
export TEST_POSTGRES_DB=${TEST_POSTGRES_DB:-aetherium_test}

# Try to connect to PostgreSQL
if ! psql -h "$TEST_POSTGRES_HOST" -U "$TEST_POSTGRES_USER" -d postgres -c '\q' 2>/dev/null; then
    echo "❌ Error: Cannot connect to PostgreSQL"
    echo "   Please ensure PostgreSQL is running and accessible"
    echo "   Connection details:"
    echo "     Host: $TEST_POSTGRES_HOST"
    echo "     User: $TEST_POSTGRES_USER"
    echo "     DB:   $TEST_POSTGRES_DB"
    echo ""
    echo "   Start PostgreSQL with:"
    echo "   docker run -d --name aetherium-test-postgres \\"
    echo "     -e POSTGRES_USER=$TEST_POSTGRES_USER \\"
    echo "     -e POSTGRES_PASSWORD=$TEST_POSTGRES_PASSWORD \\"
    echo "     -e POSTGRES_DB=$TEST_POSTGRES_DB \\"
    echo "     -p 5432:5432 postgres:15-alpine"
    exit 1
fi

echo "✅ PostgreSQL connection successful"

# Create test database if it doesn't exist
echo "📦 Creating test database..."
psql -h "$TEST_POSTGRES_HOST" -U "$TEST_POSTGRES_USER" -d postgres \
  -c "CREATE DATABASE $TEST_POSTGRES_DB" 2>/dev/null || echo "Database already exists"

# Run the tests
echo ""
echo "🏃 Running tests..."
echo ""

cd "$(dirname "$0")"

go test -v -timeout 30m \
  -run "TestWorker|TestMultiWorker" \
  ./distributed_workers_test.go

echo ""
echo "✅ All tests completed!"
