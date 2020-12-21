package blockchain

import "github.com/incognitochain/incognito-chain/common"

func (blockchain *BlockChain) GetStakingAmountShard() uint64 {
	return blockchain.config.ChainParams.StakingAmountShard
}

func (blockchain *BlockChain) GetCentralizedWebsitePaymentAddress(beaconHeight uint64) string {
	if blockchain.config.ChainParams.Net == Testnet || blockchain.config.ChainParams.Net == Testnet2 {
		return blockchain.config.ChainParams.CentralizedWebsitePaymentAddress
	}
	if blockchain.config.ChainParams.Net == Mainnet {
		if beaconHeight >= 677000 {
			// use new address
			return "12RwAheAvvMqrpxviWCV5r6JLS2puiMom3fy6GUCAmPNN1BXnEW4DXpueqMfV66zyAMpurEuegWPGV6U4HR6Mi9KzUiDL4K3uyv1xxF"
		} else if beaconHeight >= 243500 {
			// use new address
			return "12S6jZ6sjJaqsuMJKS6jG7gvE9eHUXGWa2B2dNC7PwyEYJkL6cE53Uzk926HrQMEv2i2oBvKP2GDTC6tzU9dYSVH5X3w9P58VWqux4F"
		} else {
			// use original address
			return blockchain.config.ChainParams.CentralizedWebsitePaymentAddress
		}
	}
	return ""
}

func (blockchain *BlockChain) GetBeaconHeightBreakPointBurnAddr() uint64 {
	return blockchain.config.ChainParams.BeaconHeightBreakPointBurnAddr
}

func (blockchain *BlockChain) GetETHRemoveBridgeSigEpoch() uint64 {
	return blockchain.config.ChainParams.ETHRemoveBridgeSigEpoch
}

func (blockchain *BlockChain) GetBurningAddress(beaconHeight uint64) string {
	breakPoint := blockchain.GetBeaconHeightBreakPointBurnAddr()
	if beaconHeight == 0 {
		beaconHeight = blockchain.BeaconChain.GetFinalViewHeight()
	}

	if beaconHeight <= breakPoint {
		return common.BurningAddress
	}

	return common.BurningAddress2
}

