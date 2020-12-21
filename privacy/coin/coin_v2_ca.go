package coin

import(
	"errors"
	"fmt"

	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/wallet"
)

const MAX_TRIES_OTA int = 50000

func (coin *CoinV2) ComputeCommitmentCA() (*operation.Point,error){
	if coin==nil || coin.GetRandomness()==nil || coin.GetAmount()==nil{
		return nil, errors.New("missing arguments for committing")
	}
	gRan_immutable := operation.PedCom.G[operation.PedersenRandomnessIndex]
	commitment := new(operation.Point).ScalarMult(coin.GetAssetTag(),coin.GetAmount())
	commitment.Add(commitment,new(operation.Point).ScalarMult(gRan_immutable,coin.GetRandomness()))
	return commitment,nil
}

func ComputeCommitmentCA(assetTag *operation.Point, r, v *operation.Scalar) (*operation.Point,error){
	if assetTag==nil || r==nil || v==nil{
		return nil, errors.New("missing arguments for committing to CA coin")
	}
	gRan_immutable := operation.PedCom.G[operation.PedersenRandomnessIndex]
	commitment := new(operation.Point).ScalarMult(assetTag,v)
	commitment.Add(commitment,new(operation.Point).ScalarMult(gRan_immutable,r))
	return commitment,nil
}

func ComputeAssetTagBlinder(sharedSecret *operation.Point) (*operation.Scalar,error){
	if sharedSecret==nil{
		return nil, errors.New("Missing arguments for asset tag blinder")
	}
	blinder := operation.HashToScalar(append(sharedSecret.ToBytesS(), []byte("assettag")...))
	return blinder, nil
}

// this should be an input coin
func (coin *CoinV2) RecomputeSharedSecret(privateKey []byte) (*operation.Point,error){
	// sk := new(operation.Scalar).FromBytesS(privateKey)
	var privOTA []byte = key.GeneratePrivateOTAKey(privateKey)[:]
	sk := new(operation.Scalar).FromBytesS(privOTA)
	// this is g^SharedRandom, previously created by sender of the coin
	sharedOTARandomPoint, err := coin.GetTxRandom().GetTxOTARandomPoint()
	if err != nil {
		return nil, errors.New("Cannot retrieve tx random detail")
	}
	sharedSecret := new(operation.Point).ScalarMult(sharedOTARandomPoint, sk)
	return sharedSecret, nil
}

func (coin *CoinV2) ValidateAssetTag(sharedSecret *operation.Point, tokenID *common.Hash) (bool, error){
	if coin.GetAssetTag()==nil{
		if tokenID==nil || *tokenID==common.PRVCoinID{
			// a valid PRV coin
			return true, nil
		}
		return false, errors.New("CA coin must have asset tag")
	}
	if tokenID==nil || *tokenID==common.PRVCoinID{
		// invalid
		return false, errors.New("PRV coin cannot have asset tag")
	}
	recomputedAssetTag := operation.HashToPoint(tokenID[:])
	if operation.IsPointEqual(recomputedAssetTag, coin.GetAssetTag()){
		return true, nil
	}

	blinder, err := ComputeAssetTagBlinder(sharedSecret)
	if err != nil {
		return false, err
	}
	
	recomputedAssetTag.Add(recomputedAssetTag, new(operation.Point).ScalarMult(operation.PedCom.G[PedersenRandomnessIndex], blinder))
	if operation.IsPointEqual(recomputedAssetTag, coin.GetAssetTag()){
		return true, nil
	}
	return false, nil
}

func (coin *CoinV2) SetPlainTokenID(tokenID *common.Hash) error{
	assetTag := operation.HashToPoint(tokenID[:])
	coin.SetAssetTag(assetTag)
	com, err := coin.ComputeCommitmentCA()
	if err != nil{
		return err
	}
	coin.SetCommitment(com)
	return nil
}

// for confidential asset only
func NewCoinCA(info *key.PaymentInfo, tokenID *common.Hash) (*CoinV2, *operation.Point, error) {
	receiverPublicKey, err := new(operation.Point).FromBytesS(info.PaymentAddress.Pk)
	if err != nil {
		errStr := fmt.Sprintf("Cannot parse outputCoinV2 from PaymentInfo when parseByte PublicKey, error %v ", err)
		return nil, nil, errors.New(errStr)
	}
	receiverPublicKeyBytes := receiverPublicKey.ToBytesS()
	targetShardID := common.GetShardIDFromLastByte(receiverPublicKeyBytes[len(receiverPublicKeyBytes)-1])

	c := new(CoinV2).Init()
	// Amount, Randomness, SharedRandom is transparency until we call concealData
	c.SetAmount(new(operation.Scalar).FromUint64(info.Amount))
	c.SetRandomness(operation.RandomScalar())
	c.SetSharedRandom(operation.RandomScalar()) // r
	c.SetSharedConcealRandom(operation.RandomScalar())
	c.SetInfo(info.Message)

	// If this is going to burning address then dont need to create ota
	if wallet.IsPublicKeyBurningAddress(info.PaymentAddress.Pk) {
		publicKey, err := new(operation.Point).FromBytesS(info.PaymentAddress.Pk)
		if err != nil {
			panic("Something is wrong with info.paymentAddress.pk, burning address should be a valid point")
		}
		c.SetPublicKey(publicKey)
		err = c.SetPlainTokenID(tokenID)
		if err!=nil{
			return nil, nil, err
		}
		return c, nil, nil
	}

	// Increase index until have the right shardID
	index := uint32(0)
	publicOTA := info.PaymentAddress.GetOTAPublicKey() //For generating one-time-address
	publicSpend := info.PaymentAddress.GetPublicSpend() //General public key
	//publicView := info.PaymentAddress.GetPublicView() //For generating asset tag and concealing output coin

	rK := new(operation.Point).ScalarMult(publicOTA, c.GetSharedRandom())
	for i:=MAX_TRIES_OTA;i>0;i--{
		index += 1

		// Get publickey
		hash := operation.HashToScalar(append(rK.ToBytesS(), common.Uint32ToBytes(index)...))
		HrKG := new(operation.Point).ScalarMultBase(hash)
		publicKey := new(operation.Point).Add(HrKG, publicSpend)
		c.SetPublicKey(publicKey)

		currentShardID, err := c.GetShardID()
		if err != nil {
			return nil, nil, err
		}
		if currentShardID == targetShardID {
			otaSharedRandomPoint := new(operation.Point).ScalarMultBase(c.GetSharedRandom())
			concealSharedRandomPoint := new(operation.Point).ScalarMultBase(c.GetSharedConcealRandom())
			c.SetTxRandomDetail(concealSharedRandomPoint, otaSharedRandomPoint, index)

			rAsset := new(operation.Point).ScalarMult(publicOTA, c.GetSharedRandom())
			blinder,_ := ComputeAssetTagBlinder(rAsset)
			if tokenID == nil {
				return nil, nil, errors.New("Cannot create coin without tokenID")
			}
			assetTag := operation.HashToPoint(tokenID[:])
			assetTag.Add(assetTag,new(operation.Point).ScalarMult(operation.PedCom.G[PedersenRandomnessIndex],blinder))
			c.SetAssetTag(assetTag)
			// fmt.Printf("Shared secret is %s\n", string(rK.MarshalText()))
			// fmt.Printf("Blinder is %s\n", string(blinder.MarshalText()))
			// fmt.Printf("Asset tag is %s\n", string(assetTag.MarshalText()))
			com, err := c.ComputeCommitmentCA()
			if err != nil{
				return nil, nil, errors.New("Cannot compute commitment for confidential asset")
			}
			c.SetCommitment(com)
			
			return c, rAsset, nil
		}
	}
	// MAX_TRIES_OTA could be exceeded if the OS's RNG or the statedb is corrupted
	return nil, nil, errors.New(fmt.Sprintf("Cannot create OTA after %d attempts", MAX_TRIES_OTA))
}