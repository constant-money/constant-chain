package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	pCommon "github.com/incognitochain/incognito-chain/portal/common"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/portalv4"
	pv4Common "github.com/incognitochain/incognito-chain/portalv4/common"
	pv4Meta "github.com/incognitochain/incognito-chain/portalv4/metadata"
	portalprocessv4 "github.com/incognitochain/incognito-chain/portalv4/portalprocess"
	portalTokensV4 "github.com/incognitochain/incognito-chain/portalv4/portaltokens"
	"github.com/stretchr/testify/suite"
)

var _ = func() (_ struct{}) {
	Logger.Init(common.NewBackend(nil).Logger("test", true))
	portalprocessv4.Logger.Init(common.NewBackend(nil).Logger("test", true))
	portalTokensV4.Logger.Init(common.NewBackend(nil).Logger("test", true))
	pv4Meta.Logger.Init(common.NewBackend(nil).Logger("test", true))
	//pv4Common.Logger.Init(common.NewBackend(nil).Logger("test", true))
	Logger.log.Info("This runs before init()!")
	return
}()

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type PortalTestSuiteV4 struct {
	suite.Suite
	currentPortalStateForProducer portalprocessv4.CurrentPortalV4State
	currentPortalStateForProcess  portalprocessv4.CurrentPortalV4State

	sdb          *statedb.StateDB
	portalParams portalv4.PortalParams
	blockChain   *BlockChain
}

const USER_BTC_ADDRESS_1 = "12ok3D39W4AZj4aF2rmgzqys3BB4uhcXVN"

func (s *PortalTestSuiteV4) SetupTest() {
	dbPath, err := ioutil.TempDir(os.TempDir(), "portal_test_statedb_")
	if err != nil {
		panic(err)
	}
	diskBD, _ := incdb.Open("leveldb", dbPath)
	warperDBStatedbTest := statedb.NewDatabaseAccessWarper(diskBD)
	emptyRoot := common.HexToHash(common.HexEmptyRoot)
	stateDB, _ := statedb.NewWithPrefixTrie(emptyRoot, warperDBStatedbTest)

	s.sdb = stateDB

	s.currentPortalStateForProducer = portalprocessv4.CurrentPortalV4State{
		WaitingUnshieldRequests:   map[string]map[string]*statedb.WaitingUnshieldRequest{},
		UTXOs:                     map[string]map[string]*statedb.UTXO{},
		ProcessedUnshieldRequests: map[string]map[string]*statedb.ProcessedUnshieldRequestBatch{},
		ShieldingExternalTx:       map[string]map[string]*statedb.ShieldingRequest{},
	}
	s.currentPortalStateForProcess = portalprocessv4.CurrentPortalV4State{
		WaitingUnshieldRequests:   map[string]map[string]*statedb.WaitingUnshieldRequest{},
		UTXOs:                     map[string]map[string]*statedb.UTXO{},
		ProcessedUnshieldRequests: map[string]map[string]*statedb.ProcessedUnshieldRequestBatch{},
		ShieldingExternalTx:       map[string]map[string]*statedb.ShieldingRequest{},
	}
	s.portalParams = portalv4.PortalParams{
		MultiSigAddresses: map[string]string{
			pv4Common.PortalBTCIDStr: "2MvpFqydTR43TT4emMD84Mzhgd8F6dCow1X",
		},
		MultiSigScriptHexEncode: map[string]string{
			pv4Common.PortalBTCIDStr: "",
		},
		PortalTokens: map[string]portalTokensV4.PortalTokenProcessor{
			pv4Common.PortalBTCIDStr: &portalTokensV4.PortalBTCTokenProcessor{
				&portalTokensV4.PortalToken{
					ChainID:        "Bitcoin-Testnet",
					MinTokenAmount: 10,
				},
			},
		},
		FeeUnshields: map[string]uint64{
			pv4Common.PortalBTCIDStr: 100000, // in nano pBTC - 10000 satoshi ~ 4 usd
		},
		BatchNumBlks:               45,
		PortalReplacementAddress:   "",
		MaxFeeForEachStep:          500,
		TimeSpaceForFeeReplacement: 2 * time.Minute,
	}
	s.blockChain = &BlockChain{
		config: Config{
			ChainParams: &Params{
				MinBeaconBlockInterval: 40 * time.Second,
				MinShardBlockInterval:  40 * time.Second,
				Epoch:                  100,
				PortalV4Params: map[uint64]portalv4.PortalParams{
					0: s.portalParams,
				},
			},
		},
	}
}

type portalV4InstForProducer struct {
	inst         []string
	optionalData map[string]interface{}
}

func producerPortalInstructionsV4(
	blockchain basemeta.ChainRetriever,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	insts []portalV4InstForProducer,
	currentPortalState *portalprocessv4.CurrentPortalV4State,
	portalParams portalv4.PortalParams,
	shardID byte,
	pm *portalprocessv4.PortalV4Manager,
) ([][]string, error) {
	var newInsts [][]string

	for _, item := range insts {
		inst := item.inst
		optionalData := item.optionalData

		metaType, _ := strconv.Atoi(inst[0])
		contentStr := inst[1]
		portalProcessor := pm.PortalInstructions[metaType]
		newInst, err := portalProcessor.BuildNewInsts(
			blockchain,
			contentStr,
			shardID,
			currentPortalState,
			beaconHeight,
			shardHeights,
			portalParams,
			optionalData,
		)
		if err != nil {
			Logger.log.Error(err)
			return newInsts, err
		}

		newInsts = append(newInsts, newInst...)
	}

	return newInsts, nil
}

func processPortalInstructionsV4(
	blockchain basemeta.ChainRetriever,
	beaconHeight uint64,
	insts [][]string,
	portalStateDB *statedb.StateDB,
	currentPortalState *portalprocessv4.CurrentPortalV4State,
	portalParams portalv4.PortalParams,
	pm *portalprocessv4.PortalV4Manager,
) error {
	updatingInfoByTokenID := map[common.Hash]basemeta.UpdatingInfo{}
	for _, inst := range insts {
		if len(inst) < 4 {
			continue // Not error, just not Portal instruction
		}

		var err error
		metaType, _ := strconv.Atoi(inst[0])
		processor := pm.PortalInstructions[metaType]
		if processor != nil {
			err = processor.ProcessInsts(portalStateDB, beaconHeight, inst, currentPortalState, portalParams, updatingInfoByTokenID)
			if err != nil {
				Logger.log.Errorf("Process portal instruction err: %v, inst %+v", err, inst)
			}
			continue
		}
	}
	// update info of bridge portal token
	for _, updatingInfo := range updatingInfoByTokenID {
		var updatingAmt uint64
		var updatingType string
		if updatingInfo.CountUpAmt > updatingInfo.DeductAmt {
			updatingAmt = updatingInfo.CountUpAmt - updatingInfo.DeductAmt
			updatingType = "+"
		}
		if updatingInfo.CountUpAmt < updatingInfo.DeductAmt {
			updatingAmt = updatingInfo.DeductAmt - updatingInfo.CountUpAmt
			updatingType = "-"
		}
		err := statedb.UpdateBridgeTokenInfo(
			portalStateDB,
			updatingInfo.TokenID,
			updatingInfo.ExternalTokenID,
			updatingInfo.IsCentralized,
			updatingAmt,
			updatingType,
		)
		if err != nil {
			return err
		}
	}

	// store updated currentPortalState to leveldb with new beacon height
	err := portalprocessv4.StorePortalV4StateToDB(portalStateDB, currentPortalState)
	if err != nil {
		Logger.log.Error(err)
	}

	return nil
}

/*
	Shielding Request
*/
type TestCaseShieldingRequest struct {
	tokenID                  string
	incAddressStr            string
	shieldingProof           string
	txID                     string
	isExistsInPreviousBlocks bool
}

type ExpectedResultShieldingRequest struct {
	utxos          map[string]map[string]*statedb.UTXO
	numBeaconInsts uint
	statusInsts    []string
}

func (s *PortalTestSuiteV4) SetupTestShieldingRequest() {
	// do nothing
}

func generateUTXOKeyAndValue(tokenID string, walletAddress string, txHash string, outputIdx uint32, outputAmount uint64) (string, *statedb.UTXO) {
	utxoKey := statedb.GenerateUTXOObjectKey(pv4Common.PortalBTCIDStr, walletAddress, txHash, outputIdx).String()
	utxoValue := statedb.NewUTXOWithValue(walletAddress, txHash, outputIdx, outputAmount)
	return utxoKey, utxoValue
}

func buildTestCaseAndExpectedResultShieldingRequest() ([]TestCaseShieldingRequest, *ExpectedResultShieldingRequest) {
	// build test cases
	testcases := []TestCaseShieldingRequest{
		// valid shielding request
		{
			tokenID:                  pv4Common.PortalBTCIDStr,
			incAddressStr:            "12S5Lrs1XeQLbqN4ySyKtjAjd2d7sBP2tjFijzmp6avrrkQCNFMpkXm3FPzj2Wcu2ZNqJEmh9JriVuRErVwhuQnLmWSaggobEWsBEci",
			shieldingProof:           "eyJNZXJrbGVQcm9vZnMiOlt7IlByb29mSGFzaCI6WzM2LDgyLDI4LDUyLDgzLDE3OCwxNywxMzgsMjA1LDkyLDIyNCw4Myw2Myw2MSwxOTEsNTMsMTQ4LDIyNyw4OSwxODcsMjA0LDE3MSwxOCw3OSwyOCwxNDUsMzksMCwyMjMsMjksMTcwLDk3XSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzE5NiwxMSwxMiwyNTQsMTg1LDE0NCwxNTgsODgsMTU4LDI1LDQ2LDkxLDE2NCwyMjAsMTA4LDE2NSwxNzcsMjE5LDE5NSwyMDEsMjQ1LDE4OSw1LDQsMTIzLDE2OCw3NSwxNjcsMTQzLDE0NywyNDIsMjMyXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxMzgsNSwzOSw3NCwyNCw3NSw4MSw2MCwxNjcsNDYsMTg2LDEwNiwxNTAsNDQsMjAwLDIxLDIzOCw0MSwyMzQsMzksMjI1LDkyLDExLDIzNCwxNDAsMTA3LDI0OCwyNDQsMTQ0LDExNiwyMTksMTM2XSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzE4MiwxMDYsOTEsMTYxLDE0NSwxMzMsMjQ2LDc1LDIwOSw3NCwxODEsMTgyLDkyLDI1NCw0OSwxOTMsNTEsMjMzLDE1NywxODUsNTQsNzMsNTAsMjQ0LDEwNywzMiwzMSwxODksNDMsNCwxMTIsMTI4XSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxMiwzNSwxODIsMTk3LDE5NiwxODYsMTQzLDE1MSw0MywxMDMsMjU1LDE2LDE2MSwyNDAsMTM5LDE2OCwxNzEsOTgsODYsMTA3LDk3LDIxMiw5MCwxNjUsMTQ5LDYyLDMwLDY1LDc1LDIyOCw2NywxODFdLCJJc0xlZnQiOnRydWV9LHsiUHJvb2ZIYXNoIjpbNDcsMTY0LDYwLDcsODAsMTQsNzQsMTY1LDE5NSwxODYsMTE2LDY0LDExOCwxMDIsMTk1LDEsMTMxLDQ0LDU5LDE3MSwyMDEsMTU3LDc2LDUzLDgyLDksMTM4LDE3OSw5MSw2LDQsNDRdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzIzNSwyNDEsMTY4LDM0LDE4MywxNzgsNDEsMjUzLDEwMiwxMzYsMTg2LDg3LDE4OCwyMzQsMzgsMTU4LDExMSwyMjUsMTIyLDIzMCwyMjksNDgsMTgyLDEwNiwyNyw2NSwyMTQsNDIsMTUzLDQyLDMwLDkwXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOls2LDE4NCwyOCwyNDgsMTcyLDM0LDE0MywyNTEsMTcwLDEzLDIxNyw3OSwyMjcsMTA2LDIxMiw1NSw5MSwyMDMsMTAzLDkwLDkwLDIyLDI0Niw2NSw0OCwyMTMsMjU1LDE5OSwzOCwxMTMsMTkxLDIxXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsyOCwxOTQsMTQyLDMzLDQzLDg3LDIxLDIzNCwxOSwxOTEsMTYzLDIxMiwyMTcsMjUsNDksMTk5LDIwMywxNzIsMjUsNywxNjEsMTM2LDE2MywzMyw3OSwxODcsNDQsNzEsMTAxLDI5LDE4NSwyNTNdLCJJc0xlZnQiOmZhbHNlfV0sIkJUQ1R4Ijp7IlZlcnNpb24iOjEsIlR4SW4iOlt7IlByZXZpb3VzT3V0UG9pbnQiOnsiSGFzaCI6WzExLDMwLDIxNiwzLDUzLDE3MSwyNTUsMTcsMTEwLDE4NCwxMjYsMTEyLDE2LDE0MSwyMzgsMjIwLDE5NiwxMzEsMTI0LDE5Nyw5MywxOTcsMjAwLDIxMSwxMDIsMTM1LDQ4LDQ0LDEwLDIzMCw1MCwxMjJdLCJJbmRleCI6MH0sIlNpZ25hdHVyZVNjcmlwdCI6IlJ6QkVBaUJHajhPZnBtYzQzRHZyVm5icTlDa0ZhcnBDUzBIWWtUZUNMVUp6YW40UEhRSWdJckxYUUV2WXV4cHJEV3VEemxGQlV6cXlVQXNzOHplc1NIL2JwUVRLRW5jQklRUFBJQk5QVmtqaTl2RjNwbFVySmJxWDVGc1VoVTVQRXlwSzhQNTlxVC9FL0E9PSIsIldpdG5lc3MiOm51bGwsIlNlcXVlbmNlIjo0Mjk0OTY3Mjk1fV0sIlR4T3V0IjpbeyJWYWx1ZSI6MCwiUGtTY3JpcHQiOiJhaXhTVFVwUGJDdG9NMDVGYmxJME5uUmhORkpFZEN0MFRITm1XSEZLYmprMGFFWkllQzlrVG5aMlNWQlJQUT09In0seyJWYWx1ZSI6MjAwLCJQa1NjcmlwdCI6InFSUW5KNmR2OHZvNVhjVWxaS2pyS3F1TS9sSElkb2M9In0seyJWYWx1ZSI6MjI5MjcyLCJQa1NjcmlwdCI6ImRxa1Vndnk2bFFpK0VpUXk5N3VRMmw5MEFVQ21XNGlJckE9PSJ9XSwiTG9ja1RpbWUiOjB9LCJCbG9ja0hhc2giOlsxNzIsMjM2LDE2OCwxMDUsMTM0LDMzLDEzNSwxMzIsMTIsMjI1LDEyMywyMTIsMzksMjQ1LDE1LDE5MywxNDYsMTIzLDEwNSwxMTIsMzYsMTgwLDE4MiwxMDUsNDIsMjA3LDExNSwyMTgsMCwwLDAsMF19",
			txID:                     common.HashH([]byte{1}).String(),
			isExistsInPreviousBlocks: false,
		},
		// valid shielding request
		{
			tokenID:                  pv4Common.PortalBTCIDStr,
			incAddressStr:            "12S5Lrs1XeQLbqN4ySyKtjAjd2d7sBP2tjFijzmp6avrrkQCNFMpkXm3FPzj2Wcu2ZNqJEmh9JriVuRErVwhuQnLmWSaggobEWsBEci",
			shieldingProof:           "eyJNZXJrbGVQcm9vZnMiOlt7IlByb29mSGFzaCI6WzE0MywyMTEsMjI2LDExNiwyNTMsNjksMjQ2LDIyNCwxMTAsMTg0LDMwLDE1Nyw4NCwyMDcsMTQyLDI1MywxMjIsNTAsMTk0LDgsMjAzLDExOSw3NSwxODMsMjUsNjUsMTU1LDIxMywxODYsMTg0LDEyNSwxMF0sIklzTGVmdCI6dHJ1ZX0seyJQcm9vZkhhc2giOls0OCwxODIsMTE2LDI1MCwzOSwxMDgsMTk1LDE0NCwyMSw3OSwyMjIsNzQsMTk3LDE2MSwxMDcsMTYwLDIxLDMwLDIwNiwyNDksMTc5LDExMSwyMjMsMzIsNDcsMTM5LDE1MywyOCwxOTIsMjIwLDE0NiwyNV0sIklzTGVmdCI6dHJ1ZX0seyJQcm9vZkhhc2giOlsxMzgsNSwzOSw3NCwyNCw3NSw4MSw2MCwxNjcsNDYsMTg2LDEwNiwxNTAsNDQsMjAwLDIxLDIzOCw0MSwyMzQsMzksMjI1LDkyLDExLDIzNCwxNDAsMTA3LDI0OCwyNDQsMTQ0LDExNiwyMTksMTM2XSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzE4MiwxMDYsOTEsMTYxLDE0NSwxMzMsMjQ2LDc1LDIwOSw3NCwxODEsMTgyLDkyLDI1NCw0OSwxOTMsNTEsMjMzLDE1NywxODUsNTQsNzMsNTAsMjQ0LDEwNywzMiwzMSwxODksNDMsNCwxMTIsMTI4XSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxMiwzNSwxODIsMTk3LDE5NiwxODYsMTQzLDE1MSw0MywxMDMsMjU1LDE2LDE2MSwyNDAsMTM5LDE2OCwxNzEsOTgsODYsMTA3LDk3LDIxMiw5MCwxNjUsMTQ5LDYyLDMwLDY1LDc1LDIyOCw2NywxODFdLCJJc0xlZnQiOnRydWV9LHsiUHJvb2ZIYXNoIjpbNDcsMTY0LDYwLDcsODAsMTQsNzQsMTY1LDE5NSwxODYsMTE2LDY0LDExOCwxMDIsMTk1LDEsMTMxLDQ0LDU5LDE3MSwyMDEsMTU3LDc2LDUzLDgyLDksMTM4LDE3OSw5MSw2LDQsNDRdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzIzNSwyNDEsMTY4LDM0LDE4MywxNzgsNDEsMjUzLDEwMiwxMzYsMTg2LDg3LDE4OCwyMzQsMzgsMTU4LDExMSwyMjUsMTIyLDIzMCwyMjksNDgsMTgyLDEwNiwyNyw2NSwyMTQsNDIsMTUzLDQyLDMwLDkwXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOls2LDE4NCwyOCwyNDgsMTcyLDM0LDE0MywyNTEsMTcwLDEzLDIxNyw3OSwyMjcsMTA2LDIxMiw1NSw5MSwyMDMsMTAzLDkwLDkwLDIyLDI0Niw2NSw0OCwyMTMsMjU1LDE5OSwzOCwxMTMsMTkxLDIxXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsyOCwxOTQsMTQyLDMzLDQzLDg3LDIxLDIzNCwxOSwxOTEsMTYzLDIxMiwyMTcsMjUsNDksMTk5LDIwMywxNzIsMjUsNywxNjEsMTM2LDE2MywzMyw3OSwxODcsNDQsNzEsMTAxLDI5LDE4NSwyNTNdLCJJc0xlZnQiOmZhbHNlfV0sIkJUQ1R4Ijp7IlZlcnNpb24iOjEsIlR4SW4iOlt7IlByZXZpb3VzT3V0UG9pbnQiOnsiSGFzaCI6WzE0MywyMTEsMjI2LDExNiwyNTMsNjksMjQ2LDIyNCwxMTAsMTg0LDMwLDE1Nyw4NCwyMDcsMTQyLDI1MywxMjIsNTAsMTk0LDgsMjAzLDExOSw3NSwxODMsMjUsNjUsMTU1LDIxMywxODYsMTg0LDEyNSwxMF0sIkluZGV4IjoyfSwiU2lnbmF0dXJlU2NyaXB0IjoiU0RCRkFpRUE5WS9XeDZvMDh4QjAzZkdya3EyVGQ5NXV5akxiK0ZTRk13cHpWcHdkZTFNQ0lGcWJzdWlWeis5Wnhka05YWmZXQ1p5WHZMdUJrK3Y1KzZzYk1kbGUwSVkvQVNFRHp5QVRUMVpJNHZieGQ2WlZLeVc2bCtSYkZJVk9UeE1xU3ZEK2Zhay94UHc9IiwiV2l0bmVzcyI6bnVsbCwiU2VxdWVuY2UiOjQyOTQ5NjcyOTV9XSwiVHhPdXQiOlt7IlZhbHVlIjowLCJQa1NjcmlwdCI6ImFpeFNUVXBQYkN0b00wNUZibEkwTm5SaE5GSkVkQ3QwVEhObVdIRktiamswYUVaSWVDOWtUbloyU1ZCUlBRPT0ifSx7IlZhbHVlIjo2MDAsIlBrU2NyaXB0IjoicVJRbko2ZHY4dm81WGNVbFpLanJLcXVNL2xISWRvYz0ifSx7IlZhbHVlIjoyMjQzNzIsIlBrU2NyaXB0IjoiZHFrVWd2eTZsUWkrRWlReTk3dVEybDkwQVVDbVc0aUlyQT09In1dLCJMb2NrVGltZSI6MH0sIkJsb2NrSGFzaCI6WzE3MiwyMzYsMTY4LDEwNSwxMzQsMzMsMTM1LDEzMiwxMiwyMjUsMTIzLDIxMiwzOSwyNDUsMTUsMTkzLDE0NiwxMjMsMTA1LDExMiwzNiwxODAsMTgyLDEwNSw0MiwyMDcsMTE1LDIxOCwwLDAsMCwwXX0=",
			txID:                     common.HashH([]byte{2}).String(),
			isExistsInPreviousBlocks: false,
		},
		// invalid shielding request: duplicated shielding proof in previous blocks
		{
			tokenID:                  pv4Common.PortalBTCIDStr,
			incAddressStr:            "12S5Lrs1XeQLbqN4ySyKtjAjd2d7sBP2tjFijzmp6avrrkQCNFMpkXm3FPzj2Wcu2ZNqJEmh9JriVuRErVwhuQnLmWSaggobEWsBEci",
			shieldingProof:           "eyJNZXJrbGVQcm9vZnMiOlt7IlByb29mSGFzaCI6WzM2LDgyLDI4LDUyLDgzLDE3OCwxNywxMzgsMjA1LDkyLDIyNCw4Myw2Myw2MSwxOTEsNTMsMTQ4LDIyNyw4OSwxODcsMjA0LDE3MSwxOCw3OSwyOCwxNDUsMzksMCwyMjMsMjksMTcwLDk3XSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzE5NiwxMSwxMiwyNTQsMTg1LDE0NCwxNTgsODgsMTU4LDI1LDQ2LDkxLDE2NCwyMjAsMTA4LDE2NSwxNzcsMjE5LDE5NSwyMDEsMjQ1LDE4OSw1LDQsMTIzLDE2OCw3NSwxNjcsMTQzLDE0NywyNDIsMjMyXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxMzgsNSwzOSw3NCwyNCw3NSw4MSw2MCwxNjcsNDYsMTg2LDEwNiwxNTAsNDQsMjAwLDIxLDIzOCw0MSwyMzQsMzksMjI1LDkyLDExLDIzNCwxNDAsMTA3LDI0OCwyNDQsMTQ0LDExNiwyMTksMTM2XSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzE4MiwxMDYsOTEsMTYxLDE0NSwxMzMsMjQ2LDc1LDIwOSw3NCwxODEsMTgyLDkyLDI1NCw0OSwxOTMsNTEsMjMzLDE1NywxODUsNTQsNzMsNTAsMjQ0LDEwNywzMiwzMSwxODksNDMsNCwxMTIsMTI4XSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxMiwzNSwxODIsMTk3LDE5NiwxODYsMTQzLDE1MSw0MywxMDMsMjU1LDE2LDE2MSwyNDAsMTM5LDE2OCwxNzEsOTgsODYsMTA3LDk3LDIxMiw5MCwxNjUsMTQ5LDYyLDMwLDY1LDc1LDIyOCw2NywxODFdLCJJc0xlZnQiOnRydWV9LHsiUHJvb2ZIYXNoIjpbNDcsMTY0LDYwLDcsODAsMTQsNzQsMTY1LDE5NSwxODYsMTE2LDY0LDExOCwxMDIsMTk1LDEsMTMxLDQ0LDU5LDE3MSwyMDEsMTU3LDc2LDUzLDgyLDksMTM4LDE3OSw5MSw2LDQsNDRdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzIzNSwyNDEsMTY4LDM0LDE4MywxNzgsNDEsMjUzLDEwMiwxMzYsMTg2LDg3LDE4OCwyMzQsMzgsMTU4LDExMSwyMjUsMTIyLDIzMCwyMjksNDgsMTgyLDEwNiwyNyw2NSwyMTQsNDIsMTUzLDQyLDMwLDkwXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOls2LDE4NCwyOCwyNDgsMTcyLDM0LDE0MywyNTEsMTcwLDEzLDIxNyw3OSwyMjcsMTA2LDIxMiw1NSw5MSwyMDMsMTAzLDkwLDkwLDIyLDI0Niw2NSw0OCwyMTMsMjU1LDE5OSwzOCwxMTMsMTkxLDIxXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsyOCwxOTQsMTQyLDMzLDQzLDg3LDIxLDIzNCwxOSwxOTEsMTYzLDIxMiwyMTcsMjUsNDksMTk5LDIwMywxNzIsMjUsNywxNjEsMTM2LDE2MywzMyw3OSwxODcsNDQsNzEsMTAxLDI5LDE4NSwyNTNdLCJJc0xlZnQiOmZhbHNlfV0sIkJUQ1R4Ijp7IlZlcnNpb24iOjEsIlR4SW4iOlt7IlByZXZpb3VzT3V0UG9pbnQiOnsiSGFzaCI6WzExLDMwLDIxNiwzLDUzLDE3MSwyNTUsMTcsMTEwLDE4NCwxMjYsMTEyLDE2LDE0MSwyMzgsMjIwLDE5NiwxMzEsMTI0LDE5Nyw5MywxOTcsMjAwLDIxMSwxMDIsMTM1LDQ4LDQ0LDEwLDIzMCw1MCwxMjJdLCJJbmRleCI6MH0sIlNpZ25hdHVyZVNjcmlwdCI6IlJ6QkVBaUJHajhPZnBtYzQzRHZyVm5icTlDa0ZhcnBDUzBIWWtUZUNMVUp6YW40UEhRSWdJckxYUUV2WXV4cHJEV3VEemxGQlV6cXlVQXNzOHplc1NIL2JwUVRLRW5jQklRUFBJQk5QVmtqaTl2RjNwbFVySmJxWDVGc1VoVTVQRXlwSzhQNTlxVC9FL0E9PSIsIldpdG5lc3MiOm51bGwsIlNlcXVlbmNlIjo0Mjk0OTY3Mjk1fV0sIlR4T3V0IjpbeyJWYWx1ZSI6MCwiUGtTY3JpcHQiOiJhaXhTVFVwUGJDdG9NMDVGYmxJME5uUmhORkpFZEN0MFRITm1XSEZLYmprMGFFWkllQzlrVG5aMlNWQlJQUT09In0seyJWYWx1ZSI6MjAwLCJQa1NjcmlwdCI6InFSUW5KNmR2OHZvNVhjVWxaS2pyS3F1TS9sSElkb2M9In0seyJWYWx1ZSI6MjI5MjcyLCJQa1NjcmlwdCI6ImRxa1Vndnk2bFFpK0VpUXk5N3VRMmw5MEFVQ21XNGlJckE9PSJ9XSwiTG9ja1RpbWUiOjB9LCJCbG9ja0hhc2giOlsxNzIsMjM2LDE2OCwxMDUsMTM0LDMzLDEzNSwxMzIsMTIsMjI1LDEyMywyMTIsMzksMjQ1LDE1LDE5MywxNDYsMTIzLDEwNSwxMTIsMzYsMTgwLDE4MiwxMDUsNDIsMjA3LDExNSwyMTgsMCwwLDAsMF19",
			txID:                     common.HashH([]byte{3}).String(),
			isExistsInPreviousBlocks: true,
		},
		// invalid shielding request: duplicated shielding proof in the current block
		{
			tokenID:                  pv4Common.PortalBTCIDStr,
			incAddressStr:            "12S5Lrs1XeQLbqN4ySyKtjAjd2d7sBP2tjFijzmp6avrrkQCNFMpkXm3FPzj2Wcu2ZNqJEmh9JriVuRErVwhuQnLmWSaggobEWsBEci",
			shieldingProof:           "eyJNZXJrbGVQcm9vZnMiOlt7IlByb29mSGFzaCI6WzM2LDgyLDI4LDUyLDgzLDE3OCwxNywxMzgsMjA1LDkyLDIyNCw4Myw2Myw2MSwxOTEsNTMsMTQ4LDIyNyw4OSwxODcsMjA0LDE3MSwxOCw3OSwyOCwxNDUsMzksMCwyMjMsMjksMTcwLDk3XSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzE5NiwxMSwxMiwyNTQsMTg1LDE0NCwxNTgsODgsMTU4LDI1LDQ2LDkxLDE2NCwyMjAsMTA4LDE2NSwxNzcsMjE5LDE5NSwyMDEsMjQ1LDE4OSw1LDQsMTIzLDE2OCw3NSwxNjcsMTQzLDE0NywyNDIsMjMyXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxMzgsNSwzOSw3NCwyNCw3NSw4MSw2MCwxNjcsNDYsMTg2LDEwNiwxNTAsNDQsMjAwLDIxLDIzOCw0MSwyMzQsMzksMjI1LDkyLDExLDIzNCwxNDAsMTA3LDI0OCwyNDQsMTQ0LDExNiwyMTksMTM2XSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzE4MiwxMDYsOTEsMTYxLDE0NSwxMzMsMjQ2LDc1LDIwOSw3NCwxODEsMTgyLDkyLDI1NCw0OSwxOTMsNTEsMjMzLDE1NywxODUsNTQsNzMsNTAsMjQ0LDEwNywzMiwzMSwxODksNDMsNCwxMTIsMTI4XSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxMiwzNSwxODIsMTk3LDE5NiwxODYsMTQzLDE1MSw0MywxMDMsMjU1LDE2LDE2MSwyNDAsMTM5LDE2OCwxNzEsOTgsODYsMTA3LDk3LDIxMiw5MCwxNjUsMTQ5LDYyLDMwLDY1LDc1LDIyOCw2NywxODFdLCJJc0xlZnQiOnRydWV9LHsiUHJvb2ZIYXNoIjpbNDcsMTY0LDYwLDcsODAsMTQsNzQsMTY1LDE5NSwxODYsMTE2LDY0LDExOCwxMDIsMTk1LDEsMTMxLDQ0LDU5LDE3MSwyMDEsMTU3LDc2LDUzLDgyLDksMTM4LDE3OSw5MSw2LDQsNDRdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzIzNSwyNDEsMTY4LDM0LDE4MywxNzgsNDEsMjUzLDEwMiwxMzYsMTg2LDg3LDE4OCwyMzQsMzgsMTU4LDExMSwyMjUsMTIyLDIzMCwyMjksNDgsMTgyLDEwNiwyNyw2NSwyMTQsNDIsMTUzLDQyLDMwLDkwXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOls2LDE4NCwyOCwyNDgsMTcyLDM0LDE0MywyNTEsMTcwLDEzLDIxNyw3OSwyMjcsMTA2LDIxMiw1NSw5MSwyMDMsMTAzLDkwLDkwLDIyLDI0Niw2NSw0OCwyMTMsMjU1LDE5OSwzOCwxMTMsMTkxLDIxXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsyOCwxOTQsMTQyLDMzLDQzLDg3LDIxLDIzNCwxOSwxOTEsMTYzLDIxMiwyMTcsMjUsNDksMTk5LDIwMywxNzIsMjUsNywxNjEsMTM2LDE2MywzMyw3OSwxODcsNDQsNzEsMTAxLDI5LDE4NSwyNTNdLCJJc0xlZnQiOmZhbHNlfV0sIkJUQ1R4Ijp7IlZlcnNpb24iOjEsIlR4SW4iOlt7IlByZXZpb3VzT3V0UG9pbnQiOnsiSGFzaCI6WzExLDMwLDIxNiwzLDUzLDE3MSwyNTUsMTcsMTEwLDE4NCwxMjYsMTEyLDE2LDE0MSwyMzgsMjIwLDE5NiwxMzEsMTI0LDE5Nyw5MywxOTcsMjAwLDIxMSwxMDIsMTM1LDQ4LDQ0LDEwLDIzMCw1MCwxMjJdLCJJbmRleCI6MH0sIlNpZ25hdHVyZVNjcmlwdCI6IlJ6QkVBaUJHajhPZnBtYzQzRHZyVm5icTlDa0ZhcnBDUzBIWWtUZUNMVUp6YW40UEhRSWdJckxYUUV2WXV4cHJEV3VEemxGQlV6cXlVQXNzOHplc1NIL2JwUVRLRW5jQklRUFBJQk5QVmtqaTl2RjNwbFVySmJxWDVGc1VoVTVQRXlwSzhQNTlxVC9FL0E9PSIsIldpdG5lc3MiOm51bGwsIlNlcXVlbmNlIjo0Mjk0OTY3Mjk1fV0sIlR4T3V0IjpbeyJWYWx1ZSI6MCwiUGtTY3JpcHQiOiJhaXhTVFVwUGJDdG9NMDVGYmxJME5uUmhORkpFZEN0MFRITm1XSEZLYmprMGFFWkllQzlrVG5aMlNWQlJQUT09In0seyJWYWx1ZSI6MjAwLCJQa1NjcmlwdCI6InFSUW5KNmR2OHZvNVhjVWxaS2pyS3F1TS9sSElkb2M9In0seyJWYWx1ZSI6MjI5MjcyLCJQa1NjcmlwdCI6ImRxa1Vndnk2bFFpK0VpUXk5N3VRMmw5MEFVQ21XNGlJckE9PSJ9XSwiTG9ja1RpbWUiOjB9LCJCbG9ja0hhc2giOlsxNzIsMjM2LDE2OCwxMDUsMTM0LDMzLDEzNSwxMzIsMTIsMjI1LDEyMywyMTIsMzksMjQ1LDE1LDE5MywxNDYsMTIzLDEwNSwxMTIsMzYsMTgwLDE4MiwxMDUsNDIsMjA3LDExNSwyMTgsMCwwLDAsMF19",
			txID:                     common.HashH([]byte{3}).String(),
			isExistsInPreviousBlocks: false,
		},
		// invalid shielding request: invalid proof (invalid memo)
		{
			tokenID:                  pv4Common.PortalBTCIDStr,
			incAddressStr:            "12S5Lrs1XeQLbqN4ySyKtjAjd2d7sBP2tjFijzmp6avrrkQCNFMpkXm3FPzj2Wcu2ZNqJEmh9JriVuRErVwhuQnLmWSaggobEWsBEci",
			shieldingProof:           "eyJNZXJrbGVQcm9vZnMiOlt7IlByb29mSGFzaCI6WzIxNywxMTgsNTgsOTMsMTM1LDIxLDEyLDIxLDIzNyw3MiwxNzgsMTU4LDkzLDIxOSwxMTYsMTMsMTIxLDE3OCwxMzQsMjUsMjM0LDE3OSwxOSwxMjMsMjM5LDIwNSwxNzEsMjMwLDE1OSwyMjYsNjIsMTgxXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOls1MCwxODgsMTgwLDkwLDUxLDIxNCwxOTYsODAsNTksNiwxOTQsNzUsNTYsMjI3LDE5Nyw4NiwxODYsMzYsMTk3LDExMywxODIsNzIsNzYsOTUsOSw0MywxMTAsMjI3LDc3LDE1MiwyNywxMjhdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzEyMywyNCwxOTUsNzIsMTU1LDE5MSwyMDMsNTEsMTU3LDQ3LDE4NCw3NCwxMTAsNjIsOTksMjA1LDEwMCwxMDksNDAsMTY2LDQwLDUsMTgwLDkxLDEyMiw5MywxOTIsNDgsMTMzLDIxNSw1NCwyMzFdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzExNSw4LDI0Nyw3LDI1NSwxNDMsNjUsNzcsMywxODEsMTMzLDI3LDE1MSwyMDksMTk0LDE5OSwxNDIsMjMwLDI1LDE1NCwxOCwxNTEsMTI4LDIzMiwxNDQsMjEzLDEsMTk5LDEzMSwyMDgsNDYsMjAzXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxOTUsMTkzLDM2LDQyLDE3MSwxNTUsMTM4LDIxMCwxNCw4OCwxMjYsMTI1LDE2NiwxMzMsMTIwLDE5NCw0NSw0OCw4LDE0MSwxNiwxMiwxODAsMyw3NSwxNTIsNjcsMTM4LDMwLDYsMTU0LDc4XSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxNTEsMjA2LDE3OCwyMzksMTE0LDM2LDYwLDE4OSwxNTMsNDYsMTYzLDE4MSwyMDUsNDIsMTg5LDE0MSwzNiw1MSwyMzYsMjI0LDI0OCwxOTYsNTgsMTY0LDE2MSwxMCw3MSwyMjgsMTk2LDI0NywxMDEsMTg5XSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzIyOSw2MSwyMTksNTYsMTUxLDIxNCw1LDQ3LDkzLDAsMjEyLDE2OSwxOTYsMTA0LDE3OCwxOTMsNzQsNTEsMzgsMTI0LDE3LDE3MSw3NiwxNjEsMTkwLDQ3LDEwNywyMzcsNDIsMTAzLDQ0LDkzXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxMjcsMjgsMzcsMjEzLDQ0LDE5MCwxNzYsMjIsMjMwLDEyMSwyNDEsMTc1LDEwMiwxOTAsMTc2LDI0Niw0MCwxMjYsODUsOTksMTAyLDE4MCwyMzYsMTIyLDE4Niw5NCw2NSw3MiwzMyw3OSwxNDIsMjM1XSwiSXNMZWZ0IjpmYWxzZX1dLCJCVENUeCI6eyJWZXJzaW9uIjoxLCJUeEluIjpbeyJQcmV2aW91c091dFBvaW50Ijp7Ikhhc2giOls0NywxMjAsODMsMTgzLDE4MCwxMzcsMjQ5LDIwMiwyMDQsMjQyLDIxOCwyNDksODIsNzksMjM3LDkxLDE1MywxNjMsMTM5LDM2LDE4NywxMzQsOTIsMTM2LDE2NCwxODAsMTQ5LDI0Myw3LDUxLDIzOCwxOTZdLCJJbmRleCI6Mn0sIlNpZ25hdHVyZVNjcmlwdCI6IlJ6QkVBaUFYQ2NwKzAzZmhSU3drVW82WktmNThteW9FbXJjMDQzNkV1aVFmOUs0dFd3SWdERUkxMjZ6TVZpTjR2Y0pWQ29scXNnd0YrRzdSOGQvZTNHTUFDOEFnOVZNQklRUFBJQk5QVmtqaTl2RjNwbFVySmJxWDVGc1VoVTVQRXlwSzhQNTlxVC9FL0E9PSIsIldpdG5lc3MiOm51bGwsIlNlcXVlbmNlIjo0Mjk0OTY3Mjk1fV0sIlR4T3V0IjpbeyJWYWx1ZSI6MCwiUGtTY3JpcHQiOiJhaXgzUkUxNmVXWjRkbGRTWTI5d09HbGpjSGx3U1cwNWIxSlRNa1pNZFhGQllWRnVOMmRQUjB4eVNVOVpQVEU9In0seyJWYWx1ZSI6NDAwLCJQa1NjcmlwdCI6InFSUW5KNmR2OHZvNVhjVWxaS2pyS3F1TS9sSElkb2M9In0seyJWYWx1ZSI6MjE5NTcyLCJQa1NjcmlwdCI6ImRxa1Vndnk2bFFpK0VpUXk5N3VRMmw5MEFVQ21XNGlJckE9PSJ9XSwiTG9ja1RpbWUiOjB9LCJCbG9ja0hhc2giOlsxNTUsMTksNSw4MSwzOSwyMTIsODEsMjMyLDk1LDM5LDE3MiwxMzQsMTQ4LDkwLDIwNiwyNiwyMywxNzYsMTkzLDIxOSw0NCwyNSwyNDIsODgsOCwwLDAsMCwwLDAsMCwwXX0=",
			txID:                     common.HashH([]byte{4}).String(),
			isExistsInPreviousBlocks: false,
		},
	}

	walletAddress := "2MvpFqydTR43TT4emMD84Mzhgd8F6dCow1X"

	// build expected results
	var txHash string
	var outputIdx uint32
	var outputAmount uint64

	txHash = "9fbfc05bc9359544ff1925ea89812ed81f38353af13f83cd34439f83769c6ba4"
	outputIdx = 1
	outputAmount = 200

	key1, value1 := generateUTXOKeyAndValue(pv4Common.PortalBTCIDStr, walletAddress, txHash, outputIdx, outputAmount)

	txHash = "6a3b123367bdcd6aaf61f391d4158cdaa6f34ee6c4d52d3a9d57920e683c396c"
	outputIdx = 1
	outputAmount = 600

	key2, value2 := generateUTXOKeyAndValue(pv4Common.PortalBTCIDStr, walletAddress, txHash, outputIdx, outputAmount)

	expectedRes := &ExpectedResultShieldingRequest{
		utxos: map[string]map[string]*statedb.UTXO{
			pv4Common.PortalBTCIDStr: {
				key1: value1,
				key2: value2,
			},
		},
		numBeaconInsts: 5,
		statusInsts: []string{
			pv4Common.PortalRequestAcceptedChainStatus,
			pv4Common.PortalRequestAcceptedChainStatus,
			pv4Common.PortalRequestRejectedChainStatus,
			pv4Common.PortalRequestRejectedChainStatus,
			pv4Common.PortalRequestRejectedChainStatus,
		},
	}

	return testcases, expectedRes
}

func buildPortalShieldingRequestAction(
	tokenID string,
	incAddressStr string,
	shieldingProof string,
	txID string,
	shardID byte,
) []string {
	data := pv4Meta.PortalShieldingRequest{
		MetadataBase: basemeta.MetadataBase{
			Type: basemeta.PortalShieldingRequestMeta,
		},
		TokenID:         tokenID,
		IncogAddressStr: incAddressStr,
		ShieldingProof:  shieldingProof,
	}
	txIDHash, _ := common.Hash{}.NewHashFromStr(txID)
	actionContent := pv4Meta.PortalShieldingRequestAction{
		Meta:    data,
		TxReqID: *txIDHash,
		ShardID: shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(basemeta.PortalShieldingRequestMeta), actionContentBase64Str}
}

func buildShieldingRequestActionsFromTcs(tcs []TestCaseShieldingRequest, shardID byte, shardHeight uint64) []portalV4InstForProducer {
	insts := []portalV4InstForProducer{}

	for _, tc := range tcs {
		inst := buildPortalShieldingRequestAction(
			tc.tokenID, tc.incAddressStr, tc.shieldingProof, tc.txID, shardID)
		insts = append(insts, portalV4InstForProducer{
			inst: inst,
			optionalData: map[string]interface{}{
				"isExistProofTxHash": tc.isExistsInPreviousBlocks,
			},
		})
	}

	return insts
}

func (s *PortalTestSuiteV4) TestShieldingRequest() {
	fmt.Println("Running TestShieldingRequest - beacon height 1003 ...")
	bc := s.blockChain

	// TODO: Init btc relaying blockchain and/or turn off verify merkle roof

	pm := portalprocessv4.NewPortalV4Manager()
	beaconHeight := uint64(1003)
	shardHeight := uint64(1003)
	shardHeights := map[byte]uint64{
		0: uint64(1003),
	}
	shardID := byte(0)

	s.SetupTestShieldingRequest()

	// build test cases
	testcases, expectedResult := buildTestCaseAndExpectedResultShieldingRequest()

	// build actions from testcases
	instsForProducer := buildShieldingRequestActionsFromTcs(testcases, shardID, shardHeight)

	// producer instructions
	newInsts, err := producerPortalInstructionsV4(
		bc, beaconHeight-1, shardHeights, instsForProducer, &s.currentPortalStateForProducer, s.portalParams, shardID, pm)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructionsV4(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, pm)

	// check results
	s.Equal(expectedResult.numBeaconInsts, uint(len(newInsts)))
	s.Equal(nil, err)

	for i, inst := range newInsts {
		s.Equal(expectedResult.statusInsts[i], inst[2], "Instruction index %v", i)
	}

	s.Equal(expectedResult.utxos, s.currentPortalStateForProducer.UTXOs)

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

/*
	Feature 6: Users redeem request
*/
type TestCaseUnshieldRequest struct {
	tokenID        string
	unshieldAmount uint64
	incAddressStr  string
	remoteAddress  string
	txId           string
	isExisted      bool
}

type ExpectedResultUnshieldRequest struct {
	waitingUnshieldReqs map[string]map[string]*statedb.WaitingUnshieldRequest
	//custodianPool     map[string]*statedb.CustodianState
	//waitingRedeemReq  map[string]*statedb.RedeemRequest
	numBeaconInsts uint
	statusInsts    []string
}

func (s *PortalTestSuiteV4) SetupTestUnshieldRequest() {
	// do nothing
}

func buildTestCaseAndExpectedResultUnshieldRequest() ([]TestCaseUnshieldRequest, *ExpectedResultUnshieldRequest) {
	beaconHeight := uint64(1003)
	// build test cases
	testcases := []TestCaseUnshieldRequest{
		// valid unshield request
		{
			tokenID:        pv4Common.PortalBNBIDStr,
			unshieldAmount: 1 * 1e9,
			incAddressStr:  USER_INC_ADDRESS_1,
			remoteAddress:  USER_BTC_ADDRESS_1,
			txId:           common.HashH([]byte{1}).String(),
			isExisted:      false,
		},
		// valid unshield request
		{
			tokenID:        pv4Common.PortalBNBIDStr,
			unshieldAmount: 0.5 * 1e9,
			incAddressStr:  USER_INC_ADDRESS_1,
			remoteAddress:  USER_BTC_ADDRESS_1,
			txId:           common.HashH([]byte{2}).String(),
			isExisted:      false,
		},
		// invalid unshield request
		{
			tokenID:        pv4Common.PortalBNBIDStr,
			unshieldAmount: 1 * 1e9,
			incAddressStr:  USER_INC_ADDRESS_1,
			remoteAddress:  USER_BTC_ADDRESS_1,
			txId:           common.HashH([]byte{3}).String(),
			isExisted:      true,
		},
	}

	// build expected results
	// waiting unshielding requests
	waitingUnshieldReqKey1 := statedb.GenerateWaitingUnshieldRequestObjectKey(pv4Common.PortalBTCIDStr, common.HashH([]byte{1}).String()).String()
	waitingUnshieldReq1 := statedb.NewWaitingUnshieldRequestStateWithValue(
		USER_BTC_ADDRESS_1, 1*1e9, common.HashH([]byte{1}).String(), beaconHeight)
	waitingUnshieldReqKey2 := statedb.GenerateWaitingUnshieldRequestObjectKey(pv4Common.PortalBTCIDStr, common.HashH([]byte{2}).String()).String()
	waitingUnshieldReq2 := statedb.NewWaitingUnshieldRequestStateWithValue(
		USER_BTC_ADDRESS_1, 0.5*1e9, common.HashH([]byte{2}).String(), beaconHeight)

	expectedRes := &ExpectedResultUnshieldRequest{
		waitingUnshieldReqs: map[string]map[string]*statedb.WaitingUnshieldRequest{
			pv4Common.PortalBTCIDStr: {
				waitingUnshieldReqKey1: waitingUnshieldReq1,
				waitingUnshieldReqKey2: waitingUnshieldReq2,
			},
		},
		numBeaconInsts: 3,
		statusInsts: []string{
			pv4Common.PortalRequestAcceptedChainStatus,
			pv4Common.PortalRequestAcceptedChainStatus,
			pv4Common.PortalRequestRejectedChainStatus,
		},
	}

	return testcases, expectedRes
}

func buildPortalUnshieldRequestAction(
	tokenID string,
	unshieldAmount uint64,
	incAddressStr string,
	remoteAddress string,
	txID string,
	shardID byte,
) []string {
	data := pv4Meta.PortalUnshieldRequest{
		MetadataBase: basemeta.MetadataBase{
			Type: basemeta.PortalBurnPTokenMeta,
		},
		IncAddressStr:  incAddressStr,
		RemoteAddress:  remoteAddress,
		TokenID:        tokenID,
		UnshieldAmount: unshieldAmount,
	}
	txIDHash, _ := common.Hash{}.NewHashFromStr(txID)
	actionContent := pv4Meta.PortalUnshieldRequestAction{
		Meta:    data,
		TxReqID: *txIDHash,
		ShardID: shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(basemeta.PortalBurnPTokenMeta), actionContentBase64Str}
}

func buildUnshieldRequestActionsFromTcs(tcs []TestCaseUnshieldRequest, shardID byte, shardHeight uint64) []portalV4InstForProducer {
	insts := []portalV4InstForProducer{}

	for _, tc := range tcs {
		inst := buildPortalUnshieldRequestAction(
			tc.tokenID, tc.unshieldAmount, tc.incAddressStr, tc.remoteAddress, tc.txId, shardID)
		insts = append(insts, portalV4InstForProducer{
			inst: inst,
			optionalData: map[string]interface{}{
				"isExistUnshieldID": tc.isExisted,
			},
		})
	}

	return insts
}

func (s *PortalTestSuiteV4) TestUnshieldRequest() {
	fmt.Println("Running TestUnshieldRequest - beacon height 1003 ...")
	bc := s.blockChain
	pm := portalprocessv4.NewPortalV4Manager()
	beaconHeight := uint64(1003)
	shardHeight := uint64(1003)
	shardHeights := map[byte]uint64{
		0: uint64(1003),
	}
	shardID := byte(0)

	s.SetupTestUnshieldRequest()

	// build test cases
	testcases, expectedResult := buildTestCaseAndExpectedResultUnshieldRequest()

	// build actions from testcases
	instsForProducer := buildUnshieldRequestActionsFromTcs(testcases, shardID, shardHeight)

	// producer instructions
	newInsts, err := producerPortalInstructionsV4(
		bc, beaconHeight-1, shardHeights, instsForProducer, &s.currentPortalStateForProducer, s.portalParams, shardID, pm)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructionsV4(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, pm)

	// check results
	s.Equal(expectedResult.numBeaconInsts, uint(len(newInsts)))
	s.Equal(nil, err)

	for i, inst := range newInsts {
		s.Equal(expectedResult.statusInsts[i], inst[2], "Instruction index %v", i)
	}

	s.Equal(expectedResult.waitingUnshieldReqs, s.currentPortalStateForProducer.WaitingUnshieldRequests)

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

/*
	Feature 7: fee replacement
*/

const BatchID1 = "batch1"
const BatchID2 = "batch2"
const BatchID3 = "batch3"
const keyBatchShield1 = "9da36f3e18071935a3d812f47e2cb86f48f49260681df4129d4538f9bfcd4cad"
const keyBatchShield2 = "b83ad865d55f3e5399e455ad5c561ecc9b31f8cb681df4129d4538f9bfcd4cad"
const keyBatchShield3 = "8da36f3e18071935a3d812f47e2cb86f48f49260681df4129d4538f9bfcd4cad"

type OutPut struct {
	externalAddress string
	amount          uint64
}

type TestCaseFeeReplacement struct {
	custodianIncAddress string
	batchID             string
	fee                 uint
	tokenID             string
	outputs             []OutPut
}

type ExpectedResultFeeReplacement struct {
	processedUnshieldRequests map[string]map[string]*statedb.ProcessedUnshieldRequestBatch
	numBeaconInsts            uint
	statusInsts               []string
}

func (s *PortalTestSuiteV4) SetupTestFeeReplacement() {

	btcMultiSigAddress := s.portalParams.MultiSigAddresses[pv4Common.PortalBTCIDStr]
	processUnshield1 := statedb.NewProcessedUnshieldRequestBatchWithValue(
		BatchID1,
		[]string{"txid1", "txid2", "txid3"},
		map[string][]*statedb.UTXO{
			btcMultiSigAddress: {
				statedb.NewUTXOWithValue(btcMultiSigAddress, "7a4734c33040cc93794722b29c75020a9a8364cb294a525704f33712acbb41aa", 1, 1000000),
				statedb.NewUTXOWithValue(btcMultiSigAddress, "49491148bd2f7b5432a26472af97724e114f22a74d9d2fb20c619b4f79f19fd9", 0, 2000000),
				statedb.NewUTXOWithValue(btcMultiSigAddress, "b751ff30df21ad84ce3f509ee3981c348143bd6a5aa30f4256ecb663fab14fd1", 1, 3000000),
			},
		},
		map[uint64]uint{
			900: 900,
		},
	)

	processUnshield2 := statedb.NewProcessedUnshieldRequestBatchWithValue(
		BatchID2,
		[]string{"txid4", "txid5"},
		map[string][]*statedb.UTXO{
			btcMultiSigAddress: {
				statedb.NewUTXOWithValue(btcMultiSigAddress, "163a6cc24df4efbd5c997aa623d4e319f1b7671be83a86bb0fa27bc701ae4a76", 1, 1000000),
			},
		},
		map[uint64]uint{
			1000: 1000,
		},
	)

	processedUnshieldRequests := map[string]map[string]*statedb.ProcessedUnshieldRequestBatch{
		pv4Common.PortalBTCIDStr: {
			keyBatchShield1: processUnshield1,
			keyBatchShield2: processUnshield2,
		},
	}

	s.currentPortalStateForProducer.ProcessedUnshieldRequests = processedUnshieldRequests
	s.currentPortalStateForProcess.ProcessedUnshieldRequests = CloneUnshieldBatchRequests(processedUnshieldRequests)

}

func buildFeeReplacementActionsFromTcs(tcs []TestCaseFeeReplacement, shardID byte) []portalV4InstForProducer {
	insts := []portalV4InstForProducer{}

	for _, tc := range tcs {
		inst := buildPortalFeeReplacementAction(
			tc.custodianIncAddress,
			tc.tokenID,
			tc.batchID,
			tc.fee,
			shardID,
		)
		optionalData := make(map[string]interface{})
		outputs := make([]*portalTokensV4.OutputTx, 0)
		for _, v := range tc.outputs {
			outputs = append(outputs, &portalTokensV4.OutputTx{ReceiverAddress: v.externalAddress, Amount: v.amount})
		}
		optionalData["outputs"] = outputs
		insts = append(insts, portalV4InstForProducer{
			inst:         inst,
			optionalData: optionalData,
		})
	}

	return insts
}

func buildPortalFeeReplacementAction(
	incAddressStr string,
	tokenID string,
	batchID string,
	fee uint,
	shardID byte,
) []string {
	data := pv4Meta.PortalReplacementFeeRequest{
		MetadataBase: basemeta.MetadataBase{
			Type: basemeta.PortalReplacementFeeRequestMeta,
		},
		IncAddressStr: incAddressStr,
		TokenID:       tokenID,
		BatchID:       batchID,
		Fee:           fee,
	}

	actionContent := pv4Meta.PortalReplacementFeeRequestAction{
		Meta:    data,
		TxReqID: common.Hash{},
		ShardID: shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(basemeta.PortalReplacementFeeRequestMeta), actionContentBase64Str}
}

func buildExpectedResultFeeReplacement(s *PortalTestSuiteV4) ([]TestCaseFeeReplacement, *ExpectedResultFeeReplacement) {

	testcases := []TestCaseFeeReplacement{
		// request replace fee higher than max step
		{
			custodianIncAddress: CUS_INC_ADDRESS_1,
			tokenID:             pv4Common.PortalBTCIDStr,
			batchID:             BatchID1,
			fee:                 1500,
			outputs: []OutPut{
				{
					externalAddress: "bc1qqyxfxeh6k5kt29e30pzhxs7kd59fvr76u95qat",
					amount:          100,
				},
				{
					externalAddress: "bc1qj9dgez2sstg8d06ehjgw6wf4hsjxr3aake0dzs",
					amount:          100,
				},
			},
		},
		// request replace lower than latest request
		{
			custodianIncAddress: CUS_INC_ADDRESS_1,
			tokenID:             pv4Common.PortalBTCIDStr,
			batchID:             BatchID1,
			fee:                 800,
			outputs: []OutPut{
				{
					externalAddress: "bc1qqyxfxeh6k5kt29e30pzhxs7kd59fvr76u95qat",
					amount:          100,
				},
				{
					externalAddress: "bc1qj9dgez2sstg8d06ehjgw6wf4hsjxr3aake0dzs",
					amount:          100,
				},
			},
		},
		// request replace fee successfully
		{
			custodianIncAddress: CUS_INC_ADDRESS_1,
			tokenID:             pv4Common.PortalBTCIDStr,
			batchID:             BatchID1,
			fee:                 1200,
			outputs: []OutPut{
				{
					externalAddress: "bc1qqyxfxeh6k5kt29e30pzhxs7kd59fvr76u95qat",
					amount:          100,
				},
				{
					externalAddress: "bc1qj9dgez2sstg8d06ehjgw6wf4hsjxr3aake0dzs",
					amount:          100,
				},
			},
		},
		// request replace fee with beacon height lower than next acceptable beacon height
		{
			custodianIncAddress: CUS_INC_ADDRESS_1,
			tokenID:             pv4Common.PortalBTCIDStr,
			batchID:             BatchID1,
			fee:                 1300,
			outputs: []OutPut{
				{
					externalAddress: "bc1qqyxfxeh6k5kt29e30pzhxs7kd59fvr76u95qat",
					amount:          100,
				},
				{
					externalAddress: "bc1qj9dgez2sstg8d06ehjgw6wf4hsjxr3aake0dzs",
					amount:          100,
				},
			},
		},
		// request replace fee new batch id
		{
			custodianIncAddress: CUS_INC_ADDRESS_1,
			tokenID:             pv4Common.PortalBTCIDStr,
			batchID:             BatchID2,
			fee:                 1500,
			outputs: []OutPut{
				{
					externalAddress: "18d9DFY9oGVCLUg7mPbqj3ZxePspypsUHo",
					amount:          200,
				},
			},
		},
		// request replace fee with non exist batch id
		{
			custodianIncAddress: CUS_INC_ADDRESS_1,
			tokenID:             pv4Common.PortalBTCIDStr,
			batchID:             BatchID3,
			fee:                 1500,
			outputs: []OutPut{
				{
					externalAddress: "18d9DFY9oGVCLUg7mPbqj3ZxePspypsUHo",
					amount:          100,
				},
			},
		},
	}

	btcMultiSigAddress := s.portalParams.MultiSigAddresses[pv4Common.PortalBTCIDStr]
	processUnshield1 := statedb.NewProcessedUnshieldRequestBatchWithValue(
		BatchID1,
		[]string{"txid1", "txid2", "txid3"},
		map[string][]*statedb.UTXO{
			btcMultiSigAddress: {
				statedb.NewUTXOWithValue(btcMultiSigAddress, "7a4734c33040cc93794722b29c75020a9a8364cb294a525704f33712acbb41aa", 1, 1000000),
				statedb.NewUTXOWithValue(btcMultiSigAddress, "49491148bd2f7b5432a26472af97724e114f22a74d9d2fb20c619b4f79f19fd9", 0, 2000000),
				statedb.NewUTXOWithValue(btcMultiSigAddress, "b751ff30df21ad84ce3f509ee3981c348143bd6a5aa30f4256ecb663fab14fd1", 1, 3000000),
			},
		},
		map[uint64]uint{
			900:  900,
			1500: 1200,
		},
	)

	processUnshield2 := statedb.NewProcessedUnshieldRequestBatchWithValue(
		BatchID2,
		[]string{"txid4", "txid5"},
		map[string][]*statedb.UTXO{
			btcMultiSigAddress: {
				statedb.NewUTXOWithValue(btcMultiSigAddress, "163a6cc24df4efbd5c997aa623d4e319f1b7671be83a86bb0fa27bc701ae4a76", 1, 1000000),
			},
		},
		map[uint64]uint{
			1000: 1000,
			1500: 1500,
		},
	)

	processedUnshieldRequests := map[string]map[string]*statedb.ProcessedUnshieldRequestBatch{
		pv4Common.PortalBTCIDStr: {
			keyBatchShield1: processUnshield1,
			keyBatchShield2: processUnshield2,
		},
	}

	// build expected results
	expectedRes := &ExpectedResultFeeReplacement{
		processedUnshieldRequests: processedUnshieldRequests,
		numBeaconInsts:            6,
		statusInsts: []string{
			pCommon.PortalRequestRejectedChainStatus,
			pCommon.PortalRequestRejectedChainStatus,
			pCommon.PortalRequestAcceptedChainStatus,
			pCommon.PortalRequestRejectedChainStatus,
			pCommon.PortalRequestAcceptedChainStatus,
			pCommon.PortalRequestRejectedChainStatus,
		},
	}

	return testcases, expectedRes
}

func (s *PortalTestSuiteV4) TestFeeReplacement() {
	fmt.Println("Running TestUnlockOverRateCollaterals - beacon height 1501 ...")
	bc := s.blockChain
	beaconHeight := uint64(1501)
	shardHeights := map[byte]uint64{
		0: uint64(1501),
	}
	shardID := byte(0)
	pm := portalprocessv4.NewPortalV4Manager()

	s.SetupTestFeeReplacement()

	unshieldBatchPool := s.currentPortalStateForProducer.ProcessedUnshieldRequests
	for key, unshieldBatch := range unshieldBatchPool {
		fmt.Printf("cusKey %v - custodian address: %v\n", key, unshieldBatch)
	}

	testcases, expectedResult := buildExpectedResultFeeReplacement(s)

	// build actions from testcases
	instsForProducer := buildFeeReplacementActionsFromTcs(testcases, shardID)

	newInsts, err := producerPortalInstructionsV4(
		bc, beaconHeight-1, shardHeights, instsForProducer, &s.currentPortalStateForProducer, s.portalParams, shardID, pm)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructionsV4(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, pm)

	// check results
	s.Equal(expectedResult.numBeaconInsts, uint(len(newInsts)))
	s.Equal(nil, err)

	for i, inst := range newInsts {
		s.Equal(expectedResult.statusInsts[i], inst[2], "Instruction index %v", i)
	}

	s.Equal(expectedResult.processedUnshieldRequests, s.currentPortalStateForProducer.ProcessedUnshieldRequests)

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

/*
	Feature 8: submit confirmed external transaction
*/

const confirmedTxProof1 = "eyJNZXJrbGVQcm9vZnMiOlt7IlByb29mSGFzaCI6WzEwNCw2MywyMzUsMTMzLDE5NCw4MiwyMDEsMTU2LDE3Miw4MiwyLDc5LDc2LDIzLDUwLDIyNCwxMDgsMTU3LDc5LDEyOCwxNzcsMTkzLDE3Myw1MSw2LDAsMTE1LDIzNiw1NSwxMDAsMTYwLDY0XSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsyMTMsMTMzLDExLDc5LDIyNSwxMTcsMTgsMjQ3LDEyMSwxOTcsNjUsMTk4LDYxLDYyLDE4NSwyMTEsMTIsMjA5LDEzNCw2NSwyMzYsOTUsMTA0LDEyNywxMiwyMTEsMjE0LDMyLDE5Miw0MSwxNDcsMTcxXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxNjcsMjIzLDEyMCwyMjAsMTM1LDIyMCwyMzMsODgsMjQsMjQyLDQwLDEsMTU2LDIzMSwzNywyMTAsMTAzLDE3OCw3NCwyNTAsOTgsMTU1LDY3LDIzNyw5NSwxNjksMjIzLDE2OSw5LDE4OSw3MiwxNjBdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzcxLDEzLDc4LDU3LDU1LDEzOCwxNzIsMTk5LDczLDExMyw2OSwxMyw1OCwxMTksMjU1LDI0LDIyLDQsMjMyLDIyMyw0NiwxNzgsMTkyLDE2MiwxNzMsMjE3LDE5MCwxOTUsMTk0LDIxNSwxNTAsMzVdLCJJc0xlZnQiOnRydWV9LHsiUHJvb2ZIYXNoIjpbMTIsMzUsMTgyLDE5NywxOTYsMTg2LDE0MywxNTEsNDMsMTAzLDI1NSwxNiwxNjEsMjQwLDEzOSwxNjgsMTcxLDk4LDg2LDEwNyw5NywyMTIsOTAsMTY1LDE0OSw2MiwzMCw2NSw3NSwyMjgsNjcsMTgxXSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzQ3LDE2NCw2MCw3LDgwLDE0LDc0LDE2NSwxOTUsMTg2LDExNiw2NCwxMTgsMTAyLDE5NSwxLDEzMSw0NCw1OSwxNzEsMjAxLDE1Nyw3Niw1Myw4Miw5LDEzOCwxNzksOTEsNiw0LDQ0XSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsyMzUsMjQxLDE2OCwzNCwxODMsMTc4LDQxLDI1MywxMDIsMTM2LDE4Niw4NywxODgsMjM0LDM4LDE1OCwxMTEsMjI1LDEyMiwyMzAsMjI5LDQ4LDE4MiwxMDYsMjcsNjUsMjE0LDQyLDE1Myw0MiwzMCw5MF0sIklzTGVmdCI6ZmFsc2V9LHsiUHJvb2ZIYXNoIjpbNiwxODQsMjgsMjQ4LDE3MiwzNCwxNDMsMjUxLDE3MCwxMywyMTcsNzksMjI3LDEwNiwyMTIsNTUsOTEsMjAzLDEwMyw5MCw5MCwyMiwyNDYsNjUsNDgsMjEzLDI1NSwxOTksMzgsMTEzLDE5MSwyMV0sIklzTGVmdCI6ZmFsc2V9LHsiUHJvb2ZIYXNoIjpbMjgsMTk0LDE0MiwzMyw0Myw4NywyMSwyMzQsMTksMTkxLDE2MywyMTIsMjE3LDI1LDQ5LDE5OSwyMDMsMTcyLDI1LDcsMTYxLDEzNiwxNjMsMzMsNzksMTg3LDQ0LDcxLDEwMSwyOSwxODUsMjUzXSwiSXNMZWZ0IjpmYWxzZX1dLCJCVENUeCI6eyJWZXJzaW9uIjoxLCJUeEluIjpbeyJQcmV2aW91c091dFBvaW50Ijp7Ikhhc2giOlsxMDgsNTcsNjAsMTA0LDE0LDE0Niw4NywxNTcsNTgsNDUsMjEzLDE5NiwyMzAsNzgsMjQzLDE2NiwyMTgsMTQwLDIxLDIxMiwxNDUsMjQzLDk3LDE3NSwxMDYsMjA1LDE4OSwxMDMsNTEsMTgsNTksMTA2XSwiSW5kZXgiOjJ9LCJTaWduYXR1cmVTY3JpcHQiOiJTREJGQWlFQXM2Y0lidmJBQ0VNNXRMSFdGMmpVMHZtaVJkaE0rR2FuajhxdTVXK3lvM2NDSUhhcjNKNU85cVRGeWZFVGNNZEFSNmFhcTZNQjNLK29jcnNxR3UxSEI5T29BU0VEenlBVFQxWkk0dmJ4ZDZaVkt5VzZsK1JiRklWT1R4TXFTdkQrZmFrL3hQdz0iLCJXaXRuZXNzIjpudWxsLCJTZXF1ZW5jZSI6NDI5NDk2NzI5NX1dLCJUeE91dCI6W3siVmFsdWUiOjAsIlBrU2NyaXB0IjoiYWl4M1JFMTZlV1o0ZGxkU1kyOXdPR2xqY0hsd1NXMDViMUpUTWtaTWRYRkJZVkZ1TjJkUFIweHlTVTlaUFE9PSJ9LHsiVmFsdWUiOjQwMCwiUGtTY3JpcHQiOiJxUlFuSjZkdjh2bzVYY1VsWktqcktxdU0vbEhJZG9jPSJ9LHsiVmFsdWUiOjIyMTk3MiwiUGtTY3JpcHQiOiJkcWtVZ3Z5NmxRaStFaVF5OTd1UTJsOTBBVUNtVzRpSXJBPT0ifV0sIkxvY2tUaW1lIjowfSwiQmxvY2tIYXNoIjpbMTcyLDIzNiwxNjgsMTA1LDEzNCwzMywxMzUsMTMyLDEyLDIyNSwxMjMsMjEyLDM5LDI0NSwxNSwxOTMsMTQ2LDEyMywxMDUsMTEyLDM2LDE4MCwxODIsMTA1LDQyLDIwNywxMTUsMjE4LDAsMCwwLDBdfQ=="
const confirmedTxProof2 = "eyJNZXJrbGVQcm9vZnMiOlt7IlByb29mSGFzaCI6WzEwOCw1Nyw2MCwxMDQsMTQsMTQ2LDg3LDE1Nyw1OCw0NSwyMTMsMTk2LDIzMCw3OCwyNDMsMTY2LDIxOCwxNDAsMjEsMjEyLDE0NSwyNDMsOTcsMTc1LDEwNiwyMDUsMTg5LDEwMyw1MSwxOCw1OSwxMDZdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzQ4LDE4MiwxMTYsMjUwLDM5LDEwOCwxOTUsMTQ0LDIxLDc5LDIyMiw3NCwxOTcsMTYxLDEwNywxNjAsMjEsMzAsMjA2LDI0OSwxNzksMTExLDIyMywzMiw0NywxMzksMTUzLDI4LDE5MiwyMjAsMTQ2LDI1XSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzEzOCw1LDM5LDc0LDI0LDc1LDgxLDYwLDE2Nyw0NiwxODYsMTA2LDE1MCw0NCwyMDAsMjEsMjM4LDQxLDIzNCwzOSwyMjUsOTIsMTEsMjM0LDE0MCwxMDcsMjQ4LDI0NCwxNDQsMTE2LDIxOSwxMzZdLCJJc0xlZnQiOnRydWV9LHsiUHJvb2ZIYXNoIjpbMTgyLDEwNiw5MSwxNjEsMTQ1LDEzMywyNDYsNzUsMjA5LDc0LDE4MSwxODIsOTIsMjU0LDQ5LDE5Myw1MSwyMzMsMTU3LDE4NSw1NCw3Myw1MCwyNDQsMTA3LDMyLDMxLDE4OSw0Myw0LDExMiwxMjhdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzEyLDM1LDE4MiwxOTcsMTk2LDE4NiwxNDMsMTUxLDQzLDEwMywyNTUsMTYsMTYxLDI0MCwxMzksMTY4LDE3MSw5OCw4NiwxMDcsOTcsMjEyLDkwLDE2NSwxNDksNjIsMzAsNjUsNzUsMjI4LDY3LDE4MV0sIklzTGVmdCI6dHJ1ZX0seyJQcm9vZkhhc2giOls0NywxNjQsNjAsNyw4MCwxNCw3NCwxNjUsMTk1LDE4NiwxMTYsNjQsMTE4LDEwMiwxOTUsMSwxMzEsNDQsNTksMTcxLDIwMSwxNTcsNzYsNTMsODIsOSwxMzgsMTc5LDkxLDYsNCw0NF0sIklzTGVmdCI6ZmFsc2V9LHsiUHJvb2ZIYXNoIjpbMjM1LDI0MSwxNjgsMzQsMTgzLDE3OCw0MSwyNTMsMTAyLDEzNiwxODYsODcsMTg4LDIzNCwzOCwxNTgsMTExLDIyNSwxMjIsMjMwLDIyOSw0OCwxODIsMTA2LDI3LDY1LDIxNCw0MiwxNTMsNDIsMzAsOTBdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzYsMTg0LDI4LDI0OCwxNzIsMzQsMTQzLDI1MSwxNzAsMTMsMjE3LDc5LDIyNywxMDYsMjEyLDU1LDkxLDIwMywxMDMsOTAsOTAsMjIsMjQ2LDY1LDQ4LDIxMywyNTUsMTk5LDM4LDExMywxOTEsMjFdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzI4LDE5NCwxNDIsMzMsNDMsODcsMjEsMjM0LDE5LDE5MSwxNjMsMjEyLDIxNywyNSw0OSwxOTksMjAzLDE3MiwyNSw3LDE2MSwxMzYsMTYzLDMzLDc5LDE4Nyw0NCw3MSwxMDEsMjksMTg1LDI1M10sIklzTGVmdCI6ZmFsc2V9XSwiQlRDVHgiOnsiVmVyc2lvbiI6MSwiVHhJbiI6W3siUHJldmlvdXNPdXRQb2ludCI6eyJIYXNoIjpbMTY0LDEwNywxNTYsMTE4LDEzMSwxNTksNjcsNTIsMjA1LDEzMSw2MywyNDEsNTgsNTMsNTYsMzEsMjE2LDQ2LDEyOSwxMzcsMjM0LDM3LDI1LDI1NSw2OCwxNDksNTMsMjAxLDkxLDE5MiwxOTEsMTU5XSwiSW5kZXgiOjJ9LCJTaWduYXR1cmVTY3JpcHQiOiJTREJGQWlFQXJZQkpDV1VDUWhTSFZuc3k2UWJkNXlhK0k1bWxZc01GaU1pck8wTkdiRGdDSUdEOW5DRFZVUHBvQkNDYVY4YWZPc0lBOHU1aU43Q240M2NvT0krd0tYNGpBU0VEenlBVFQxWkk0dmJ4ZDZaVkt5VzZsK1JiRklWT1R4TXFTdkQrZmFrL3hQdz0iLCJXaXRuZXNzIjpudWxsLCJTZXF1ZW5jZSI6NDI5NDk2NzI5NX1dLCJUeE91dCI6W3siVmFsdWUiOjAsIlBrU2NyaXB0IjoiYWl4U1RVcFBiQ3RvTTA1RmJsSTBOblJoTkZKRWRDdDBUSE5tV0hGS2JqazBhRVpJZUM5a1RuWjJTVkJSUFE9PSJ9LHsiVmFsdWUiOjMwMCwiUGtTY3JpcHQiOiJxUlFuSjZkdjh2bzVYY1VsWktqcktxdU0vbEhJZG9jPSJ9LHsiVmFsdWUiOjIyNjk3MiwiUGtTY3JpcHQiOiJkcWtVZ3Z5NmxRaStFaVF5OTd1UTJsOTBBVUNtVzRpSXJBPT0ifV0sIkxvY2tUaW1lIjowfSwiQmxvY2tIYXNoIjpbMTcyLDIzNiwxNjgsMTA1LDEzNCwzMywxMzUsMTMyLDEyLDIyNSwxMjMsMjEyLDM5LDI0NSwxNSwxOTMsMTQ2LDEyMywxMDUsMTEyLDM2LDE4MCwxODIsMTA1LDQyLDIwNywxMTUsMjE4LDAsMCwwLDBdfQ=="
const confirmedTxProof3 = "eyJNZXJrbGVQcm9vZnMiOlt7IlByb29mSGFzaCI6WzE0MywyMTEsMjI2LDExNiwyNTMsNjksMjQ2LDIyNCwxMTAsMTg0LDMwLDE1Nyw4NCwyMDcsMTQyLDI1MywxMjIsNTAsMTk0LDgsMjAzLDExOSw3NSwxODMsMjUsNjUsMTU1LDIxMywxODYsMTg0LDEyNSwxMF0sIklzTGVmdCI6dHJ1ZX0seyJQcm9vZkhhc2giOls0OCwxODIsMTE2LDI1MCwzOSwxMDgsMTk1LDE0NCwyMSw3OSwyMjIsNzQsMTk3LDE2MSwxMDcsMTYwLDIxLDMwLDIwNiwyNDksMTc5LDExMSwyMjMsMzIsNDcsMTM5LDE1MywyOCwxOTIsMjIwLDE0NiwyNV0sIklzTGVmdCI6dHJ1ZX0seyJQcm9vZkhhc2giOlsxMzgsNSwzOSw3NCwyNCw3NSw4MSw2MCwxNjcsNDYsMTg2LDEwNiwxNTAsNDQsMjAwLDIxLDIzOCw0MSwyMzQsMzksMjI1LDkyLDExLDIzNCwxNDAsMTA3LDI0OCwyNDQsMTQ0LDExNiwyMTksMTM2XSwiSXNMZWZ0Ijp0cnVlfSx7IlByb29mSGFzaCI6WzE4MiwxMDYsOTEsMTYxLDE0NSwxMzMsMjQ2LDc1LDIwOSw3NCwxODEsMTgyLDkyLDI1NCw0OSwxOTMsNTEsMjMzLDE1NywxODUsNTQsNzMsNTAsMjQ0LDEwNywzMiwzMSwxODksNDMsNCwxMTIsMTI4XSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsxMiwzNSwxODIsMTk3LDE5NiwxODYsMTQzLDE1MSw0MywxMDMsMjU1LDE2LDE2MSwyNDAsMTM5LDE2OCwxNzEsOTgsODYsMTA3LDk3LDIxMiw5MCwxNjUsMTQ5LDYyLDMwLDY1LDc1LDIyOCw2NywxODFdLCJJc0xlZnQiOnRydWV9LHsiUHJvb2ZIYXNoIjpbNDcsMTY0LDYwLDcsODAsMTQsNzQsMTY1LDE5NSwxODYsMTE2LDY0LDExOCwxMDIsMTk1LDEsMTMxLDQ0LDU5LDE3MSwyMDEsMTU3LDc2LDUzLDgyLDksMTM4LDE3OSw5MSw2LDQsNDRdLCJJc0xlZnQiOmZhbHNlfSx7IlByb29mSGFzaCI6WzIzNSwyNDEsMTY4LDM0LDE4MywxNzgsNDEsMjUzLDEwMiwxMzYsMTg2LDg3LDE4OCwyMzQsMzgsMTU4LDExMSwyMjUsMTIyLDIzMCwyMjksNDgsMTgyLDEwNiwyNyw2NSwyMTQsNDIsMTUzLDQyLDMwLDkwXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOls2LDE4NCwyOCwyNDgsMTcyLDM0LDE0MywyNTEsMTcwLDEzLDIxNyw3OSwyMjcsMTA2LDIxMiw1NSw5MSwyMDMsMTAzLDkwLDkwLDIyLDI0Niw2NSw0OCwyMTMsMjU1LDE5OSwzOCwxMTMsMTkxLDIxXSwiSXNMZWZ0IjpmYWxzZX0seyJQcm9vZkhhc2giOlsyOCwxOTQsMTQyLDMzLDQzLDg3LDIxLDIzNCwxOSwxOTEsMTYzLDIxMiwyMTcsMjUsNDksMTk5LDIwMywxNzIsMjUsNywxNjEsMTM2LDE2MywzMyw3OSwxODcsNDQsNzEsMTAxLDI5LDE4NSwyNTNdLCJJc0xlZnQiOmZhbHNlfV0sIkJUQ1R4Ijp7IlZlcnNpb24iOjEsIlR4SW4iOlt7IlByZXZpb3VzT3V0UG9pbnQiOnsiSGFzaCI6WzE0MywyMTEsMjI2LDExNiwyNTMsNjksMjQ2LDIyNCwxMTAsMTg0LDMwLDE1Nyw4NCwyMDcsMTQyLDI1MywxMjIsNTAsMTk0LDgsMjAzLDExOSw3NSwxODMsMjUsNjUsMTU1LDIxMywxODYsMTg0LDEyNSwxMF0sIkluZGV4IjoyfSwiU2lnbmF0dXJlU2NyaXB0IjoiU0RCRkFpRUE5WS9XeDZvMDh4QjAzZkdya3EyVGQ5NXV5akxiK0ZTRk13cHpWcHdkZTFNQ0lGcWJzdWlWeis5Wnhka05YWmZXQ1p5WHZMdUJrK3Y1KzZzYk1kbGUwSVkvQVNFRHp5QVRUMVpJNHZieGQ2WlZLeVc2bCtSYkZJVk9UeE1xU3ZEK2Zhay94UHc9IiwiV2l0bmVzcyI6bnVsbCwiU2VxdWVuY2UiOjQyOTQ5NjcyOTV9XSwiVHhPdXQiOlt7IlZhbHVlIjowLCJQa1NjcmlwdCI6ImFpeFNUVXBQYkN0b00wNUZibEkwTm5SaE5GSkVkQ3QwVEhObVdIRktiamswYUVaSWVDOWtUbloyU1ZCUlBRPT0ifSx7IlZhbHVlIjo2MDAsIlBrU2NyaXB0IjoicVJRbko2ZHY4dm81WGNVbFpLanJLcXVNL2xISWRvYz0ifSx7IlZhbHVlIjoyMjQzNzIsIlBrU2NyaXB0IjoiZHFrVWd2eTZsUWkrRWlReTk3dVEybDkwQVVDbVc0aUlyQT09In1dLCJMb2NrVGltZSI6MH0sIkJsb2NrSGFzaCI6WzE3MiwyMzYsMTY4LDEwNSwxMzQsMzMsMTM1LDEzMiwxMiwyMjUsMTIzLDIxMiwzOSwyNDUsMTUsMTkzLDE0NiwxMjMsMTA1LDExMiwzNiwxODAsMTgyLDEwNSw0MiwyMDcsMTE1LDIxOCwwLDAsMCwwXX0="

type TestCaseSubmitConfirmedTx struct {
	confirmedTxProof string
	batchID          string
	tokenID          string
	outputs          []OutPut
}

type ExpectedResultSubmitConfirmedTx struct {
	utxos                     map[string]map[string]*statedb.UTXO
	processedUnshieldRequests map[string]map[string]*statedb.ProcessedUnshieldRequestBatch
	numBeaconInsts            uint
	statusInsts               []string
}

func (s *PortalTestSuiteV4) SetupTestSubmitConfirmedTx() {

	btcMultiSigAddress := s.portalParams.MultiSigAddresses[pv4Common.PortalBTCIDStr]
	utxos := map[string]map[string]*statedb.UTXO{
		pv4Common.PortalBTCIDStr: {
			statedb.GenerateUTXOObjectKey(pv4Common.PortalBTCIDStr, btcMultiSigAddress, "7a4734c33040cc93794722b29c75020a9a8364cb294a525704f33712acbb41aa", 0).String(): statedb.NewUTXOWithValue(btcMultiSigAddress, "7a4734c33040cc93794722b29c75020a9a8364cb294a525704f33712acbb41aa", 1, 100000),
		},
	}

	processUnshield1 := statedb.NewProcessedUnshieldRequestBatchWithValue(
		BatchID1,
		[]string{"txid1", "txid2", "txid3"},
		map[string][]*statedb.UTXO{
			btcMultiSigAddress: {
				statedb.NewUTXOWithValue(btcMultiSigAddress, "6a3b123367bdcd6aaf61f391d4158cdaa6f34ee6c4d52d3a9d57920e683c396c", 2, 224372),
			},
		},
		map[uint64]uint{
			900: 900,
		},
	)

	processUnshield2 := statedb.NewProcessedUnshieldRequestBatchWithValue(
		BatchID2,
		[]string{"txid4", "txid5"},
		map[string][]*statedb.UTXO{
			btcMultiSigAddress: {
				statedb.NewUTXOWithValue(btcMultiSigAddress, "163a6cc24df4efbd5c997aa623d4e319f1b7671be83a86bb0fa27bc701ae4a76", 1, 1000000),
			},
		},
		map[uint64]uint{
			1000: 1000,
		},
	)

	processedUnshieldRequests := map[string]map[string]*statedb.ProcessedUnshieldRequestBatch{
		pv4Common.PortalBTCIDStr: {
			keyBatchShield1: processUnshield1,
			keyBatchShield2: processUnshield2,
		},
	}

	s.currentPortalStateForProducer.ProcessedUnshieldRequests = processedUnshieldRequests
	s.currentPortalStateForProducer.UTXOs = utxos
	s.currentPortalStateForProcess.ProcessedUnshieldRequests = CloneUnshieldBatchRequests(processedUnshieldRequests)
	s.currentPortalStateForProcess.UTXOs = CloneUTXOs(utxos)
}

func buildSubmitConfirmedTxActionsFromTcs(tcs []TestCaseSubmitConfirmedTx, shardID byte) []portalV4InstForProducer {
	insts := []portalV4InstForProducer{}

	for _, tc := range tcs {
		inst := buildPortalSubmitConfirmedTxAction(
			tc.confirmedTxProof,
			tc.tokenID,
			tc.batchID,
			shardID,
		)
		optionalData := make(map[string]interface{})
		outputs := make(map[string]uint64, 0)
		for _, v := range tc.outputs {
			outputs[v.externalAddress] = v.amount
		}
		insts = append(insts, portalV4InstForProducer{
			inst:         inst,
			optionalData: optionalData,
		})
	}

	return insts
}

func buildPortalSubmitConfirmedTxAction(
	unshieldProof string,
	tokenID string,
	batchID string,
	shardID byte,
) []string {
	data := pv4Meta.PortalSubmitConfirmedTxRequest{
		MetadataBase: basemeta.MetadataBase{
			Type: basemeta.PortalSubmitConfirmedTxMeta,
		},
		UnshieldProof: unshieldProof,
		TokenID:       tokenID,
		BatchID:       batchID,
	}

	actionContent := pv4Meta.PortalSubmitConfirmedTxAction{
		Meta:    data,
		TxReqID: common.Hash{},
		ShardID: shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(basemeta.PortalSubmitConfirmedTxMeta), actionContentBase64Str}
}

func buildExpectedResultSubmitConfirmedTx(s *PortalTestSuiteV4) ([]TestCaseSubmitConfirmedTx, *ExpectedResultSubmitConfirmedTx) {

	testcases := []TestCaseSubmitConfirmedTx{
		// request submit external confirmed tx
		{
			batchID:          BatchID1,
			confirmedTxProof: confirmedTxProof1,
			tokenID:          pv4Common.PortalBTCIDStr,
			outputs: []OutPut{
				{
					externalAddress: "bc1qqyxfxeh6k5kt29e30pzhxs7kd59fvr76u95qat",
					amount:          100,
				},
				{
					externalAddress: "bc1qj9dgez2sstg8d06ehjgw6wf4hsjxr3aake0dzs",
					amount:          100,
				},
			},
		},
		// submit existed proof
		{
			batchID:          BatchID1,
			confirmedTxProof: confirmedTxProof1,
			tokenID:          pv4Common.PortalBTCIDStr,
			outputs: []OutPut{
				{
					externalAddress: "bc1qqyxfxeh6k5kt29e30pzhxs7kd59fvr76u95qat",
					amount:          100,
				},
				{
					externalAddress: "bc1qj9dgez2sstg8d06ehjgw6wf4hsjxr3aake0dzs",
					amount:          100,
				},
			},
		},
		// request submit proof with non-exist batchID
		{
			batchID:          BatchID3,
			confirmedTxProof: confirmedTxProof2,
			tokenID:          pv4Common.PortalBTCIDStr,
			outputs: []OutPut{
				{
					externalAddress: "bc1qqyxfxeh6k5kt29e30pzhxs7kd59fvr76u95qat",
					amount:          100,
				},
				{
					externalAddress: "bc1qj9dgez2sstg8d06ehjgw6wf4hsjxr3aake0dzs",
					amount:          100,
				},
			},
		},
		// request submit wrong proof
		{
			batchID:          BatchID2,
			confirmedTxProof: confirmedTxProof3,
			tokenID:          pv4Common.PortalBTCIDStr,
			outputs: []OutPut{
				{
					externalAddress: "bc1qqyxfxeh6k5kt29e30pzhxs7kd59fvr76u95qat",
					amount:          100,
				},
				{
					externalAddress: "bc1qj9dgez2sstg8d06ehjgw6wf4hsjxr3aake0dzs",
					amount:          100,
				},
			},
		},
	}

	btcMultiSigAddress := s.portalParams.MultiSigAddresses[pv4Common.PortalBTCIDStr]
	processUnshield2 := statedb.NewProcessedUnshieldRequestBatchWithValue(
		BatchID2,
		[]string{"txid4", "txid5"},
		map[string][]*statedb.UTXO{
			btcMultiSigAddress: {
				statedb.NewUTXOWithValue(btcMultiSigAddress, "163a6cc24df4efbd5c997aa623d4e319f1b7671be83a86bb0fa27bc701ae4a76", 1, 1000000),
			},
		},
		map[uint64]uint{
			1000: 1000,
			1500: 1500,
		},
	)

	processedUnshieldRequests := map[string]map[string]*statedb.ProcessedUnshieldRequestBatch{
		pv4Common.PortalBTCIDStr: {
			keyBatchShield2: processUnshield2,
		},
	}

	// build expected results
	expectedRes := &ExpectedResultSubmitConfirmedTx{
		processedUnshieldRequests: processedUnshieldRequests,
		numBeaconInsts:            4,
		statusInsts: []string{
			pCommon.PortalRequestRejectedChainStatus,
			pCommon.PortalRequestRejectedChainStatus,
			pCommon.PortalRequestAcceptedChainStatus,
			pCommon.PortalRequestRejectedChainStatus,
		},
	}

	return testcases, expectedRes
}

func (s *PortalTestSuiteV4) TestSubmitConfirmedTx() {
	fmt.Println("Running TestUnlockOverRateCollaterals - beacon height 1501 ...")
	bc := s.blockChain
	beaconHeight := uint64(1501)
	shardHeights := map[byte]uint64{
		0: uint64(1501),
	}
	shardID := byte(0)
	pm := portalprocessv4.NewPortalV4Manager()

	s.SetupTestSubmitConfirmedTx()

	unshieldBatchPool := s.currentPortalStateForProducer.ProcessedUnshieldRequests
	for key, unshieldBatch := range unshieldBatchPool {
		fmt.Printf("cusKey %v - custodian address: %v\n", key, unshieldBatch)
	}

	testcases, expectedResult := buildExpectedResultSubmitConfirmedTx(s)

	// build actions from testcases
	instsForProducer := buildSubmitConfirmedTxActionsFromTcs(testcases, shardID)

	newInsts, err := producerPortalInstructionsV4(
		bc, beaconHeight-1, shardHeights, instsForProducer, &s.currentPortalStateForProducer, s.portalParams, shardID, pm)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructionsV4(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, pm)

	// check results
	s.Equal(expectedResult.numBeaconInsts, uint(len(newInsts)))
	s.Equal(nil, err)

	for i, inst := range newInsts {
		s.Equal(expectedResult.statusInsts[i], inst[2], "Instruction index %v", i)
	}

	s.Equal(expectedResult.processedUnshieldRequests, s.currentPortalStateForProducer.ProcessedUnshieldRequests)

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

func TestPortalSuiteV4(t *testing.T) {
	suite.Run(t, new(PortalTestSuiteV4))
}

// util functions
func CloneUnshieldBatchRequests(processedUnshieldRequestBatch map[string]map[string]*statedb.ProcessedUnshieldRequestBatch) map[string]map[string]*statedb.ProcessedUnshieldRequestBatch {
	newReqs := make(map[string]map[string]*statedb.ProcessedUnshieldRequestBatch, len(processedUnshieldRequestBatch))
	for key, batch := range processedUnshieldRequestBatch {
		newBatch := make(map[string]*statedb.ProcessedUnshieldRequestBatch, len(batch))
		for key2, batch2 := range batch {
			newBatch[key2] = statedb.NewProcessedUnshieldRequestBatchWithValue(
				batch2.GetBatchID(),
				batch2.GetUnshieldRequests(),
				batch2.GetUTXOs(),
				batch2.GetExternalFees(),
			)
		}
		newReqs[key] = newBatch
	}
	return newReqs
}

func CloneUTXOs(utxos map[string]map[string]*statedb.UTXO) map[string]map[string]*statedb.UTXO {
	newReqs := make(map[string]map[string]*statedb.UTXO, len(utxos))
	for key, batch := range utxos {
		newBatch := make(map[string]*statedb.UTXO, len(batch))
		for key2, batch2 := range batch {
			newBatch[key2] = statedb.NewUTXOWithValue(
				batch2.GetWalletAddress(),
				batch2.GetTxHash(),
				batch2.GetOutputIndex(),
				batch2.GetOutputAmount(),
			)
		}
		newReqs[key] = newBatch
	}
	return newReqs
}
