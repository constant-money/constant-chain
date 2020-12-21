#!/usr/bin/env bash
echo "gomobile bind -x -v -target=android"
gomobile bind -x -v -target=android -ldflags -w
echo "gomobile bind -x -v -target=ios"
gomobile bind -x -v -target=ios -ldflags -w