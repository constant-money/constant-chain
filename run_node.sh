#!/usr/bin/env bash
#GrafanaURL=http://128.199.96.206:8086/write?db=mydb
###### MULTI_MEMBERS
# Shard 0
if [ "$1" == "shard0-0" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXB47RhSdyVRU41TEf78nxbtWGtmjutwSp9YqsNaCpFxQGXcnwcXTtBkCGDk1KLBRBeWMvb2aXG5SeDUJRHtFV8jTB3weHEkbMJ1AL" --nodemode "auto" --datadir "data/shard0-0" --listen "0.0.0.0:9434" --externaladdress "0.0.0.0:9434" --norpcauth --rpclisten "0.0.0.0:9334" --rpcwslisten "0.0.0.0:19334" 
fi
if [ "$1" == "shard0-1" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXVdfBqBMigSs5fm9NSS8rgsVVURUxArpv6DxYmPZujKqomqUa2H9wh1zkkmDGtDn2woK4NuRDYnYRtVkUhK34TMfbUF4MShSkrCw5" --nodemode "auto" --datadir "data/shard0-1" --listen "0.0.0.0:9435" --externaladdress "0.0.0.0:9435" --norpcauth --rpclisten "0.0.0.0:9335" --rpcwslisten "0.0.0.0:19335" 
fi
if [ "$1" == "shard0-2" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXi8eKJ5RYJjyQYcFMThfbXHgaL6pq5AF5bWsDXwfsw8pqQUreDv6qgWyiABoDdphvqE7NFr9K92aomX7Gi5Nm1e4tEoV3qRLVdfSR" --nodemode "auto" --datadir "data/shard0-2" --listen "0.0.0.0:9436" --externaladdress "0.0.0.0:9436" --norpcauth --rpclisten "0.0.0.0:9336" 
fi
if [ "$1" == "shard0-3" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnY42xRqJghQX3zvhgEa2ZJBwSzJ46SXyVQEam1yNpN4bfAqJwh1SsobjHAz8wwRvwnqJBfxrbwUuTxqgEbuEE8yMu6F14QmwtwyM43" --nodemode "auto" --datadir "data/shard0-3" --listen "0.0.0.0:9437" --externaladdress "0.0.0.0:9437" --norpcauth --rpclisten "0.0.0.0:9337" 
fi
if [ "$1" == "shard0-new-0" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rospAEaouNQgnK8vAAJGzH6ysLAGeZGqmZ5RJTT7CrF1zK8zwqVqx4DEdoD6MDTNiSK9W1vbZXtVe7vqvfEuf6LpuBbUiHvvkfF9L3X" --nodemode "auto" --datadir "data/shard0-0" --listen "0.0.0.0:9434" --externaladdress "0.0.0.0:9434" --norpcauth --rpclisten "0.0.0.0:9334" --rpcwslisten "0.0.0.0:19334"
fi
if [ "$1" == "shard0-new-1" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rs4QMxRdQLLyG7b5ifeBwH39TuZU92UuTXHynDKVgtE366Jd6qs99gj6gKtz46ad5NKXbaJ2UyXfxbjCouhtN6Es8ve5yyQVyEtgXod" --nodemode "auto" --datadir "data/shard0-1" --listen "0.0.0.0:9435" --externaladdress "0.0.0.0:9435" --norpcauth --rpclisten "0.0.0.0:9335" --rpcwslisten "0.0.0.0:19335"
fi
if [ "$1" == "shard0-new-2" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rtB7mXUvqSqgGJZTa84JwUJyruLXNYwrypD98t8UkAGxhBhhs6P696Z7iZ1WxdhWEFKeDbEkR5PdXNf4V8CRwmszhjUAU6AqyXQ6ME7" --nodemode "auto" --datadir "data/shard0-2" --listen "0.0.0.0:9436" --externaladdress "0.0.0.0:9436" --norpcauth --rpclisten "0.0.0.0:9336"
fi
if [ "$1" == "shard0-new-3" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rwXq7wh46QCmZUuqUTaQwbrXi7mvZnPayhjaUqa2MCnGbXRxHQhxyShfRLpvQzUzqqmDGMqfmLR6R2WayyyBhXU1b6Hhz8CnZVPTBG7" --nodemode "auto" --datadir "data/shard0-3" --listen "0.0.0.0:9437" --externaladdress "0.0.0.0:9437" --norpcauth --rpclisten "0.0.0.0:9337"
fi
# Shard 1
if [ "$1" == "shard1-0" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXBPJQWJTyPdzWsfsUCFTDhcas3y2MYsauKo66euh1udG8dSh2ZszSbfqHwCpYHPRSpFTxYkUcVa619XUM6DjdV7FfUWvYoziWE2Bm" --nodemode "auto" --datadir "data/shard1-0" --listen "0.0.0.0:9438" --externaladdress "0.0.0.0:9438" --norpcauth --rpclisten "0.0.0.0:9338" --rpcwslisten "127.0.0.1:19338" 
fi
if [ "$1" == "shard1-1" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXN2SLxQncPYvFdzEivznKjBxK5byYmPbAhnEEv8TderLG7NUD7nwAEDu7DJ7pnCKw9N5PuTuELCHz8qKc7z9S9jF8QG41u7Vomc6L" --nodemode "auto" --datadir "data/shard1-1" --listen "0.0.0.0:9439" --externaladdress "0.0.0.0:9439" --norpcauth --rpclisten "0.0.0.0:9339" --rpcwslisten "127.0.0.1:19339" 
fi
if [ "$1" == "shard1-2" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXs5os49h71E7utfHatnWGQnirbVF2b5Ua8h1ttidk1S5AFcUqHCDmpMziiFC15BG8W1LQKK5tYcvr2CM7DyYgsfVmAWYh4kQ6f33T" --nodemode "auto" --datadir "data/shard1-2" --listen "0.0.0.0:9440" --externaladdress "0.0.0.0:9440" --norpcauth --rpclisten "0.0.0.0:9340" 
fi
if [ "$1" == "shard1-3" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXvcE6sxwt7nQ6mH6KdPMWyQRv6xAd3WWorzS7k26YPjm4mvFtC51bRaU18yubQm1N3gBeDJJyXqWmxi5QdCkqYExCEkSqNpD1Wzpz" --nodemode "auto" --datadir "data/shard1-3" --listen "0.0.0.0:9441" --externaladdress "0.0.0.0:9441" --norpcauth --rpclisten "0.0.0.0:9341" 
fi
# Shard 2
if [ "$1" == "shard2-0" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnX2Fngsy1KuJv5GLLNUeB3gZwoHMss2cqRr1ECa2ibR7FQNUyE7kMFvq7rGtqVJULo8XfAxThuLxUwd8vv76MojbL3wPhxmTvbcd2S" --nodemode "auto" --datadir "data/shard2-0" --listen "0.0.0.0:9442" --externaladdress "0.0.0.0:9442" --norpcauth --rpclisten "0.0.0.0:9342"
fi
if [ "$1" == "shard2-1" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXRrZ1gC7MYNFVUA1paZrE3iSiAb9AR7Z5quNBgR2ovrcfj8p4kTb3ynx6ddjnoPey3qA2vRiP17tCvpCHU9xBDwMq8D1Mg2GBM9eC" --nodemode "auto" --datadir "data/shard2-1" --listen "0.0.0.0:9443" --externaladdress "0.0.0.0:9443" --norpcauth --rpclisten "0.0.0.0:9343"
fi
if [ "$1" == "shard2-2" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXbsJF4f5xtzM2jPW6dCYHChNksth7X64iUkLR4bGi2JBVjgJQRLeKRsdbiFaYMsxzrfbfKAp4TELGre45QkxHWCnwVXPGnnZjJKVL" --nodemode "auto" --datadir "data/shard2-2" --listen "0.0.0.0:9444" --externaladdress "0.0.0.0:9444" --norpcauth --rpclisten "0.0.0.0:9344"
fi
if [ "$1" == "shard2-3" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnY4DeSGZYb8r8sSN5WJr6ZL3NCafYAQ7f7Am9KXQDGSc3Qddpn7BfHW1i6CoVVk8vKEzJ25vA9uc9EdhoLU98eoUw7fMrPPrBdNB7Q" --nodemode "auto" --datadir "data/shard2-3" --listen "0.0.0.0:9445" --externaladdress "0.0.0.0:9445" --norpcauth --rpclisten "0.0.0.0:9345"
fi
# Shard 3
if [ "$1" == "shard3-0" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXBg35mhRzZDw7nNQCiRcM9QLg2DTHewY7kRZ3XCmgkdQa4iVcMUwMd5DTvvjEcCvv3SnCo5zSQpS93zskAxG6tdvR1QPBxtCmaBCK" --nodemode "auto" --datadir "data/shard3-0" --listen "0.0.0.0:9446" --externaladdress "0.0.0.0:9446" --norpcauth --rpclisten "0.0.0.0:9346"
fi
if [ "$1" == "shard3-1" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXa3USo2wMdyuZTfurZrYe1ZJhqibnp9GHx8Jkf9dh5cU39sjqBTKoWPtNHvVZn2eqGj6V26PmELez85bUUBMBKG6tqFQrer2GkuJ4" --nodemode "auto" --datadir "data/shard3-1" --listen "0.0.0.0:9447" --externaladdress "0.0.0.0:9447" --norpcauth --rpclisten "0.0.0.0:9347"
fi
if [ "$1" == "shard3-2" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXc38wPUsDdRasThfk58ME3KoSdeyXJS3tsyRuxPrSkvuc8MBT1gxiQjSCMBbifqibxEVAHimGcfnLbjVEEett8FtwFuBQ8zAUHTet" --nodemode "auto" --datadir "data/shard3-2" --listen "0.0.0.0:9448" --externaladdress "0.0.0.0:9448" --norpcauth --rpclisten "0.0.0.0:9348"
fi
if [ "$1" == "shard3-3" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnY6BgeU6bk7Nui1TwB2w2MsG3UeSXrgJ1n2tGVs66Qk9YT6E15PbXR4ai7eW1qyPrW7a2AUtN2otuXtBAZKtm2DDiUxh3ngZ6JPYHv" --nodemode "auto" --datadir "data/shard3-3" --listen "0.0.0.0:9449" --externaladdress "0.0.0.0:9449" --norpcauth --rpclisten "0.0.0.0:9349"
fi
# Beacon
if [ "$1" == "beacon-0" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXCerQX2RRd8KhPfsFCj2rrBYUx42FZJKgRFcdBfg36Mid3ygKyMn5LSc5LBHsxqapRaN6xMav7bGhA6VtGUzNNYuA9Y78CB5oGkti" --nodemode "auto" --datadir "data/beacon-0" --listen "0.0.0.0:9450" --externaladdress "0.0.0.0:9450" --norpcauth --rpclisten "0.0.0.0:9350" --rpcwslisten "0.0.0.0:19350"
fi
if [ "$1" == "beacon-1" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXYgxipKvTJJfHg7tQhcdmA2R1jPpCPmXg37Xi1VfgrFzWFuNy4U6828q1yfbD7VEdutD63HfVYAqL6U32joXVjqdkfUP52LnNGXda" --nodemode "auto" --datadir "data/beacon-1" --listen "0.0.0.0:9451" --externaladdress "0.0.0.0:9451" --norpcauth --rpclisten "0.0.0.0:9351" 
fi
if [ "$1" == "beacon-2" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXe3Jxg5d1Rejg2fB1NwnqNsr94RCT3PX14h5NNDjrdgLeEWFkqcMNamKCHask1Gx46g5WYZDKHKx7kzLVD7h1cgvU6NxNijkyGmA9" --nodemode "auto" --datadir "data/beacon-2" --listen "0.0.0.0:9452" --externaladdress "0.0.0.0:9452" --norpcauth --rpclisten "0.0.0.0:9352" 
fi
if [ "$1" == "beacon-3" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnY2gqonwhnhGD6rKeEXkbJDB7DHUtZQKC8SfLci6ABb5eCEj4o7ezWBZWaGbu7CJ1R1mrADGqmRjugg42GeA6jhaXbNDeP2HUr8udw" --nodemode "auto" --datadir "data/beacon-3" --listen "0.0.0.0:9453" --externaladdress "0.0.0.0:9453" --norpcauth --rpclisten "0.0.0.0:9353" 
fi
# Beacon
if [ "$1" == "beacon-new-0" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8sSijugr8azxAiHMWS9rA22grKDv5o7AEXQ9datpT1V7N5FLHiJMjvfVnXcitL3fpj35Xt5DNnBq8iFq618X31nCgn2RjrYx5tZZWCtj" --nodemode "auto" --datadir "data/beacon-0" --listen "0.0.0.0:9460" --externaladdress "0.0.0.0:9460" --norpcauth --rpclisten "0.0.0.0:9360"
fi
if [ "$1" == "beacon-new-1" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8sSjSEck5J5RKWGWurVfigruYDxEzjjVPqHaTRJ57YFNo7gXBH8onUQxtdpoyFnBZrLhfGWQ4k4MNadwa6F7qYwcuFLW9R1VxTfN7q4d" --nodemode "auto" --datadir "data/beacon-1" --listen "0.0.0.0:9461" --externaladdress "0.0.0.0:9461" --norpcauth --rpclisten "0.0.0.0:9361"
fi
if [ "$1" == "beacon-new-2" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8sSjkAqVJi4KkbCS75GYrsag7QZYP7FTPRfZ63D1AJgzfmdHnE9sbpdJV4Kx5tN9MgbqbRYDgzER2xpgsxrHvWxNgTHHrghYwLJLfe2R" --nodemode "auto" --datadir "data/beacon-2" --listen "0.0.0.0:9462" --externaladdress "0.0.0.0:9462" --norpcauth --rpclisten "0.0.0.0:9362"
fi
if [ "$1" == "beacon-new-3" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8sSj637mhpaJUboUEjkXsEUQm8q82T6kND3mWtNwig71qX2aFeZegWYsLVtyxBWdiZMBoNkdJ1MZYAcWetUP8DjYFnUac4vW7kzHfYsc" --nodemode "auto" --datadir "data/beacon-3" --listen "0.0.0.0:9463" --externaladdress "0.0.0.0:9463" --norpcauth --rpclisten "0.0.0.0:9363"
fi
# FullNode testnet
if [ "$1" == "fullnode-testnet1" ]; then
GETH_NAME=kovan.infura.io/v3/1f15d22470684b4a8c92c130925fc679 GETH_PROTOCOL=https GETH_PORT="" ./incognito --testnet true --nodemode "relay" --relayshards "all" --externaladdress "127.0.0.1:9433" --enablewallet --wallet "wallet" --walletpassphrase "12345678" --walletautoinit --norpcauth --datadir "../testnet/fullnode" --discoverpeersaddress "testnet-bootnode.incognito.org:9330" --norpcauth --rpclisten "0.0.0.0:8334" --rpcwslisten "127.0.0.1:18338" > ../testnet/log.txt 2> ../testnet/error_log.txt &
fi
if [ "$1" == "fullnode-testnet12" ]; then
GETH_NAME=kovan.infura.io/v3/1f15d22470684b4a8c92c130925fc679 GETH_PROTOCOL=https GETH_PORT="" ./incognito --testnet true --nodemode "relay" --relayshards "all" --externaladdress "127.0.0.1:9432" --enablewallet --wallet "wallet" --walletpassphrase "12345678" --walletautoinit --norpcauth --datadir "../testnet12/fullnode" --discoverpeersaddress "testnet-bootnode.incognito.org:9330" --norpcauth --rpclisten "0.0.0.0:7334" --rpcwslisten "127.0.0.1:17338" > ../testnet12/log.txt 2> ../testnet12/error_log.txt &
fi
if [ "$1" == "fullnode-testnet2" ]; then
GETH_NAME=kovan.infura.io/v3/1f15d22470684b4a8c92c130925fc679 GETH_PROTOCOL=https GETH_PORT="" ./incognito --testnet 1 --testnetversion "2" --nodemode "relay" --relayshards "all" --externaladdress "127.0.0.1:9431" --enablewallet --wallet "wallet" --walletpassphrase "12345678" --walletautoinit --norpcauth --datadir "../testnet2/fullnode" --discoverpeersaddress "74.207.247.250:9330" --norpcauth --rpclisten "0.0.0.0:6334" --rpcwslisten "127.0.0.1:16338" > ../testnet2/log.txt 2> ../testnet2/error_log.txt &
fi
if [ "$1" == "fullnode-testnet-b" ]; then
GO111MODULE=on GETH_NAME=kovan.infura.io/v3/1138a1e99b154b10bae5c382ad894361 GETH_PROTOCOL=https GETH_PORT="" ./incognito --testnet true --nodemode "relay" --relayshards "[0]" --externaladdress "127.0.0.1:9433" --enablewallet --wallet "wallet" --walletpassphrase "12345678" --walletautoinit --norpcauth --datadir "../testnet/fullnode" --discoverpeersaddress "testnet-bootnode.incognito.org:9330" --norpcauth --rpclisten "0.0.0.0:8334" --rpcwslisten "127.0.0.1:18338" > ../testnet/log.txt 2> ../testnet/error_log.txt &
fi
if [ "$1" == "fullnode-mainnet" ]; then
./incognito --testnet true --nodemode "relay" --externaladdress "127.0.0.1:9433" --enablewallet --wallet "wallet" --walletpassphrase "12345678" --walletautoinit --norpcauth --datadir "../mainnet/fullnode" --discoverpeersaddress "mainnet-bootnode.incognito.org:9330"
fi
######
if [ "$1" == "shard-candidate0-1" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rq19Uu7UGbTApZzZwCAvVszAgRNAzHzr3p8Cu75jPH3h5AUtRXMKiqF3hw8NbEfeLcjtbpeUvJfw4tGj7pbqwDYngc8wB13Gf77o33f" --nodemode "auto" --datadir "data/shard-stake" --listen "127.0.0.1:9455" --externaladdress "127.0.0.1:9455" --norpcauth --rpclisten "127.0.0.1:9355"
fi
if [ "$1" == "shard-candidate0-2" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rrEW3NPNgU8xzbeqE7cr4WTT8JvyaQqSZyczA5hBJVvpQMTBVqNfcCdzhvquWCHH11jHihZtgyJqbdWPhWYbmmsw5aV29WSXBEsgbVX" --nodemode "auto" --datadir "data/shard-stake-2" --listen "127.0.0.1:9456" --externaladdress "127.0.0.1:9456" --norpcauth --rpclisten "127.0.0.1:9356"
fi
if [ "$1" == "shard-candidate0-3" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnY15HgjpbJn1es84ysseB6q9UQ5SwB6Eb82yejEQ3yzhd1dm5ShEiezdfMoEzBgvkuKcFdP5TY3SuWNHXKa1Krprsfxnk5wy7wZ6Dg" --nodemode "auto" --datadir "data/shard-stake-6" --listen "0.0.0.0:9460" --externaladdress "0.0.0.0:9460" --norpcauth --rpclisten "0.0.0.0:9360"
fi
if [ "$1" == "shard-candidate1-1" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8roHikeAFyuBpdCU76kXurEqrC9VYWyRyfFb6PwX6nip9KGYbwpXL78H92mUoWK2GWkA2WysgXbHqwSxnC6XCkmtxBVb3zJeCXgfcYyL" --nodemode "auto" --datadir "data/shard-stake-3" --listen "0.0.0.0:9457" --externaladdress "0.0.0.0:9457" --norpcauth --rpclisten "0.0.0.0:9357"
fi
if [ "$1" == "shard-candidate1-2" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rr4sE2L8WzsVNEN9WsiGcMTDCmEH9TC1ZK8517cxURRFNoWoStYQTgqXpiAMU4gzmkmnWahHdGvQqFaY1JTVsn3nHfD5Ppgz8hQDiVC" --nodemode "auto" --datadir "data/shard-stake-4" --listen "0.0.0.0:9458" --externaladdress "0.0.0.0:9458" --norpcauth --rpclisten "0.0.0.0:9358"
fi
if [ "$1" == "shard-candidate1-3" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rtt9Kd5LUcfXNmd7aMnQehCnKabArVB3BUk2RHVjeh88x5MJnJY4okB8JdFm4JNm4A2WjSe58qWNVkJPEFjpLHNYfKHpWfRdqyfDD9f" --nodemode "auto" --datadir "data/shard-stake-5" --listen "0.0.0.0:9459" --externaladdress "0.0.0.0:9459" --norpcauth --rpclisten "0.0.0.0:9359"
fi
if [ "$1" == "shard-candidate1-4" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnY15HgjpbJn1es84ysseB6q9UQ5SwB6Eb82yejEQ3yzhd1dm5ShEiezdfMoEzBgvkuKcFdP5TY3SuWNHXKa1Krprsfxnk5wy7wZ6Dg" --nodemode "auto" --datadir "data/shard-stake-7" --listen "0.0.0.0:9461" --externaladdress "0.0.0.0:9461" --norpcauth --rpclisten "0.0.0.0:9361"
fi
if [ "$1" == "shard-candidate1-5" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rna7M8BYBfNjNHmw3Tie6Yir9mQgp5rSRgUngTqn6A6iSRvAPex4sXsmGxVzXcpUUDfnRfRys3QrPnTHauiipdUNtj7Ef6t3mHUwiC3" --nodemode "auto" --datadir "data/shard-stake-8" --listen "0.0.0.0:9462" --externaladdress "0.0.0.0:9462" --norpcauth --rpclisten "0.0.0.0:9362"
fi
#if [ "$1" == "shard-stake-9" ]; then
#./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnaLC8yRN5im7BgETP2y6nDbWrxfn2sfQaJvDqV7siRoLLaYnaehad7dY4L7n3dTd4XbYFfbr867vFq2uqCm36PmTq9usop6oH3MKQf" --nodemode "auto" --datadir "data/shard-stake-9" --listen "0.0.0.0:9463" --externaladdress "0.0.0.0:9463" --norpcauth --rpclisten "0.0.0.0:9363"
#fi
####full node
# ./incognito --discoverpeersaddress "0.0.0.0:9330" --nodemode "relay" --relayshards "all" --datadir "data/fullnode" --listen "0.0.0.0:9459" --externaladdress "0.0.0.0:9459" --norpcauth --rpclisten "0.0.0.0:9359"

###### SINGLE_MEMBER
######
## Shard: 0, Role: Proposer
#if [ "$1" == "shard0-proposer" ]; then
#./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rqJHgJp2TPpNpLNx34aWHB5VH5Pys3hVjjhhf9tctVeCNmX2zQLBqzHau6LpUbSV52kXtG2hRZsuYWkXWF5kw2v24RJq791fWmQxVqy" --nodemode "auto" --datadir "data/shard-0" --listen "0.0.0.0:9460" --externaladdress "0.0.0.0:9460" --norpcauth --rpclisten "0.0.0.0:9360"
#fi
## Shard: 1, Role: Proposer
#if [ "$1" == "shard1-proposer" ]; then
#./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rrEEcDQBMnUM5J17qniHZZmckmr8LGCv9nBjP9x5wmGFGUryKTNvEAf1jh2wwW69rxwtANq4m8JmzowfKVPayUHPmAKdwQw5718GKuH" --nodemode "auto" --datadir "data/shard-1" --listen "0.0.0.0:9461" --externaladdress "0.0.0.0:9461" --norpcauth --rpclisten "0.0.0.0:9361"
#fi
## Beacon, Role: Proposer
#if [ "$1" == "shard2-proposer" ]; then
#./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rotpPVSeHrknwVUTLQgy2avatUWKh2oV9EjVMw6eEtwyJT1FsrHGzvBaLpHL4gPVfJjuSUWvTtiTKuWGNNwGuLo8SHCgfA36ttJ5J7u" --nodemode "auto" --datadir "data/beacon" --listen "0.0.0.0:9462" --externaladdress "0.0.0.0:9462" --norpcauth --rpclisten "0.0.0.0:9362"
#fi
## Relay node
#if [ "$1" == "relaynode" ]; then
#./incognito --relayshards "all" --datadir "data/relaynode" --listen "0.0.0.0:9463" --externaladdress "0.0.0.0:9463" --norpcauth --rpclisten "0.0.0.0:9363" --enablewallet --wallet "wallet3" --walletpassphrase "12345678" --walletautoinit
#fi
