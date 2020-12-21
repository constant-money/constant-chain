#!/usr/bin/env bash

SALT=testnet-salt OLD_NODES=../deploy/group_vars/deploy_temp.yml NEW_NODES=./newnodes.yml OLD_PUBKEYS=../../keylist.json NEW_PUBKEYS=./keylist-v2.json ./regenvalidatorkey
