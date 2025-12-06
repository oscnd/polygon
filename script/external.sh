#!/bin/zsh

set -e

# project directory
DIR=$(git rev-parse --show-toplevel)
EXTERNAL_DIR="$DIR/external"

# change to external directory
cd "$EXTERNAL_DIR"

# create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# clone repository
git clone --depth 1 --filter=blob:none --sparse https://github.com/sqlc-dev/sqlc.git "$TEMP_DIR"

# navigate to temporary directory and setup sparse checkout
cd "$TEMP_DIR"
git sparse-checkout init --cone
git sparse-checkout set internal

# create external/sqlc directory
cp -r -P internal "$EXTERNAL_DIR/sqlc/"

# go back to external directory
cd "$EXTERNAL_DIR"

# replace occurrences import path
find sqlc -name "*.go" -type f -exec sed -i '' 's|github\.com/sqlc-dev/sqlc/internal|go.scnd.dev/polygon/external/sqlc|g' {} +


