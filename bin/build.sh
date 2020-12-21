#!/usr/bin/env bash

if [ -f ./incognito ]; then
    rm -rf ./incognito
fi
if [ -f ./bootnode ]; then
    rm -rf ./bootnode
fi
env CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags '-w' -o incognito ../*.go
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ENV=testnet go build -ldflags '-w' -o incognito-test ../tests/*.go
cp ../keylist.json .
cp ../keylist-v2.json .
cp ../keylist_256.json .
cp ../sample-config.conf .

commit=`git show --summary --oneline | cut -d ' ' -f 1`
docker build --build-arg commit=$commit . -t incognitochaintestnet/incognito:${tag} && docker push incognitochaintestnet/incognito:${tag} && echo "Commit: $commit"
docker rmi -f $(docker images --filter "dangling=true" -q)
