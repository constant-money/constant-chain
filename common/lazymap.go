package common

import (
	"sort"
)

//MapStringString ...
type MapStringString struct {
	data    map[string]string
	hash    *Hash
	updated bool
}

func (s *MapStringString) Data() map[string]string {
	return s.data
}

func (s *MapStringString) SetUpdated(updated bool) {
	s.updated = updated
}

func (s *MapStringString) SetHash(hash *Hash) {
	s.hash = hash
}

func (s *MapStringString) SetData(data map[string]string) {
	s.data = data
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
	s.data = make(map[string]string)
	for k, v := range prev {
		s.data[k] = v
	}
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
func (s *MapStringString) GenerateHash() (Hash, error) {
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
	h, err := GenerateHashFromStringArray(res)
	if err != nil {
		return Hash{}, err
	}
	s.hash = &h
	return h, nil
}

type MapStringBool struct {
	data    map[string]bool
	hash    *Hash
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
	s.data = make(map[string]bool)
	for k, v := range prev {
		s.data[k] = v
	}
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

func (s *MapStringBool) GenerateHash() (Hash, error) {
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
	h, err := GenerateHashFromStringArray(res)
	if err != nil {
		return Hash{}, err
	}
	s.hash = &h
	return h, nil
}
