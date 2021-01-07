package pubsub

const ChanWorkLoad = 100

// TOPIC
const (
	NewShardblockTopic              = "newshardblocktopic"
	NewBeaconBlockTopic             = "newbeaconblocktopic"
	TransactionHashEnterNodeTopic   = "transactionhashenternodetopic"
	ShardRoleTopic                  = "shardroletopic"
	BeaconRoleTopic                 = "beaconroletopic"
	NodeRoleDetailTopic             = "noderoledetailtopic"
	MempoolInfoTopic                = "mempoolinfotopic"
	BeaconBeststateTopic            = "beaconbeststatetopic"
	ShardBeststateTopic             = "shardbeststatetopic"
	RequestShardBlockByHashTopic    = "requestshardblockbyhashtopic"
	RequestShardBlockByHeightTopic  = "requestshardblockbyheighttopic"
	RequestBeaconBlockByHeightTopic = "requestbeaconblockbyheighttopic"
	RequestBeaconBlockByHashTopic   = "requestbeaconblockbyhashtopic"
	TestTopic                       = "testtopic"
)

var Topics = []string{
	NewShardblockTopic,
	NewBeaconBlockTopic,
	MempoolInfoTopic,
	TestTopic,
	TransactionHashEnterNodeTopic,
	ShardRoleTopic,
	BeaconRoleTopic,
	NodeRoleDetailTopic,
	BeaconBeststateTopic,
	RequestBeaconBlockByHashTopic,
	RequestBeaconBlockByHeightTopic,
	RequestShardBlockByHeightTopic,
	RequestShardBlockByHashTopic,
	ShardBeststateTopic,
}

type NodeRole struct {
	CID  int
	Role string
}
