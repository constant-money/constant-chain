package simplemetric

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"
)

type TimeCounter interface {
	AddKey(string)
	AddSubKey(string, string)
	AddSubKeyWithValue(string, string, time.Duration)
	StartSubKey(string, string)
	PauseSubKey(string, string)
	EndSubKey(string, string)
	EndKey(string, string)
	Report() string
}

type FeatureTimeCounter struct {
	summary time.Duration
	start   time.Time
	end     time.Time
}

func NewFC() *FeatureTimeCounter {
	return &FeatureTimeCounter{
		summary: 0,
	}
}

func (fc *FeatureTimeCounter) Start() {
	fc.start = time.Now()
}

func (fc *FeatureTimeCounter) Pause() {
	fc.summary += time.Since(fc.start)
}

func (fc *FeatureTimeCounter) End() {
	fc.summary += time.Since(fc.start)
	fc.end = time.Now()
}

type SubKeyTimeCounter struct {
	FeatureMap map[string]*FeatureTimeCounter
	sync.RWMutex
}

func NewST() *SubKeyTimeCounter {
	return &SubKeyTimeCounter{
		FeatureMap: map[string]*FeatureTimeCounter{},
	}
}

var ConsensusTimer ConsensusCounter

func init() {
	ConsensusTimer = ConsensusCounter{
		KeyMap: map[string]*SubKeyTimeCounter{},
	}
}

type ConsensusCounter struct {
	KeyMap map[string]*SubKeyTimeCounter
	sync.RWMutex
}

func (cc *ConsensusCounter) AddKey(k string) {
	cc.Lock()
	defer cc.Unlock()
	if _, ok := cc.KeyMap[k]; ok {
		return
	}
	cc.KeyMap[k] = NewST()
}

func (cc *ConsensusCounter) AddSubKey(k string, sk string) {
	cc.RLock()
	if km, ok := cc.KeyMap[k]; ok {
		cc.RUnlock()
		km.Lock()
		if _, iok := km.FeatureMap[sk]; iok {
			km.Unlock()
			return
		}
		km.FeatureMap[sk] = NewFC()
		km.Unlock()
	} else {
		cc.RUnlock()
		cc.AddKey(k)
		cc.AddSubKey(k, sk)
	}
}

func (cc *ConsensusCounter) AddSubKeyWithValue(k string, sk string, vl time.Duration) {
	cc.RLock()
	if km, ok := cc.KeyMap[k]; ok {
		cc.RUnlock()
		km.Lock()
		if _, iok := km.FeatureMap[sk]; iok {
			km.Unlock()
			return
		}
		km.FeatureMap[sk] = &FeatureTimeCounter{
			summary: vl,
		}
		km.Unlock()
	} else {
		cc.RUnlock()
		cc.AddKey(k)
		cc.AddSubKeyWithValue(k, sk, vl)
	}
}

func (cc *ConsensusCounter) StartSubKey(k string, sk string) {
	cc.RLock()
	if km, ok := cc.KeyMap[k]; ok {
		cc.RUnlock()
		km.RLock()
		if skm, iok := km.FeatureMap[sk]; iok {
			skm.Start()
		}
		km.RUnlock()
	} else {
		cc.RUnlock()
	}
}

func (cc *ConsensusCounter) PauseSubKey(k string, sk string) {
	cc.RLock()
	if km, ok := cc.KeyMap[k]; ok {
		cc.RUnlock()
		km.RLock()
		if skm, iok := km.FeatureMap[sk]; iok {
			skm.Start()
		}
		km.RUnlock()
	} else {
		cc.RUnlock()
	}
}

func (cc *ConsensusCounter) EndSubKey(k string, sk string) {
	cc.RLock()
	if km, ok := cc.KeyMap[k]; ok {
		cc.RUnlock()
		km.RLock()
		if skm, iok := km.FeatureMap[sk]; iok {
			skm.End()
		}
		km.RUnlock()
	} else {
		cc.RUnlock()
	}
}

func (cc *ConsensusCounter) EndKey(k string) {
	cc.RLock()
	if km, ok := cc.KeyMap[k]; ok {
		cc.RUnlock()
		km.Lock()
		for _, skm := range km.FeatureMap {
			skm.End()
		}
		km.Unlock()
	} else {
		cc.RUnlock()
	}
}

func (cc *ConsensusCounter) Report(name string) string {
	cc.RLock()
	defer cc.RUnlock()
	res := map[string]map[string]int64{}
	resjson := []map[string]interface{}{}
	listSubKey := map[string]struct{}{}
	subKeys := []string{}
	records := [][]string{}

	for _, km := range cc.KeyMap {
		for sk := range km.FeatureMap {
			listSubKey[sk] = struct{}{}
		}
	}
	record := []string{}
	record = append(record, "Key")
	for sk := range listSubKey {
		record = append(record, sk)
		subKeys = append(subKeys, sk)
	}
	sort.Strings(subKeys)
	records = append(records, record)

	for k, km := range cc.KeyMap {
		record := []string{}
		record = append(record, k)
		for _, sk := range subKeys {
			if skm, ok := km.FeatureMap[sk]; ok {
				record = append(record, fmt.Sprintf("%v", skm.summary))
			} else {
				record = append(record, "")
			}
		}
		res[k] = map[string]int64{}
		tmp := map[string]interface{}{}
		tmp["Key"] = k
		for sk, skm := range km.FeatureMap {
			res[k][sk] = skm.summary.Milliseconds()
			tmp[sk] = skm.summary
		}
		resjson = append(resjson, tmp)
		records = append(records, record)
	}
	resStr, err := json.Marshal(resjson)
	if err != nil {
		return err.Error()
	}
	currentTime := time.Now()
	// ioutil.WriteFile(name+currentTime.Format("2006-01-02")+".json", resStr, os.ModePerm)
	f, err := os.Create(name + currentTime.Format("2006-01-02") + ".csv")
	if err != nil {
		log.Fatalln("failed to open file", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	_ = w.WriteAll(records)
	return string(resStr)
}
