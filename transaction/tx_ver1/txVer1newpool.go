package tx_ver1

import (
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	"github.com/pkg/errors"
)

func (tx *Tx) LoadCommitment(txDB *statedb.StateDB) error {
	prf := tx.GetProof()
	if prf == nil {
		return errors.Errorf("Nani?")
	}
	tokenID := tx.GetValidationEnv().TokenID()
	shardID := byte(tx.GetValidationEnv().ShardID())
	prfV1, ok := prf.(*privacy.ProofV1)
	if !ok {
		return utils.NewTransactionErr(utils.RejectTxVersion, errors.New("Wrong proof version"))
	}
	oneOfManyProof := prfV1.GetOneOfManyProof()
	commitmentIndices := prfV1.GetCommitmentIndices()
	commitmentInputSND := prfV1.GetCommitmentInputSND()
	commitmentInputValue := prfV1.GetCommitmentInputValue()
	commitmentInputShardID := prfV1.GetCommitmentInputShardID()
	commitmentInputSecretKey := prfV1.GetCommitmentInputSecretKey()

	commitmentsRing := make([][privacy.CommitmentRingSize]*privacy.Point, len(oneOfManyProof))
	sumSKeySID := new(privacy.Point).Add(commitmentInputShardID, commitmentInputSecretKey)

	for i, commitments := range commitmentsRing {
		cmInputSum := new(privacy.Point).Add(commitmentInputSND[i], commitmentInputValue[i])
		cmInputSum.Add(cmInputSum, sumSKeySID)
		for j, commitment := range commitments {
			index := commitmentIndices[i*privacy.CommitmentRingSize+j]
			commitmentBytes, err := statedb.GetCommitmentByIndex(txDB, *tokenID, index, shardID)
			if err != nil {
				utils.Logger.Log.Errorf("GetCommitmentInDatabase: Error when getCommitmentByIndex from database", index, err)
				return utils.NewTransactionErr(utils.GetCommitmentsInDatabaseError, err)
			}
			commitment, err = new(privacy.Point).FromBytesS(commitmentBytes)
			if err != nil {
				utils.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: Cannot decompress commitment from database", index, err)
				return errhandler.NewPrivacyErr(utils.VerifyOneOutOfManyProofFailedErr, err)
			}
			commitment.Sub(commitment, cmInputSum)
		}
	}
	// return &commitments, nil
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
