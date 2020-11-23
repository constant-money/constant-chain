#!/usr/bin/env bash


if [ "$1" == "b0" ];  then 
Profiling=18080 ./incognito --name "b0" --discoverpeersaddress "0.0.0.0:9330" --miningkeys "117FYq9sjG87ny4nFq9cqdtFonzJ5BFpdidLJ7FCXe2dmxEEBu" --nodemode "auto" --datadir "data/b0" --listen "0.0.0.0:19430" --externaladdress "0.0.0.0:19430" --norpcauth --rpclisten "0.0.0.0:19330" --rpcwslisten "0.0.0.0:29330"
fi

if [ "$1" == "b1" ];  then 
./incognito --name "b1" --discoverpeersaddress "0.0.0.0:9330" --miningkeys "12CeFkSTYxzTG8eQyEeGaof4VWGyqav6qMrXUkwtHqKU1yA2cRU" --nodemode "auto" --datadir "data/b1" --listen "0.0.0.0:19431" --externaladdress "0.0.0.0:19431" --norpcauth --rpclisten "0.0.0.0:19331" --rpcwslisten "0.0.0.0:29331"
fi

if [ "$1" == "b2" ];  then 
./incognito --name "b2" --discoverpeersaddress "0.0.0.0:9330" --miningkeys "12njcQBofHdiLVA6qZAC9apvBnS2SSfTHon2AsBixKBfahz7eQR" --nodemode "auto" --datadir "data/b2" --listen "0.0.0.0:19432" --externaladdress "0.0.0.0:19432" --norpcauth --rpclisten "0.0.0.0:19332" --rpcwslisten "0.0.0.0:29332"
fi

if [ "$1" == "b3" ];  then 
./incognito --name "b3" --discoverpeersaddress "0.0.0.0:9330" --miningkeys "1TmwTeXvXQMfb82Hb7fPuGGuySwTqQrXERMEzmZgUMcJq2ybYX" --nodemode "auto" --datadir "data/b3" --listen "0.0.0.0:19433" --externaladdress "0.0.0.0:19433" --norpcauth --rpclisten "0.0.0.0:19333" --rpcwslisten "0.0.0.0:29333"
fi

if [ "$1" == "s00" ];  then 
Profiling=18081 ./incognito --name "s00" --discoverpeersaddress "0.0.0.0:9330" --miningkeys "126ictrQpcri19gXPysssCVZ2Kjh98wVfzxFSMqwGN34HunkqzB" --nodemode "auto" --datadir "data/s00" --listen "0.0.0.0:18430" --externaladdress "0.0.0.0:18430" --norpcauth --rpclisten "0.0.0.0:18330" --rpcwslisten "0.0.0.0:29330"
fi

if [ "$1" == "s01" ];  then 
./incognito --name "s01" --discoverpeersaddress "0.0.0.0:9330" --miningkeys "12NDsWtAdfKbkDpYQty2rt5r2K5cxdMwAqRZLoHtwojuvJTyPaB" --nodemode "auto" --datadir "data/s01" --listen "0.0.0.0:18431" --externaladdress "0.0.0.0:18431" --norpcauth --rpclisten "0.0.0.0:18331" --rpcwslisten "0.0.0.0:29331"
fi

if [ "$1" == "s02" ];  then 
./incognito --name "s02" --discoverpeersaddress "0.0.0.0:9330" --miningkeys "12USvsQpy8DdsnA9MKiNYz3ZxLCwyf3NUVJ5vagfn7X22Sa4v4i" --nodemode "auto" --datadir "data/s02" --listen "0.0.0.0:18432" --externaladdress "0.0.0.0:18432" --norpcauth --rpclisten "0.0.0.0:18332" --rpcwslisten "0.0.0.0:29332"
fi

if [ "$1" == "s03" ];  then 
./incognito --name "s03" --discoverpeersaddress "0.0.0.0:9330" --miningkeys "1UHTiXaQqQ3xfvgy8sYURcwM7dAuTZEkp5ovETtobD4YSrDRWq" --nodemode "auto" --datadir "data/s03" --listen "0.0.0.0:18433" --externaladdress "0.0.0.0:18433" --norpcauth --rpclisten "0.0.0.0:18333" --rpcwslisten "0.0.0.0:29333"
fi