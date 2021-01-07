package transaction

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
)

type ValidationEnv struct {
	builderSView common.Hash
	isPrivacy    bool
	isConfimed   bool
	txType       string
	shardID      int
	shardHeight  uint64
	beaconHeight uint64
	confimedTime int64
	version      int
}

func NewValEnv(
	isPrivacyTx bool,
	txType string,
	sHeight uint64,
	bHeight uint64,
	sID int,
	confimedTime uint64,
	ver int,
) *ValidationEnv {
	return &ValidationEnv{}
}

func DefaultValEnv() *ValidationEnv {
	return &ValidationEnv{
		builderSView: common.EmptyRoot,
		isConfimed:   false,
		shardHeight:  0,
		beaconHeight: 0,
		confimedTime: 0,
	}
}

func WithBuilderSView(vE metadata.ValidationEnviroment, view common.Hash) *ValidationEnv {
	return &ValidationEnv{
		builderSView: view,
		isConfimed:   vE.IsConfimed(),
		shardHeight:  vE.ShardHeight(),
		beaconHeight: vE.BeaconHeight(),
		confimedTime: vE.ConfimedTime(),
		shardID:      vE.ShardID(),
		version:      vE.Version(),
		isPrivacy:    true,
		txType:       vE.TxType(),
	}
}

func WithPrivacy(vE metadata.ValidationEnviroment) *ValidationEnv {
	return &ValidationEnv{
		builderSView: vE.BuilderSView(),
		isConfimed:   vE.IsConfimed(),
		shardHeight:  vE.ShardHeight(),
		beaconHeight: vE.BeaconHeight(),
		confimedTime: vE.ConfimedTime(),
		shardID:      vE.ShardID(),
		version:      vE.Version(),
		isPrivacy:    true,
		txType:       vE.TxType(),
	}
}

func WithNoPrivacy(vE metadata.ValidationEnviroment) *ValidationEnv {
	return &ValidationEnv{
		builderSView: vE.BuilderSView(),
		isConfimed:   vE.IsConfimed(),
		shardHeight:  vE.ShardHeight(),
		beaconHeight: vE.BeaconHeight(),
		confimedTime: vE.ConfimedTime(),
		shardID:      vE.ShardID(),
		version:      vE.Version(),
		isPrivacy:    false,
		txType:       vE.TxType(),
	}
}

func WithShardID(vE metadata.ValidationEnviroment, sID int) *ValidationEnv {
	return &ValidationEnv{
		builderSView: vE.BuilderSView(),
		isConfimed:   vE.IsConfimed(),
		shardHeight:  vE.ShardHeight(),
		beaconHeight: vE.BeaconHeight(),
		confimedTime: vE.ConfimedTime(),
		shardID:      sID,
		version:      vE.Version(),
		isPrivacy:    vE.IsPrivacy(),
		txType:       vE.TxType(),
	}
}

func WithShardHeight(vE metadata.ValidationEnviroment, sHeight uint64) *ValidationEnv {
	return &ValidationEnv{
		builderSView: vE.BuilderSView(),
		isConfimed:   vE.IsConfimed(),
		shardHeight:  sHeight,
		beaconHeight: vE.BeaconHeight(),
		confimedTime: vE.ConfimedTime(),
		shardID:      vE.ShardID(),
		version:      vE.Version(),
		isPrivacy:    vE.IsPrivacy(),
		txType:       vE.TxType(),
	}
}

func WithBeaconHeight(vE metadata.ValidationEnviroment, bcHeight uint64) *ValidationEnv {
	return &ValidationEnv{
		builderSView: vE.BuilderSView(),
		isConfimed:   vE.IsConfimed(),
		shardHeight:  vE.ShardHeight(),
		beaconHeight: bcHeight,
		confimedTime: vE.ConfimedTime(),
		shardID:      vE.ShardID(),
		version:      vE.Version(),
		isPrivacy:    vE.IsPrivacy(),
		txType:       vE.TxType(),
	}
}

func WithConfirmedTime(vE metadata.ValidationEnviroment, confimedTime int64) *ValidationEnv {
	return &ValidationEnv{
		builderSView: vE.BuilderSView(),
		isConfimed:   true,
		shardHeight:  vE.ShardHeight(),
		beaconHeight: vE.BeaconHeight(),
		confimedTime: confimedTime,
		shardID:      vE.ShardID(),
		version:      vE.Version(),
		isPrivacy:    vE.IsPrivacy(),
		txType:       vE.TxType(),
	}
}

func WithType(vE metadata.ValidationEnviroment, t string) *ValidationEnv {
	return &ValidationEnv{
		builderSView: vE.BuilderSView(),
		isConfimed:   vE.IsConfimed(),
		shardHeight:  vE.ShardHeight(),
		beaconHeight: vE.BeaconHeight(),
		confimedTime: vE.ConfimedTime(),
		shardID:      vE.ShardID(),
		version:      vE.Version(),
		isPrivacy:    vE.IsPrivacy(),
		txType:       t,
	}
}

func (vE *ValidationEnv) IsPrivacy() bool {
	return vE.isPrivacy
}
func (vE *ValidationEnv) IsConfimed() bool {
	return vE.isConfimed
}
func (vE *ValidationEnv) TxType() string {
	return vE.txType
}
func (vE *ValidationEnv) ShardID() int {
	return vE.shardID
}
func (vE *ValidationEnv) ShardHeight() uint64 {
	return vE.shardHeight
}
func (vE *ValidationEnv) BeaconHeight() uint64 {
	return vE.beaconHeight
}
func (vE *ValidationEnv) ConfimedTime() int64 {
	return vE.confimedTime
}
func (vE *ValidationEnv) Version() int {
	return vE.version
}
func (vE *ValidationEnv) BuilderSView() common.Hash {
	return vE.builderSView
}
