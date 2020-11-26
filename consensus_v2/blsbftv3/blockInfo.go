package blsbftv3

import (
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type ProposeBlockInfo struct {
	block      types.BlockInterface
	committees []incognitokey.CommitteePublicKey
	votes      map[string]BFTVote //pk->BFTVote
	isValid    bool
	hasNewVote bool
	isVoted    bool
}

//NewProposeBlockInfoValue : new propose block info
func newProposeBlockForProposeMsg(
	block types.BlockInterface,
	committees []incognitokey.CommitteePublicKey,
	votes map[string]BFTVote,
	isValid, hasNewVote bool,
) *ProposeBlockInfo {
	return &ProposeBlockInfo{
		block:      block,
		committees: incognitokey.DeepCopy(committees),
		votes:      votes,
		isValid:    isValid,
		hasNewVote: hasNewVote,
	}
}

func (proposeBlockInfo *ProposeBlockInfo) addBlockInfo(
	block types.BlockInterface,
	committees []incognitokey.CommitteePublicKey) {
	proposeBlockInfo.block = block
	proposeBlockInfo.committees = incognitokey.DeepCopy(committees)
}

func newBlockInfoForVoteMsg() *ProposeBlockInfo {
	return &ProposeBlockInfo{
		votes:      make(map[string]BFTVote),
		hasNewVote: true,
	}
}
