#!/bin/zsh

set -e
shopt -s extglob

# variables
UPSTREAM_REPO=$1
TAG_NAME=$2

# project directory
DIR=$(git rev-parse --show-toplevel)

# temporary directory
TEMP_DIR=$(mktemp -d)

# clone distributed repository
git clone --depth 1 "$UPSTREAM_REPO" "$TEMP_DIR"

# move polygon source
rm -r "${TEMP_DIR:-}/"!(.git|.gitattributes) || true
mkdir -p "$TEMP_DIR/external"
cp -rP "$DIR/external/"!(go.mod|go.sum) "$TEMP_DIR"/external/
cp -r "$DIR"/polygon/* "$TEMP_DIR"/
sed -i '/^replace /d' go.mod

# push
export GOPROXY=direct
( cd "$TEMP_DIR" && go mod tidy && git add . && git commit -m "Sync polygon source ${TAG_NAME}" && git tag -a "$TAG_NAME" -m "Tagging polygon source ${TAG_NAME}" && git push origin main --tags )

# cleanup
rm -rf "$TEMP_DIR"