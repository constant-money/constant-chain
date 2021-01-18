package tx_ver1

import (
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/pkg/errors"
)

func (tx *Tx) LoadCommitment(txDB *statedb.StateDB) error {
	return errors.Errorf("Implement pls")
}

func (tx *Tx) ValidateTxCorrectness() (bool, error) {
	return false, errors.Errorf("Implement pls")
}

func (tx *Tx) VerifySigTx() (bool, error) {
	return false, errors.Errorf("Implement pls")
}

func (tx *Tx) ValidateSanityDataByItSelf() (bool, error) {
	return false, errors.Errorf("Implement pls")
}

func (tx *Tx) ValidateSanityDataWithBlockchain(
	chainRetriever metadata.ChainRetriever,
	shardViewRetriever metadata.ShardViewRetriever,
	beaconViewRetriever metadata.BeaconViewRetriever,
	beaconHeight uint64,
) (bool, error) {
	return false, errors.Errorf("Implement pls")
}

func (tx *Tx) ValidateDoubleSpendWithBlockChain(stateDB *statedb.StateDB) (bool, error) {
	return false, errors.Errorf("Implement pls")
}

func (tx *Tx) GetValidationEnv() metadata.ValidationEnviroment {
	panic("err")
	return nil
}
func (tx *Tx) SetValidationEnv(metadata.ValidationEnviroment) {
	return
}
