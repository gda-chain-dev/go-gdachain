#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
gdadir="$workspace/src/github.com/gdaereum"
if [ ! -L "$gdadir/go-gdaereum" ]; then
    mkdir -p "$gdadir"
    cd "$gdadir"
    ln -s ../../../../../. go-gdaereum
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$gdadir/go-gdaereum"
PWD="$gdadir/go-gdaereum"

# Launch the arguments with the configured environment.
exec "$@"
