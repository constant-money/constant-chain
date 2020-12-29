package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/metrics"
	"os"
	"syscall"

	"github.com/google/uuid"
	"github.com/incognitochain/incognito-chain/blockchain"

	"io/ioutil"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var monitorFile *os.File
var globalParam *logKV
var blockchainObj *blockchain.BlockChain

func getCPUSample() (idle, total uint64) {
	contents, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == "cpu" {
			numFields := len(fields)
			for i := 1; i < numFields; i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					fmt.Println("Error: ", i, fields[i], err)
				}
				total += val // tally up all the numbers to get total ticks
				if i == 4 {  // idle is the 5th field in the cpu line
					idle = val
				}
			}
			return
		}
	}
	return
}

func init() {
	uid := uuid.New()
	globalParam = &logKV{param: make(map[string]interface{})}
	SetGlobalParam("UID", uid.String())
	//var err error
	//monitorFile, err = os.OpenFile("/data/monitor.json", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	//if err != nil {
	//	panic("Cannot open to monitor file")
	//}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		idle0, total0 := getCPUSample()
		var m runtime.MemStats
		for _ = range ticker.C {
			if blockchainObj == nil {
				time.Sleep(time.Second)
				continue
			}
			l := NewLog()
			idle1, total1 := getCPUSample()
			idleTicks := float64(idle1 - idle0)
			totalTicks := float64(total1 - total0)
			cpuUsage := 100 * (totalTicks - idleTicks) / totalTicks
			runtime.ReadMemStats(&m)
			bheight := blockchainObj.GetBeaconBestState().BeaconHeight
			bhash := blockchainObj.GetBeaconBestState().BestBlockHash

			//beaconValidator := []string{}
			//for _, v := range blockchain.GetBeaconBestState().GetBeaconCommittee() {
			//	beaconValidator = append(beaconValidator, v.GetMiningKeyBase58("bls"))
			//}
			//l.Add("beaconValidator", beaconValidator)
			//
			//waitingCandidateCurrentNumber := make([]string, len(blockchain.GetBeaconBestState().CandidateShardWaitingForCurrentRandom))
			//for _, v := range blockchain.GetBeaconBestState().CandidateShardWaitingForCurrentRandom {
			//	waitingCandidateCurrentNumber = append(waitingCandidateCurrentNumber, v.GetMiningKeyBase58("bls"))
			//}
			//l.Add("waitingShardCandidateCurrentNumber  ", waitingCandidateCurrentNumber)
			//
			//waitingCandidateNextNumber := make([]string, len(blockchain.GetBeaconBestState().CandidateShardWaitingForNextRandom))
			//for _, v := range blockchain.GetBeaconBestState().CandidateShardWaitingForNextRandom {
			//	waitingCandidateNextNumber = append(waitingCandidateNextNumber, v.GetMiningKeyBase58("bls"))
			//}
			//l.Add("waitingShardCandidateNextNumber", waitingCandidateNextNumber)
			//pendingShardValidator := map[byte][]string{}
			//shardValidator := map[byte][]string{}

			for i := 0; i < blockchainObj.GetBeaconBestState().ActiveShards; i++ {
				//for _, v := range blockchain.GetBestStateShard(byte(i)).ShardCommittee {
				//	shardValidator[byte(i)] = append(shardValidator[byte(i)], v.GetMiningKeyBase58("bls"))
				//}
				//l.Add("shardValidator", shardValidator)
				//
				//for _, v := range blockchain.GetBestStateShard(byte(i)).ShardPendingValidator {
				//	pendingShardValidator[byte(i)] = append(pendingShardValidator[byte(i)], v.GetMiningKeyBase58("bls"))
				//
				//}
				//l.Add("pendingShardValidator", pendingShardValidator)

				shash := blockchainObj.GetBestStateShard(byte(i)).BestBlockHash
				sheight := blockchainObj.GetBestStateShard(byte(i)).ShardHeight
				l.Add(fmt.Sprintf("Shard%v", i), fmt.Sprintf("%v:%v", sheight, shash.String()))
			}

			//disk usage
			fs := syscall.Statfs_t{}
			err := syscall.Statfs("/data", &fs)
			if err == nil {
				All := fs.Blocks * uint64(fs.Bsize)
				Free := fs.Bfree * uint64(fs.Bsize)
				Used := All - Free
				l.Add("DISK_USAGE", fmt.Sprintf("%.2f", float64(Used*100)/float64(All)))
			}
			l.Add("CPU_USAGE", fmt.Sprintf("%.2f", cpuUsage), "MEM_USAGE", m.Sys>>20, "Beacon", fmt.Sprintf("%v:%v", bheight, bhash.String()))
			idle0, total0 = getCPUSample()
			l.Write()
		}
	}()
}

type logKV struct {
	param map[string]interface{}
	sync.RWMutex
}

func SetGlobalParam(p ...interface{}) {
	globalParam.Add(p...)
}

func SetBlockChainObj(obj *blockchain.BlockChain) {
	blockchainObj = obj
}

func NewLog(p ...interface{}) *logKV {
	nl := (&logKV{param: make(map[string]interface{})}).Add(p...)
	globalParam.RLock()
	for k, v := range globalParam.param {
		nl.param[k] = v
	}
	globalParam.RUnlock()
	return nl
}

func (s *logKV) Add(p ...interface{}) *logKV {
	if len(p) == 0 || len(p)%2 != 0 {
		return s
	}
	s.Lock()
	defer s.Unlock()
	for i, v := range p {
		if i%2 == 0 {
			s.param[v.(string)] = p[i+1]
		}
	}
	return s
}

func (s *logKV) Write() {
	s.RLock()
	defer s.RUnlock()
	//fn, f, l := getMethodName(2)
	//s.param["FILE"] = fmt.Sprintf("%s:%s", f, l)
	//r, _ := regexp.Compile("(^[^\\.]*)")
	//s.param["PACKAGE"] = fmt.Sprintf("%s", r.FindStringSubmatch(fn)[1])
	s.param["Time"] = fmt.Sprintf("%s", time.Now().Format(time.RFC3339))
	b, _ := json.Marshal(s.param)
	if v, ok := s.param["Role"]; !ok || v == "" {
		return
	}
	//io.Copy(monitorFile, bytes.NewReader(b))
	//io.Copy(monitorFile, bytes.NewReader([]byte("\n")))

	go func() {
		req, err := http.NewRequest(http.MethodPost, "http://51.91.220.58:33333", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			metrics.IncLogger.Log.Debug("Create Request failed with err: ", err)
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
		defer cancel()
		req = req.WithContext(ctx)
		client := &http.Client{}
		client.Do(req)
	}()
}

func getMethodName(depthList ...int) (string, string, string) {
	var depth int
	if depthList == nil {
		depth = 1
	} else {
		depth = depthList[0]
	}
	function, file, line, _ := runtime.Caller(depth)
	r, _ := regexp.Compile("([^/]*$)")
	r1, _ := regexp.Compile("/([^/]*$)")
	return r.FindStringSubmatch(runtime.FuncForPC(function).Name())[1], r1.FindStringSubmatch(file)[1], strconv.Itoa(line)
}
