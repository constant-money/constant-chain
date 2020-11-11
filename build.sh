#!/usr/bin/env bash
echo "Start Install Dependencies Package"
GO111MODULE=on go get -v

cd ./blockchain/committeestate/ && mockery --name=BeaconCommitteeEngine && cd -
cd ./metadata/ && mockery --name=ChainRetriever && mockery --name=BeaconViewRetriever && mockery --name=ShardViewRetriever && cd -
echo "Start Unit-Test"
echo "package committeestate"
GO111MODULE=on go test -cover ./blockchain/committeestate/*.go
echo "package statedb"
GO111MODULE=on go test -cover ./dataaccessobject/statedb/*.go
echo "package instruction"
GO111MODULE=on go test -cover ./instruction/*.go
echo "package blockchain"
GO111MODULE=on go test -cover ./blockchain/*.go
echo "package metadata"
GO111MODULE=on go test -cover ./metadata/*.go


echo "Start build Incognito"
APP_NAME="incognito"
echo "go build -o $APP_NAME"
GO111MODULE=on go build -o $APP_NAME

echo "Build Incognito success!"
