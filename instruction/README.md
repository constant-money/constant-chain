# Instruction 
## Stake
  - Stake Shard
  ```["stake", "pubkey1,pubkey2,..." "shard" "txStake1,txStake2,..." "rewardReceiver1,rewardReceiver2,..." "flag1,flag2,..."]```
  
  ```["stake", "pubkey1,pubkey2,..." "beacon" "txStake1,txStake2,..." "rewardReceiver1,rewardReceiver2,..." "flag1,flag2,..."]```

## Swap
  - Normal case:
    ```["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "shard" "{shardID}" "punishedPubkey1,..."] ```
    
    ```["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "beacon" "punishedPubkey1,..."] ```
  - Replace case:
    ```["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "beacon" "" "punishedPubkey1,..." "newRewardReceiver1,..."] ```
    
    ```["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "shard" "{shardID}" "punishedPubkey1,..." "newRewardReceiver1,..."] ```

## Assign
  ```["assign" "shardCandidate1,shardCandidate2,..." "shard" "{shardID}"]```
  
## Stop auto stake
  ```["stopautostake" "pubkey1,pubkey2,..."]```
  
## Random 
  ```["random" "{nonce}" "{blockheight}" "{timestamp}" "{bitcoinTimestamp}"]```
  
## Request Shard Swap
    ```["request_shard_swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "{shardID}" "epoch" "RandomNumber"] ```
## Confirm Shard Swap    
    ```["confirm_shard_swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "{shardID}" "epoch" "RandomNumber"] ```