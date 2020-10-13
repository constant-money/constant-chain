package wallet

// import (
// 	"bytes"
// 	"encoding/hex"
// 	"encoding/json"
// 	"fmt"
// 	"github.com/incognitochain/incognito-chain/common"
// 	"github.com/incognitochain/incognito-chain/common/base58"
// 	"io/ioutil"
// )

// type AccountWallet struct {
// 	Name       string
// 	Key        KeyWallet
// 	Child      []AccountWallet
// 	IsImported bool
// }

// type Wallet struct {
// 	Seed          []byte
// 	Entropy       []byte
// 	PassPhrase    string
// 	Mnemonic      string
// 	MasterAccount AccountWallet
// 	Name          string
// 	config        *WalletConfig
// }

// type WalletConfig struct {
// 	DataDir        string
// 	DataFile       string
// 	DataPath       string
// 	IncrementalFee uint64
// 	ShardID        *byte //default is nil -> create account for any shard
// }

// // GetConfig returns configuration of wallet
// func (wallet Wallet) GetConfig() *WalletConfig {
// 	return wallet.config
// }

// // SetConfig sets config to configuration of wallet
// func (wallet *Wallet) SetConfig(config *WalletConfig) {
// 	wallet.config = config
// }

// // Init initializes new wallet with pass phrase, number of accounts and wallet name
// // It returns error if there are any errors when initializing wallet. Otherwise, it returns nil
// // passPhrase can be empty string, it is used to generate seed and master key
// // If numOfAccount equals zero, wallet is initialized with one account
// // If name is empty string, it returns error
// func (wallet *Wallet) Init(passPhrase string, numOfAccount uint32, name string) error {
// 	if name == "" {
// 		return NewWalletError(EmptyWalletNameErr, nil)
// 	}

// 	mnemonicGen := MnemonicGenerator{}
// 	wallet.Name = name
// 	wallet.Entropy, _ = mnemonicGen.newEntropy(128)
// 	wallet.Mnemonic, _ = mnemonicGen.newMnemonic(wallet.Entropy)
// 	wallet.Seed = mnemonicGen.NewSeed(wallet.Mnemonic, passPhrase)
// 	wallet.PassPhrase = passPhrase

// 	masterKey, err := NewMasterKey(wallet.Seed)
// 	if err != nil {
// 		return err
// 	}
// 	wallet.MasterAccount = AccountWallet{
// 		Key:   *masterKey,
// 		Child: make([]AccountWallet, 0),
// 		Name:  "master",
// 	}

// 	if numOfAccount == 0 {
// 		numOfAccount = 1
// 	}

// 	for i := uint32(0); i < numOfAccount; i++ {
// 		childKey, _ := wallet.MasterAccount.Key.NewChildKey(i)
// 		account := AccountWallet{
// 			Key:   *childKey,
// 			Child: make([]AccountWallet, 0),
// 			Name:  fmt.Sprintf("AccountWallet %d", i),
// 		}
// 		wallet.MasterAccount.Child = append(wallet.MasterAccount.Child, account)
// 	}

// 	return nil
// }

// // CreateNewAccount create new account with accountName
// // it returns that new account and returns errors if accountName is existed
// // If shardID is nil, new account will belong to any shards
// // Otherwise, new account will belong to specific shard
// func (wallet *Wallet) CreateNewAccount(accountName string, shardID *byte) (*AccountWallet, error) {
// 	if accountName != "" {
// 		for _, acc := range wallet.MasterAccount.Child {
// 			if acc.Name == accountName {
// 				return nil, NewWalletError(ExistedAccountNameErr, nil)
// 			}
// 		}
// 	}

// 	if shardID != nil {
// 		// only create account for specific Shard
// 		newIndex := uint64(0)
// 		// loop to get newest index of childs
// 		for j := len(wallet.MasterAccount.Child) - 1; j >= 0; j-- {
// 			temp := wallet.MasterAccount.Child[j]
// 			if !temp.IsImported {
// 				childNumber := temp.Key.ChildNumber
// 				childNumberInt32, err := common.BytesToInt32(childNumber)
// 				if err != nil {
// 					return nil, NewWalletError(UnexpectedErr, err)
// 				}
// 				newIndex = uint64(childNumberInt32 + 1)
// 				break
// 			}
// 		}

// 		// loop to get create a new child which can be equal shardID param
// 		var childKey *KeyWallet
// 		for true {
// 			childKey, _ = wallet.MasterAccount.Key.NewChildKey(uint32(newIndex))
// 			lastByte := childKey.KeySet.PaymentAddress.Pk[len(childKey.KeySet.PaymentAddress.Pk)-1]
// 			if common.GetShardIDFromLastByte(lastByte) == *shardID {
// 				break
// 			}
// 			newIndex += 1
// 		}
// 		// use chosen childKey tp create an child account for wallet
// 		if accountName == "" {
// 			accountName = fmt.Sprintf("AccountWallet %d", len(wallet.MasterAccount.Child))
// 		}

// 		account := AccountWallet{
// 			Key:   *childKey,
// 			Child: make([]AccountWallet, 0),
// 			Name:  accountName,
// 		}
// 		wallet.MasterAccount.Child = append(wallet.MasterAccount.Child, account)
// 		err := wallet.Save(wallet.PassPhrase)
// 		if err != nil {
// 			Logger.log.Error(err)
// 		}
// 		return &account, nil

// 	} else {
// 		newIndex := uint32(len(wallet.MasterAccount.Child))
// 		childKey, _ := wallet.MasterAccount.Key.NewChildKey(newIndex)
// 		if accountName == "" {
// 			accountName = fmt.Sprintf("AccountWallet %d", len(wallet.MasterAccount.Child))
// 		}
// 		account := AccountWallet{
// 			Key:   *childKey,
// 			Child: make([]AccountWallet, 0),
// 			Name:  accountName,
// 		}
// 		wallet.MasterAccount.Child = append(wallet.MasterAccount.Child, account)
// 		err := wallet.Save(wallet.PassPhrase)
// 		if err != nil {
// 			Logger.log.Error(err)
// 		}
// 		return &account, nil
// 	}
// }

// // ExportAccount returns a private key string of account at childIndex in wallet
// // It is base58 check serialized
// func (wallet *Wallet) ExportAccount(childIndex uint32) string {
// 	if int(childIndex) >= len(wallet.MasterAccount.Child) {
// 		return ""
// 	}
// 	return wallet.MasterAccount.Child[childIndex].Key.Base58CheckSerialize(PriKeyType)
// }

// func (wallet *Wallet) RemoveAccount(privateKeyStr string, passPhrase string) error {
// 	if passPhrase != wallet.PassPhrase {
// 		return NewWalletError(WrongPassphraseErr, nil)
// 	}
// 	for i, account := range wallet.MasterAccount.Child {
// 		if account.Key.Base58CheckSerialize(PriKeyType) == privateKeyStr {
// 			wallet.MasterAccount.Child = append(wallet.MasterAccount.Child[:i], wallet.MasterAccount.Child[i+1:]...)
// 			err := wallet.Save(passPhrase)
// 			if err != nil {
// 				Logger.log.Error(err)
// 			}
// 			return nil
// 		}
// 	}
// 	return NewWalletError(NotFoundAccountErr, nil)
// }

// // ImportAccount adds account into wallet with privateKeyStr, accountName, and passPhrase which is used to init wallet
// // It returns AccountWallet which is imported and errors (if any)
// func (wallet *Wallet) ImportAccount(privateKeyStr string, accountName string, passPhrase string) (*AccountWallet, error) {
// 	if passPhrase != wallet.PassPhrase {
// 		return nil, NewWalletError(WrongPassphraseErr, nil)
// 	}

// 	for _, account := range wallet.MasterAccount.Child {
// 		if account.Key.Base58CheckSerialize(PriKeyType) == privateKeyStr {
// 			return nil, NewWalletError(ExistedAccountErr, nil)
// 		}
// 		if account.Name == accountName {
// 			return nil, NewWalletError(ExistedAccountNameErr, nil)
// 		}
// 	}

// 	keyWallet, err := Base58CheckDeserialize(privateKeyStr)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	if err != nil {
// 		return nil, err
// 	}

// 	Logger.log.Debugf("Pub-key : %s", keyWallet.Base58CheckSerialize(PaymentAddressType))
// 	Logger.log.Debugf("Readonly-key : %s", keyWallet.Base58CheckSerialize(ReadonlyKeyType))

// 	account := AccountWallet{
// 		Key:        *keyWallet,
// 		Child:      make([]AccountWallet, 0),
// 		IsImported: true,
// 		Name:       accountName,
// 	}
// 	wallet.MasterAccount.Child = append(wallet.MasterAccount.Child, account)
// 	err = wallet.Save(wallet.PassPhrase)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &account, nil
// }

// // Save saves encrypted wallet (using AES encryption scheme) in config data file of wallet
// // It returns error if any
// func (wallet *Wallet) Save(password string) error {
// 	if password == "" {
// 		password = wallet.PassPhrase
// 	}

// 	if password != wallet.PassPhrase {
// 		return NewWalletError(WrongPassphraseErr, nil)
// 	}

// 	// parse to byte[]
// 	data, err := json.Marshal(*wallet)
// 	if err != nil {
// 		Logger.log.Error(err)
// 		return NewWalletError(JsonMarshalErr, err)
// 	}

// 	// encrypt data
// 	cipherText, err := encryptByPassPhrase(password, data)
// 	if err != nil {
// 		Logger.log.Error(err)
// 		return NewWalletError(UnexpectedErr, err)
// 	}
// 	// and
// 	// save file
// 	cipherTexInBytes := []byte(cipherText)
// 	err = ioutil.WriteFile(wallet.config.DataPath, cipherTexInBytes, 0644)
// 	if err != nil {
// 		return NewWalletError(WriteFileErr, err)
// 	}
// 	return nil
// }

// // LoadWallet loads encrypted wallet from file and then decrypts it to wallet struct
// // It returns error if any
// func (wallet *Wallet) LoadWallet(password string) error {
// 	// read file and decrypt
// 	bytesData, err := ioutil.ReadFile(wallet.config.DataPath)
// 	if err != nil {
// 		return NewWalletError(ReadFileErr, err)
// 	}
// 	bufBytes, err := decryptByPassPhrase(password, string(bytesData))
// 	if err != nil {
// 		return NewWalletError(AESDecryptErr, err)
// 	}

// 	// read to struct
// 	err = json.Unmarshal(bufBytes, &wallet)
// 	if err != nil {
// 		return NewWalletError(JsonUnmarshalErr, err)
// 	}
// 	return nil
// }

// // DumpPrivkey receives base58 check serialized payment address (paymentAddrSerialized)
// // and returns KeySerializedData object contains PrivateKey
// // which is corresponding to paymentAddrSerialized in all wallet accounts
// // If there is not any wallet account corresponding to paymentAddrSerialized, it returns empty KeySerializedData object
// func (wallet *Wallet) DumpPrivateKey(paymentAddrSerialized string) KeySerializedData {
// 	for _, account := range wallet.MasterAccount.Child {
// 		address := account.Key.Base58CheckSerialize(PaymentAddressType)
// 		if address == paymentAddrSerialized {
// 			key := KeySerializedData{
// 				PrivateKey: account.Key.Base58CheckSerialize(PriKeyType),
// 			}
// 			return key
// 		}
// 	}
// 	return KeySerializedData{}
// }

// // GetAddressByAccName receives accountName and shardID
// // and returns corresponding account's KeySerializedData object contains base58 check serialized PaymentAddress,
// // hex encoding Pubkey and base58 check serialized ReadonlyKey
// // If there is not any account corresponding to accountName, we will create new account
// func (wallet *Wallet) GetAddressByAccName(accountName string, shardID *byte) KeySerializedData {
// 	for _, account := range wallet.MasterAccount.Child {
// 		if account.Name == accountName {
// 			key := KeySerializedData{
// 				PaymentAddress: account.Key.Base58CheckSerialize(PaymentAddressType),
// 				Pubkey:         hex.EncodeToString(account.Key.KeySet.PaymentAddress.Pk),
// 				ReadonlyKey:    account.Key.Base58CheckSerialize(ReadonlyKeyType),
// 				PrivateKey:     account.Key.Base58CheckSerialize(PriKeyType),
// 				ValidatorKey:   base58.Base58Check{}.Encode(common.HashB(common.HashB(account.Key.KeySet.PrivateKey)), common.ZeroByte),
// 			}
// 			return key
// 		}
// 	}
// 	newAccount, _ := wallet.CreateNewAccount(accountName, shardID)
// 	key := KeySerializedData{
// 		PaymentAddress: newAccount.Key.Base58CheckSerialize(PaymentAddressType),
// 		Pubkey:         hex.EncodeToString(newAccount.Key.KeySet.PaymentAddress.Pk),
// 		ReadonlyKey:    newAccount.Key.Base58CheckSerialize(ReadonlyKeyType),
// 		PrivateKey:     newAccount.Key.Base58CheckSerialize(PriKeyType),
// 		ValidatorKey:   base58.Base58Check{}.Encode(common.HashB(common.HashB(newAccount.Key.KeySet.PrivateKey)), common.ZeroByte),
// 	}
// 	return key
// }

// // GetAddressesByAccName receives accountName
// // and returns list of KeySerializedData of accounts which has accountName
// func (wallet *Wallet) GetAddressesByAccName(accountName string) []KeySerializedData {
// 	result := make([]KeySerializedData, 0)
// 	for _, account := range wallet.MasterAccount.Child {
// 		if account.Name == accountName {
// 			item := KeySerializedData{
// 				PaymentAddress: account.Key.Base58CheckSerialize(PaymentAddressType),
// 				Pubkey:         hex.EncodeToString(account.Key.KeySet.PaymentAddress.Pk),
// 				ReadonlyKey:    account.Key.Base58CheckSerialize(ReadonlyKeyType),
// 				ValidatorKey:   base58.Base58Check{}.Encode(common.HashB(common.HashB(account.Key.KeySet.PrivateKey)), common.ZeroByte),
// 			}
// 			result = append(result, item)
// 		}
// 	}
// 	return result
// }

// // ListAccounts returns a map with key is account name and value is account wallet
// func (wallet *Wallet) ListAccounts() map[string]AccountWallet {
// 	result := make(map[string]AccountWallet)
// 	for _, account := range wallet.MasterAccount.Child {
// 		result[account.Name] = account
// 	}
// 	return result
// }

// // ContainPubKey checks whether the wallet contains any account with pubKey or not
// func (wallet *Wallet) ContainPublicKey(pubKey []byte) bool {
// 	for _, account := range wallet.MasterAccount.Child {
// 		if bytes.Equal(account.Key.KeySet.PaymentAddress.Pk[:], pubKey) {
// 			return true
// 		}
// 	}
// 	return false
// }
