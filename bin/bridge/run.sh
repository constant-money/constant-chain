#!/bin/sh bash

run()
{
  validator_key=xxx
  bootnode="testnet-bootnode.incognito.org:9330"
  latest_tag=$1
  current_tag=$2
  data_dir="data"
  eth_data_dir="eth-kovan-data"

  docker -v || bash -c "wget -qO- https://get.docker.com/ | sh"

  if [ ! -d "$PWD/${eth_data_dir}" ]
  then
    mkdir $PWD/${eth_data_dir}
    chmod -R 777 $PWD/${eth_data_dir}
  fi

  docker rm -f inc_miner
  docker rm -f inc_kovan
  if [ "$current_tag" != "" ]
  then
    docker image rm -f incognitochaintestnet/incognito:${current_tag}
  fi

  docker pull incognitochaintestnet/incognito:${latest_tag}
  docker network create --driver bridge inc_net || true

  docker run -ti --restart=always --net inc_net -d -p 8545:8545  -p 30303:30303 -p 30303:30303/udp -v $PWD/${eth_data_dir}:/home/parity/.local/share/io.parity.ethereum/ --name inc_kovan  parity/parity:stable --light  --chain kovan  --jsonrpc-interface all --jsonrpc-hosts all  --jsonrpc-apis all --mode last  --base-path=/home/parity/.local/share/io.parity.ethereum/

  docker run --restart=always --net inc_net -p 9334:9334 -p 9433:9433 -e BOOTNODE_IP=$bootnode -e GETH_NAME=inc_kovan -e MININGKEY=${validator_key} -e TESTNET=true -v $PWD/${data_dir}:/data -d --name inc_miner incognitochaintestnet/incognito:${latest_tag}
}

# kill existing run.sh processes
ps aux | grep '[r]un.sh' | awk '{ print $2}' | grep -v "^$$\$" | xargs kill -9

current_latest_tag=""
while [ 1 = 1 ]
do
  tags=`curl -X GET https://registry.hub.docker.com/v1/repositories/incognitochaintestnet/incognito/tags  | sed -e 's/[][]//g' -e 's/"//g' -e 's/ //g' | tr '}' '\n'  | awk -F: '{print $3}' | sed -e 's/\n/;/g'`

  sorted_tags=($(echo ${tags[*]}| tr " " "\n" | sort -rn))
  latest_tag=${sorted_tags[0]}

  if [ "$current_latest_tag" != "$latest_tag" ]
  then
    run $latest_tag $current_latest_tag
    current_latest_tag=$latest_tag
  fi

  sleep 3600s

done &
