package zkp

// import (
// 	// "github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
// 	// "github.com/incognitochain/incognito-chain/metadata"
// 	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
// 	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
// 	"github.com/incognitochain/incognito-chain/privacy/operation"
// 	"github.com/incognitochain/incognito-chain/privacy/privacy_util"
// 	"github.com/pkg/errors"
// )

// func (prf *PaymentProof) LoadDataFromDB(txDB *statedb.StateDB) error {
// 	tokenID := txEnv.TokenID()
// 	shardID := byte(txEnv.ShardID())
// 	oneOfManyProof := prf.GetOneOfManyProof()
// 	commitmentIndices := prf.GetCommitmentIndices()
// 	commitmentInputSND := prf.GetCommitmentInputSND()
// 	commitmentInputValue := prf.GetCommitmentInputValue()
// 	commitmentInputShardID := prf.GetCommitmentInputShardID()
// 	commitmentInputSecretKey := prf.GetCommitmentInputSecretKey()

// 	commitmentsRing := make([][privacy_util.CommitmentRingSize]*operation.Point, len(oneOfManyProof))
// 	sumSKeySID := new(operation.Point).Add(commitmentInputShardID, commitmentInputSecretKey)

// 	for i, commitments := range commitmentsRing {
// 		cmInputSum := new(operation.Point).Add(commitmentInputSND[i], commitmentInputValue[i])
// 		cmInputSum.Add(cmInputSum, sumSKeySID)
// 		for j, commitment := range commitments {
// 			index := commitmentIndices[i*privacy_util.CommitmentRingSize+j]
// 			commitmentBytes, err := statedb.GetCommitmentByIndex(txDB, *tokenID, index, shardID)
// 			if err != nil {
// 				errors.Errorf("GetCommitmentInDatabase: Error when getCommitmentByIndex from database", index, err)
// 				return err
// 			}
// 			commitment, err = new(operation.Point).FromBytesS(commitmentBytes)
// 			if err != nil {
// 				errors..Errorf("VERIFICATION PAYMENT PROOF: Cannot decompress commitment from database", index, err)
// 				return errhandler.NewPrivacyErr(errhandler.VerifyOneOutOfManyProofFailedErr, err)
// 			}
// 			commitment.Sub(commitment, cmInputSum)
// 		}
// 	}

// 	return errors.Errorf("Implement pls")
// }
