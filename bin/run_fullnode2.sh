#!/bin/sh
mkdir -p /data
if [ "$1" == "y" ]; then
    rm -rf /data/*
fi

echo ./incognito -n $NAME --discoverpeers --discoverpeersaddress $DISCOVERPEERSADDRESS --testnet 1 --testnetversion "2" --nodemode "relay" --relayshards "all" --datadir "/data" --listen "0.0.0.0:$PORT" --externaladdress "$EXTERNALADDRESS:$PORT" --norpcauth --enablewallet --wallet "incognito" --walletpassphrase "12345678" --walletautoinit --rpclisten "0.0.0.0:$RPC_PORT" --metricurl "$METRIC_URL" > cmd.sh

./incognito -n $NAME --discoverpeers --discoverpeersaddress $DISCOVERPEERSADDRESS --testnet 1 --testnetversion "2" --nodemode "relay" --relayshards "all" --datadir "/data" --listen "0.0.0.0:$PORT" --externaladdress "$EXTERNALADDRESS:$PORT" --norpcauth --enablewallet --wallet "incognito" --walletpassphrase "12345678" --walletautoinit --rpclisten "0.0.0.0:$RPC_PORT" --metricurl "$METRIC_URL"> /data/log.txt 2>/data/error_log.txt
