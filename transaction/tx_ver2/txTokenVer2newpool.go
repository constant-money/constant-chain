package tx_ver2

import (
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/pkg/errors"
)

func (tx *TxToken) LoadCommitment(txDB *statedb.StateDB) error {
	return errors.Errorf("Implement pls")
}

func (tx *TxToken) ValidateTxCorrectness() (bool, error) {
	return false, errors.Errorf("Implement pls")
}

func (tx *TxToken) VerifySigTx() (bool, error) {
	return false, errors.Errorf("Implement pls")
}

func (tx *TxToken) ValidateSanityDataByItSelf() (bool, error) {
	return false, errors.Errorf("Implement pls")
}

func (tx *TxToken) ValidateSanityDataWithBlockchain(
	chainRetriever metadata.ChainRetriever,
	shardViewRetriever metadata.ShardViewRetriever,
	beaconViewRetriever metadata.BeaconViewRetriever,
	beaconHeight uint64,
) (bool, error) {
	return false, errors.Errorf("Implement pls")
}

func (tx *TxToken) ValidateDoubleSpendWithBlockChain(stateDB *statedb.StateDB) (bool, error) {
	return false, errors.Errorf("Implement pls")
}

func (tx *TxToken) GetValidationEnv() metadata.ValidationEnviroment {
	panic("err")
	return nil
}
func (tx *TxToken) SetValidationEnv(metadata.ValidationEnviroment) {
	return
}
