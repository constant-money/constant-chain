package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	rCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata/rpccaller"
	"github.com/pkg/errors"
)

type IssuingETHRequest struct {
	BlockHash  rCommon.Hash
	TxIndex    uint
	ProofStrs  []string
	IncTokenID common.Hash
	MetadataBase
}

type IssuingETHReqAction struct {
	Meta       IssuingETHRequest `json:"meta"`
	TxReqID    common.Hash       `json:"txReqId"`
	ETHReceipt *types.Receipt    `json:"ethReceipt"`
}

type IssuingETHAcceptedInst struct {
	ShardID         byte        `json:"shardId"`
	IssuingAmount   uint64      `json:"issuingAmount"`
	ReceiverAddrStr string      `json:"receiverAddrStr"`
	IncTokenID      common.Hash `json:"incTokenId"`
	TxReqID         common.Hash `json:"txReqId"`
	UniqETHTx       []byte      `json:"uniqETHTx"`
	ExternalTokenID []byte      `json:"externalTokenId"`
}

type GetETHHeaderByHashRes struct {
	rpccaller.RPCBaseRes
	Result *types.Header `json:"result"`
}

type GetETHHeaderByNumberRes struct {
	rpccaller.RPCBaseRes
	Result *types.Header `json:"result"`
}

type GetETHBlockNumRes struct {
	rpccaller.RPCBaseRes
	Result string `json:"result"`
}

func ParseETHIssuingInstContent(instContentStr string) (*IssuingETHReqAction, error) {
	contentBytes, err := base64.StdEncoding.DecodeString(instContentStr)
	if err != nil {
		return nil, NewMetadataTxError(IssuingEthRequestDecodeInstructionError, err)
	}
	var issuingETHReqAction IssuingETHReqAction
	err = json.Unmarshal(contentBytes, &issuingETHReqAction)
	if err != nil {
		return nil, NewMetadataTxError(IssuingEthRequestUnmarshalJsonError, err)
	}
	return &issuingETHReqAction, nil
}

func NewIssuingETHRequest(
	blockHash rCommon.Hash,
	txIndex uint,
	proofStrs []string,
	incTokenID common.Hash,
	metaType int,
) (*IssuingETHRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	issuingETHReq := &IssuingETHRequest{
		BlockHash:  blockHash,
		TxIndex:    txIndex,
		ProofStrs:  proofStrs,
		IncTokenID: incTokenID,
	}
	issuingETHReq.MetadataBase = metadataBase
	return issuingETHReq, nil
}

func NewIssuingETHRequestFromMap(
	data map[string]interface{},
) (*IssuingETHRequest, error) {
	blockHash := rCommon.HexToHash(data["BlockHash"].(string))
	txIdx := uint(data["TxIndex"].(float64))
	proofsRaw := data["ProofStrs"].([]interface{})
	proofStrs := []string{}
	for _, item := range proofsRaw {
		proofStrs = append(proofStrs, item.(string))
	}

	incTokenID, err := common.Hash{}.NewHashFromStr(data["IncTokenID"].(string))
	if err != nil {
		return nil, NewMetadataTxError(IssuingEthRequestNewIssuingETHRequestFromMapEror, errors.Errorf("TokenID incorrect"))
	}

	req, _ := NewIssuingETHRequest(
		blockHash,
		txIdx,
		proofStrs,
		*incTokenID,
		IssuingETHRequestMeta,
	)
	return req, nil
}

func (iReq IssuingETHRequest) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	ethReceipt, err := iReq.verifyProofAndParseReceipt()
	if err != nil {
		return false, NewMetadataTxError(IssuingEthRequestValidateTxWithBlockChainError, err)
	}
	if ethReceipt == nil {
		return false, errors.Errorf("The eth proof's receipt could not be null.")
	}

	// check this is a normal pToken
	if statedb.PrivacyTokenIDExisted(transactionStateDB, iReq.IncTokenID) {
		isBridgeToken, err := statedb.IsBridgeTokenExistedByType(beaconViewRetriever.GetBeaconFeatureStateDB(), iReq.IncTokenID, false)
		if !isBridgeToken {
			if err != nil {
				return false, NewMetadataTxError(InvalidMeta, err)
			} else {
				return false, NewMetadataTxError(InvalidMeta, errors.New("token is invalid"))
			}
		}
	}

	return true, nil
}

func (iReq IssuingETHRequest) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	if len(iReq.ProofStrs) == 0 {
		return false, false, NewMetadataTxError(IssuingEthRequestValidateSanityDataError, errors.New("Wrong request info's proof"))
	}
	return true, true, nil
}

func (iReq IssuingETHRequest) ValidateMetadataByItself() bool {
	if iReq.Type != IssuingETHRequestMeta {
		return false
	}
	return true
}

func (iReq IssuingETHRequest) Hash() *common.Hash {
	record := iReq.BlockHash.String()
	// TODO: @hung change to record += fmt.Sprint(iReq.TxIndex)
	record += string(iReq.TxIndex)
	proofStrs := iReq.ProofStrs
	for _, proofStr := range proofStrs {
		record += proofStr
	}
	record += iReq.MetadataBase.Hash().String()
	record += iReq.IncTokenID.String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iReq *IssuingETHRequest) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
	ethReceipt, err := iReq.verifyProofAndParseReceipt()
	if err != nil {
		return [][]string{}, NewMetadataTxError(IssuingEthRequestBuildReqActionsError, err)
	}
	if ethReceipt == nil {
		return [][]string{}, NewMetadataTxError(IssuingEthRequestBuildReqActionsError, errors.Errorf("The eth proof's receipt could not be null."))
	}
	txReqID := *(tx.Hash())
	actionContent := map[string]interface{}{
		"meta":       *iReq,
		"txReqId":    txReqID,
		"ethReceipt": *ethReceipt,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, NewMetadataTxError(IssuingEthRequestBuildReqActionsError, err)
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(IssuingETHRequestMeta), actionContentBase64Str}

	//err = statedb.TrackBridgeReqWithStatus(bcr.GetBeaconFeatureStateDB(), txReqID, byte(common.BridgeRequestProcessingStatus))
	//if err != nil {
	//	return [][]string{}, NewMetadataTxError(IssuingEthRequestBuildReqActionsError, err)
	//}
	return [][]string{action}, nil
}

func (iReq *IssuingETHRequest) CalculateSize() uint64 {
	return calculateSize(iReq)
}

func (iReq *IssuingETHRequest) verifyProofAndParseReceipt() (*types.Receipt, error) {
	ethHeader, err := GetETHHeader(iReq.BlockHash)
	if err != nil {
		return nil, NewMetadataTxError(IssuingEthRequestVerifyProofAndParseReceipt, err)
	}
	if ethHeader == nil {
		Logger.log.Info("WARNING: Could not find out the ETH block header with the hash: ", iReq.BlockHash)
		return nil, NewMetadataTxError(IssuingEthRequestVerifyProofAndParseReceipt, errors.Errorf("WARNING: Could not find out the ETH block header with the hash: %s", iReq.BlockHash.String()))
	}

	mostRecentBlkNum, err := GetMostRecentETHBlockHeight()
	if err != nil {
		Logger.log.Info("WARNING: Could not find the most recent block height on Ethereum")
		return nil, NewMetadataTxError(IssuingEthRequestVerifyProofAndParseReceipt, err)
	}

	if mostRecentBlkNum.Cmp(big.NewInt(0).Add(ethHeader.Number, big.NewInt(ETHConfirmationBlocks))) == -1 {
		errMsg := fmt.Sprintf("WARNING: It needs 15 confirmation blocks for the process, the requested block (%s) but the latest block (%s)", ethHeader.Number.String(), mostRecentBlkNum.String())
		Logger.log.Info(errMsg)
		return nil, NewMetadataTxError(IssuingEthRequestVerifyProofAndParseReceipt, errors.New(errMsg))
	}

	keybuf := new(bytes.Buffer)
	keybuf.Reset()
	rlp.Encode(keybuf, iReq.TxIndex)

	nodeList := new(light.NodeList)
	for _, proofStr := range iReq.ProofStrs {
		proofBytes, err := base64.StdEncoding.DecodeString(proofStr)
		if err != nil {
			return nil, err
		}
		nodeList.Put([]byte{}, proofBytes)
	}
	proof := nodeList.NodeSet()
	val, _, err := trie.VerifyProof(ethHeader.ReceiptHash, keybuf.Bytes(), proof)
	if err != nil {
		fmt.Printf("WARNING: ETH issuance proof verification failed: %v", err)
		return nil, NewMetadataTxError(IssuingEthRequestVerifyProofAndParseReceipt, err)
	}
	// Decode value from VerifyProof into Receipt
	constructedReceipt := new(types.Receipt)
	err = rlp.DecodeBytes(val, constructedReceipt)
	if err != nil {
		return nil, NewMetadataTxError(IssuingEthRequestVerifyProofAndParseReceipt, err)
	}

	if constructedReceipt.Status != types.ReceiptStatusSuccessful {
		return nil, NewMetadataTxError(IssuingEthRequestVerifyProofAndParseReceipt, errors.New("The constructedReceipt's status is not success"))
	}

	return constructedReceipt, nil
}

func ParseETHLogData(data []byte) (map[string]interface{}, error) {
	abiIns, err := abi.JSON(strings.NewReader(common.AbiJson))
	if err != nil {
		return nil, err
	}
	dataMap := map[string]interface{}{}
	if err = abiIns.UnpackIntoMap(dataMap, "Deposit", data); err != nil {
		return nil, NewMetadataTxError(UnexpectedError, err)
	}
	return dataMap, nil
}

func IsETHTxHashUsedInBlock(uniqETHTx []byte, uniqETHTxsUsed [][]byte) bool {
	for _, item := range uniqETHTxsUsed {
		if bytes.Equal(uniqETHTx, item) {
			return true
		}
	}
	return false
}

func GetETHHeader(
	ethBlockHash rCommon.Hash,
) (*types.Header, error) {
	rpcClient := rpccaller.NewRPCClient()
	getETHHeaderByHashParams := []interface{}{ethBlockHash, false}
	var getETHHeaderByHashRes GetETHHeaderByHashRes
	err := rpcClient.RPCCall(
		EthereumLightNodeProtocol,
		EthereumLightNodeHost,
		EthereumLightNodePort,
		"eth_getBlockByHash",
		getETHHeaderByHashParams,
		&getETHHeaderByHashRes,
	)
	if err != nil {
		return nil, err
	}
	if getETHHeaderByHashRes.RPCError != nil {
		Logger.log.Infof("WARNING: an error occured during calling eth_getBlockByHash: %s", getETHHeaderByHashRes.RPCError.Message)
		return nil, errors.New(fmt.Sprintf("An error occured during calling eth_getBlockByHash: %s", getETHHeaderByHashRes.RPCError.Message))
	}

	ethHeaderByHash := getETHHeaderByHashRes.Result
	headerNum := ethHeaderByHash.Number

	getETHHeaderByNumberParams := []interface{}{fmt.Sprintf("0x%x", headerNum), false}
	var getETHHeaderByNumberRes GetETHHeaderByNumberRes
	err = rpcClient.RPCCall(
		EthereumLightNodeProtocol,
		EthereumLightNodeHost,
		EthereumLightNodePort,
		"eth_getBlockByNumber",
		getETHHeaderByNumberParams,
		&getETHHeaderByNumberRes,
	)
	if err != nil {
		return nil, err
	}
	if getETHHeaderByNumberRes.RPCError != nil {
		Logger.log.Infof("WARNING: an error occured during calling eth_getBlockByNumber: %s", getETHHeaderByNumberRes.RPCError.Message)
		return nil, errors.New(fmt.Sprintf("An error occured during calling eth_getBlockByNumber: %s", getETHHeaderByNumberRes.RPCError.Message))
	}

	ethHeaderByNum := getETHHeaderByNumberRes.Result
	if ethHeaderByNum.Hash().String() != ethHeaderByHash.Hash().String() {
		return nil, errors.New(fmt.Sprintf("The requested eth BlockHash is being on fork branch, rejected!"))
	}
	return ethHeaderByHash, nil
}

// GetMostRecentETHBlockHeight get most recent block height on Ethereum
func GetMostRecentETHBlockHeight() (*big.Int, error) {
	rpcClient := rpccaller.NewRPCClient()
	params := []interface{}{}
	var getETHBlockNumRes GetETHBlockNumRes
	err := rpcClient.RPCCall(
		EthereumLightNodeProtocol,
		EthereumLightNodeHost,
		EthereumLightNodePort,
		"eth_blockNumber",
		params,
		&getETHBlockNumRes,
	)
	if err != nil {
		return nil, err
	}
	if getETHBlockNumRes.RPCError != nil {
		return nil, errors.New(fmt.Sprintf("an error occured during calling eth_blockNumber: %s", getETHBlockNumRes.RPCError.Message))
	}

	blockNumber := new(big.Int)
	_, ok := blockNumber.SetString(getETHBlockNumRes.Result[2:], 16)
	if !ok {
		return nil, errors.New("Cannot convert blockNumber into integer")
	}
	return blockNumber, nil
}

func PickAndParseLogMapFromReceipt(constructedReceipt *types.Receipt, ethContractAddressStr string) (map[string]interface{}, error) {
	logData := []byte{}
	logLen := len(constructedReceipt.Logs)
	if logLen == 0 {
		Logger.log.Debug("WARNING: LOG data is invalid.")
		return nil, nil
	}
	for _, log := range constructedReceipt.Logs {
		if bytes.Equal(rCommon.HexToAddress(ethContractAddressStr).Bytes(), log.Address.Bytes()) {
			logData = log.Data
			break
		}
	}
	if len(logData) == 0 {
		return nil, nil
	}
	return ParseETHLogData(logData)
}
