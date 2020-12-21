package blockchain

import (
	"sort"

	"github.com/incognitochain/incognito-chain/common"
)

//MapStringString ...
type MapStringString struct {
	data    map[string]string
	hash    *common.Hash
	updated bool
}

//NewMapStringString ...
func NewMapStringString() *MapStringString {
	return &MapStringString{
		data:    make(map[string]string),
		hash:    nil,
		updated: false,
	}
}

//LazyCopy ...
func (s *MapStringString) LazyCopy() *MapStringString {
	newCopy := *s
	s.updated = false
	return &newCopy
}

//copy ...
func (s *MapStringString) copy() {
	prev := s.data
	newData := make(map[string]string)
	for k, v := range prev {
		newData[k] = v
	}
	s.data = newData
	s.updated = false
}

//Remove ...
func (s *MapStringString) Remove(k string) {
	if !s.updated {
		s.copy()
	}
	delete(s.data, k)
	s.updated = true
	s.hash = nil
}

//Set ...
func (s *MapStringString) Set(k string, v string) {
	if !s.updated {
		s.copy()
	}
	s.data[k] = v
	s.updated = true
	s.hash = nil
}

//GetMap ...
func (s *MapStringString) GetMap() map[string]string {
	return s.data
}

//Get ...
func (s *MapStringString) Get(k string) (string, bool) {
	r, ok := s.data[k]
	return r, ok
}

//GenerateHash ...
func (s *MapStringString) GenerateHash() (common.Hash, error) {
	if s.hash != nil {
		return *s.hash, nil
	}
	var keys []string
	var res []string
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		res = append(res, key)
		res = append(res, s.data[key])
	}
	h, err := generateHashFromStringArray(res)
	if err != nil {
		return common.Hash{}, err
	}
	s.hash = &h
	return h, nil
}

type MapStringBool struct {
	data    map[string]bool
	hash    *common.Hash
	updated bool
}

func NewMapStringBool() *MapStringBool {
	return &MapStringBool{
		data:    make(map[string]bool),
		hash:    nil,
		updated: false,
	}
}

func (s *MapStringBool) LazyCopy() *MapStringBool {
	newCopy := *s
	s.updated = false
	return &newCopy
}

func (s *MapStringBool) copy() {
	prev := s.data
	newData := make(map[string]bool)
	for k, v := range prev {
		newData[k] = v
	}
	s.data = newData
	s.updated = false
}

func (s *MapStringBool) Remove(k string) {
	if !s.updated {
		s.copy()
	}
	delete(s.data, k)
	s.updated = true
	s.hash = nil
}

func (s *MapStringBool) Set(k string, v bool) {
	if !s.updated {
		s.copy()
	}
	s.data[k] = v
	s.updated = true
	s.hash = nil
}

func (s *MapStringBool) GetMap() map[string]bool {
	return s.data
}

func (s *MapStringBool) Get(k string) (bool, bool) {
	r, ok := s.data[k]
	return r, ok
}

func (s *MapStringBool) GenerateHash() (common.Hash, error) {
	if s.hash != nil {
		return *s.hash, nil
	}
	var keys []string
	var res []string
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		res = append(res, key)
		if s.data[key] {
			res = append(res, "true")
		} else {
			res = append(res, "false")
		}
	}
	h, err := generateHashFromStringArray(res)
	if err != nil {
		return common.Hash{}, err
	}
	s.hash = &h
	return h, nil
}
