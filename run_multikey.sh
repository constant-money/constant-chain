#!/usr/bin/env bash

if [ "$1" == "allnode" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --miningkeys "117FYq9sjG87ny4nFq9cqdtFonzJ5BFpdidLJ7FCXe2dmxEEBu,12CeFkSTYxzTG8eQyEeGaof4VWGyqav6qMrXUkwtHqKU1yA2cRU,12njcQBofHdiLVA6qZAC9apvBnS2SSfTHon2AsBixKBfahz7eQR,1TmwTeXvXQMfb82Hb7fPuGGuySwTqQrXERMEzmZgUMcJq2ybYX,126ictrQpcri19gXPysssCVZ2Kjh98wVfzxFSMqwGN34HunkqzB,12NDsWtAdfKbkDpYQty2rt5r2K5cxdMwAqRZLoHtwojuvJTyPaB,12USvsQpy8DdsnA9MKiNYz3ZxLCwyf3NUVJ5vagfn7X22Sa4v4i,1UHTiXaQqQ3xfvgy8sYURcwM7dAuTZEkp5ovETtobD4YSrDRWq" --nodemode "auto" --datadir "data/shard0-0" --listen "0.0.0.0:9434" --externaladdress "0.0.0.0:9434" --norpcauth --rpclisten "0.0.0.0:9334" --rpcwslisten "0.0.0.0:19334" 
fi 
if [ "$1" == "fullnode" ]; then
./incognito --testnet true --nodemode "relay" --relayshards "all" --externaladdress "127.0.0.1:9433" --enablewallet --wallet "wallet" --walletpassphrase "12345678" --walletautoinit --norpcauth --datadir "../testnet/fullnode" --discoverpeersaddress "127.0.0.1:9330" --norpcauth --rpclisten "0.0.0.0:8334" --rpcwslisten "127.0.0.1:18338" 2>&1 | tee log.log
fi

