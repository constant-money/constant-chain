package bnb

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/types"
	"strings"
)

// Mainnet - Binance

// address = hexEncode(sha256(pubKey)[:20])
var MainnetValidatorAddresses = []string{
	"1175946A48EAA473868A0A6F52E6C66CCAF472EA",
	"14CFCE69B645F3F88BAF08EA5B77FA521E4480F9",
	"17B42E8F284D3CA0E420262F89CD76C749BB12C9",
	"3CD4AABABDDEB7ABFEA9618732E331077A861D2B",
	"414FB3BBA216AF84C47E07D6EBAA2DCFC3563A2F",
	"71F253E6FEA9EDD4B4753F5483549FE4F0F3A21C",
	"7235EF143D20FC0ABC427615D83014BB02D7C06C",
	"A71E5CD078B8C5C7B1AF88BCE84DD70B0557D93E",
	"A9157B3FA6EB4C1E396B9B746E95327A07DC42E5",
	"B0FBB52FF7EE93CC476DFE6B74FA1FC88584F30D",
	"B7707D9F593C62E85BB9E1A2366D12A97CD5DFF2",
}

// public key on ed25519 curve (base64 encoded)
var MainnetValidatorB64EncodePubKeys = []string{
	"03adih94tMF6ll96MNQYH6u9H5afRtPI6Dta1IRUIdg=",
	"K6ToFUL0N7euH4o13bIzx4mo3CJzQ3fZttY68cpAO2E=",
	"342oxav9s4WVORMIu3HloeCqvcHQzzgxXVDWvpObJgY=",
	"tmGe3KQUNISAAoHWmLcMk16RUq1Xsx2FwF8vefZLOfM=",
	"lEbRSthsjS10eAsIRxEAAaHC4lLu3+pHU+u7/OOiL1I=",
	"A1PGOfgMyAFZRENtqxAyJF1E+RLtwx72aP+fSkXNBZk=",
	"6B03l+BUTDpxjh8F8Pt4IhLiSOeEwahRvofneuDbIw4=",
	"Xj/NowvRnUXEtzaI2jXn2h/OfGhZssHyDtUgLSQUTj4=",
	"sGpZotdb9dAU/OfJmbXnHnqWCHD3JYR9S6MjW66qCO8=",
	"DJEOL+ZQ5OAUBrMxC0iftgqEvD/1xb7jpW1YmLaorzI=",
	"cfLXuOwci5mmU0KbARjNIB95T0CdD+pNZbG2YvKwAGM=",
}

// MainnetValidatorPubKeyBytes are results from base-64 decoding MainnetValidatorB64EncodePubKeys
var MainnetValidatorPubKeyBytes = [][]byte{
	{211, 118, 157, 138, 31, 120, 180, 193, 122, 150, 95, 122, 48, 212, 24, 31, 171, 189, 31, 150, 159, 70, 211, 200, 232, 59, 90, 212, 132, 84, 33, 216},
	{43, 164, 232, 21, 66, 244, 55, 183, 174, 31, 138, 53, 221, 178, 51, 199, 137, 168, 220, 34, 115, 67, 119, 217, 182, 214, 58, 241, 202, 64, 59, 97},
	{223, 141, 168, 197, 171, 253, 179, 133, 149, 57, 19, 8, 187, 113, 229, 161, 224, 170, 189, 193, 208, 207, 56, 49, 93, 80, 214, 190, 147, 155, 38, 6},
	{182, 97, 158, 220, 164, 20, 52, 132, 128, 2, 129, 214, 152, 183, 12, 147, 94, 145, 82, 173, 87, 179, 29, 133, 192, 95, 47, 121, 246, 75, 57, 243},
	{148, 70, 209, 74, 216, 108, 141, 45, 116, 120, 11, 8, 71, 17, 0, 1, 161, 194, 226, 82, 238, 223, 234, 71, 83, 235, 187, 252, 227, 162, 47, 82},
	{3, 83, 198, 57, 248, 12, 200, 1, 89, 68, 67, 109, 171, 16, 50, 36, 93, 68, 249, 18, 237, 195, 30, 246, 104, 255, 159, 74, 69, 205, 5, 153},
	{232, 29, 55, 151, 224, 84, 76, 58, 113, 142, 31, 5, 240, 251, 120, 34, 18, 226, 72, 231, 132, 193, 168, 81, 190, 135, 231, 122, 224, 219, 35, 14},
	{94, 63, 205, 163, 11, 209, 157, 69, 196, 183, 54, 136, 218, 53, 231, 218, 31, 206, 124, 104, 89, 178, 193, 242, 14, 213, 32, 45, 36, 20, 78, 62},
	{176, 106, 89, 162, 215, 91, 245, 208, 20, 252, 231, 201, 153, 181, 231, 30, 122, 150, 8, 112, 247, 37, 132, 125, 75, 163, 35, 91, 174, 170, 8, 239},
	{12, 145, 14, 47, 230, 80, 228, 224, 20, 6, 179, 49, 11, 72, 159, 182, 10, 132, 188, 63, 245, 197, 190, 227, 165, 109, 88, 152, 182, 168, 175, 50},
	{113, 242, 215, 184, 236, 28, 139, 153, 166, 83, 66, 155, 1, 24, 205, 32, 31, 121, 79, 64, 157, 15, 234, 77, 101, 177, 182, 98, 242, 176, 0, 99},
}

var MainnetValidatorVotingPowers = []int64{
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
}

// SHA256 returns the SHA256 of the bz.
// todo: need to be moved to common package
func SHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// DecodePublicKeyValidator decodes encoded public key to public key in bytes array
func DecodePublicKeyValidator() error {
	MainnetValidatorPubKeyBytes = make([][]byte, len(MainnetValidatorB64EncodePubKeys))
	for i, item := range MainnetValidatorB64EncodePubKeys {
		bytes, err := base64.StdEncoding.DecodeString(item)
		if err != nil {
			return err
		}

		// check public key bytes to address
		pubKeyHash := SHA256(bytes)
		addTmpStr := strings.ToUpper(hex.EncodeToString(pubKeyHash[0:20]))

		if addTmpStr == MainnetValidatorAddresses[i] {
			MainnetValidatorPubKeyBytes[i] = bytes
		} else {
			fmt.Printf("Public key is wrong %v\n", i)
		}
	}

	fmt.Printf("MainnetValidatorPubKeyBytes %#v\n", MainnetValidatorPubKeyBytes)
	return nil
}

// Local

// address = hexEncode(sha256(pubKey)[:20])
//var ValidatorAddressesTestnet = []string{
//	"87E733422966685C1B24F42A3184AC959EC49A4C",
//}
//
//// public key on ed25519 curve (base64 encoded)
//var ValidatorB64EncodePubKeysTestnet = []string{
//	"uND4Li1FIzpmjmEe9RZGZlKr53zLP8ZHUP8DSQCZpN4=",
//}
//
//// MainnetValidatorPubKeyBytes are results from base-64 decoding MainnetValidatorB64EncodePubKeys
//var ValidatorPubKeyBytesTestnet = [][]byte{
//	[]byte{0xb8, 0xd0, 0xf8, 0x2e, 0x2d, 0x45, 0x23, 0x3a, 0x66, 0x8e, 0x61, 0x1e, 0xf5, 0x16, 0x46, 0x66, 0x52, 0xab, 0xe7, 0x7c, 0xcb, 0x3f, 0xc6, 0x47, 0x50, 0xff, 0x3, 0x49, 0x0, 0x99, 0xa4, 0xde},
//}
//
//var ValidatorVotingPowersTestnet = []int64{
//	1000000000000,
//}

// Testnet
var ValidatorAddressesTestnet = []string{
	"06FD60078EB4C2356137DD50036597DB267CF616",
	"18E69CC672973992BB5F76D049A5B2C5DDF77436",
	"344C39BB8F4512D6CAB1F6AAFAC1811EF9D8AFDF",
	"37EF19AF29679B368D2B9E9DE3F8769B35786676",
	"62633D9DB7ED78E951F79913FDC8231AA77EC12B",
	"91844D296BD8E591448EFC65FD6AD51A888D58FA",
	"9CCDDD479C0AD8DCD01D754DD95FE15384E8BBDC",
	"B3727172CE6473BC780298A2D66C12F1A14F5B2A",
	"B6F20C7FAA2B2F6F24518FA02B71CB5F4A09FBA3",
	"D4CECEF238E778C7063552C3A7AB95DA35C3FB47",
	"F42F1D05AC568D12E26B9655395E7FBBD46BC5BB",
}

// MainnetValidatorPubKeyBytes are results from base-64 decoding MainnetValidatorB64EncodePubKeys
var ValidatorPubKeyBytesTestnet = [][]byte{
	{225, 124, 190, 156, 32, 205, 207, 223, 135, 107, 59, 18, 151, 141, 50, 100, 160, 7, 252, 170, 167, 28, 76, 219, 112, 29, 158, 188, 3, 35, 244, 79},
	{24, 78, 123, 16, 61, 52, 196, 16, 3, 249, 184, 100, 213, 248, 193, 173, 218, 155, 208, 67, 107, 37, 59, 179, 200, 68, 188, 115, 156, 30, 119, 201},
	{77, 66, 10, 234, 132, 62, 146, 160, 207, 230, 157, 137, 105, 109, 255, 104, 39, 118, 159, 156, 181, 42, 36, 154, 245, 55, 206, 137, 191, 42, 75, 116},
	{189, 3, 222, 159, 138, 178, 158, 40, 0, 9, 78, 21, 63, 172, 111, 105, 108, 250, 81, 37, 54, 201, 194, 248, 4, 220, 178, 194, 196, 228, 174, 214},
	{143, 74, 116, 160, 115, 81, 137, 93, 223, 55, 48, 87, 185, 143, 174, 109, 250, 242, 205, 33, 243, 122, 6, 62, 25, 96, 16, 120, 254, 71, 13, 83},
	{200, 14, 154, 190, 247, 255, 67, 156, 16, 198, 143, 232, 241, 48, 61, 237, 223, 197, 39, 113, 140, 59, 55, 216, 186, 104, 7, 68, 110, 60, 130, 122},
	{77, 27, 230, 79, 14, 154, 70, 108, 46, 102, 165, 52, 51, 146, 129, 146, 120, 62, 41, 248, 250, 33, 190, 178, 19, 52, 153, 181, 239, 119, 15, 96},
	{145, 66, 175, 204, 105, 27, 124, 192, 93, 38, 199, 176, 190, 12, 139, 70, 65, 130, 148, 23, 23, 48, 224, 121, 243, 132, 253, 226, 250, 80, 186, 252},
	{73, 178, 136, 228, 235, 187, 58, 40, 28, 45, 84, 111, 195, 2, 83, 213, 186, 240, 137, 147, 182, 229, 210, 149, 251, 120, 122, 91, 49, 74, 41, 142},
	{153, 48, 138, 163, 101, 196, 5, 84, 188, 137, 152, 42, 245, 5, 216, 93, 169, 82, 81, 68, 93, 93, 212, 169, 187, 55, 221, 37, 132, 253, 146, 211},
	{1, 119, 105, 32, 255, 11, 15, 56, 215, 140, 249, 92, 3, 60, 33, 173, 247, 4, 87, 133, 17, 78, 57, 42, 117, 68, 23, 150, 82, 224, 166, 18},
}

var ValidatorVotingPowersTestnet = []int64{
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
	1000000000000,
}

func NewFixedValidators(chainID string) (map[string]*types.Validator, error) {
	if chainID == MainnetBNBChainID {
		if len(MainnetValidatorAddresses) != len(MainnetValidatorPubKeyBytes) || len(MainnetValidatorAddresses) != len(MainnetValidatorVotingPowers) {
			return nil, errors.New("invalid validator set data")
		}
		validators := make(map[string]*types.Validator, len(MainnetValidatorAddresses))
		for i, addressStr := range MainnetValidatorAddresses {
			var pubKey ed25519.PubKeyEd25519
			copy(pubKey[:], MainnetValidatorPubKeyBytes[i])
			validators[addressStr] = &types.Validator{
				PubKey:      pubKey,
				VotingPower: MainnetValidatorVotingPowers[i],
			}
		}
		return validators, nil
	} else if chainID == TestnetBNBChainID {
		if len(ValidatorAddressesTestnet) != len(ValidatorPubKeyBytesTestnet) || len(ValidatorAddressesTestnet) != len(ValidatorVotingPowersTestnet) {
			return nil, errors.New("invalid validator set data")
		}
		validators := make(map[string]*types.Validator, len(ValidatorAddressesTestnet))
		for i, addressStr := range ValidatorAddressesTestnet {
			var pubKey ed25519.PubKeyEd25519
			copy(pubKey[:], ValidatorPubKeyBytesTestnet[i])
			validators[addressStr] = &types.Validator{
				PubKey:      pubKey,
				VotingPower: ValidatorVotingPowersTestnet[i],
			}
		}
		return validators, nil
	}

	return nil, errors.New("Invalid network chainID")
}

var validatorsMainnet, _ = NewFixedValidators(MainnetBNBChainID)
var validatorsTestnet, _ = NewFixedValidators(TestnetBNBChainID)
