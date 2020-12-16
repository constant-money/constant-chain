#!/usr/bin/env bash

echo "GOOS=js GOARCH=wasm go build -o ./build/privacy.wasm *.go"
GOOS=js GOARCH=wasm go build -o ./build/privacy.wasm *.go

# compress the binary. Takes a couple minutes
pushd build
# echo "Compressing WebAssembly binary"
# zopfli privacy.wasm

# echo "Compression complete. Deleting binary"
# echo "rm -rfv ./build/privacy.wasm"
# rm -rfv privacy.wasm
echo "Done"
# echo "Firing up basic server at port 2003"
# goexec 'http.ListenAndServe(`:2003`, gzipped.FileServer(gzipped.Dir(`./`)))'
# goexec 'http.ListenAndServe(`:2003`, http.FileServer(http.Dir(`./`)))'
popd