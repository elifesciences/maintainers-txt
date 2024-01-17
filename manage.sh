#!/bin/bash
set -e

cmd="$1"

if test ! "$cmd"; then
    echo "command required."
    echo
    echo "available commands:"
    echo "  build        build project"
    echo "  clean        removes generated files"
    echo "  release      build project for distribution"
    echo "  deps.update  update project dependencies"
    exit 1
fi

shift
rest=$*

if test "$cmd" = "build"; then
    # CGO_ENABLED=0 skips CGO and linking against glibc to build static binaries.
    # -v 'verbose'
    CGO_ENABLED=0 go build \
        -v
    exit 0

elif test "$cmd" = "release"; then
    # GOOS is 'Go OS' and is being explicit in which OS to build for.
    # CGO_ENABLED=0 skips CGO and linking against glibc to build static binaries.
    # ld -s is 'disable symbol table'
    # ld -w is 'disable DWARF generation'
    # -trimpath removes leading paths to source files
    # -v 'verbose'
    # -o 'output'
    GOOS=linux CGO_ENABLED=0 go build \
        -ldflags="-s -w" \
        -trimpath \
        -v \
        -o linux-amd64
    sha256sum linux-amd64 > linux-amd64.sha256
    echo ---
    echo "wrote linux-amd64"
    echo "wrote linux-amd64.sha256"
    exit 0

elif test "$cmd" = "deps.update"; then
    # -u 'update modules [...] to use newer minor or patch releases when available'
    go get -u
    go mod tidy
    ./manage.sh build
    exit 0

elif test "$cmd" = "clean"; then
    shopt -s nullglob # continue when glob below is empty
    for file in *--maintainers.txt; do # created during normal operation
        rm "$file"
        echo "deleted: $file"
    done
    tbd=( linux-amd64 linux-amd64.sha256 )
    for file in "${tbd[@]}"; do # created with ./manage.sh release
        if [ -e "$file" ]; then
            rm "$file"
            echo "deleted $file"
        fi
    done
    exit 0
    
# ...

fi

echo "unknown command: $cmd"
exit 1
