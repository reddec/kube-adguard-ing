package utils

import (
	"encoding/json"
	"sort"
)

type Set map[string]struct{}

func NewSet(values ...string) Set {
	var ans = make(Set, len(values))
	for _, v := range values {
		ans[v] = struct{}{}
	}
	return ans
}

func (s Set) Without(other Set) Set {
	var a = make(Set)
	for k := range s {
		if !other.Has(k) {
			a[k] = struct{}{}
		}
	}
	return a
}

func (s Set) Add(k ...string) {
	for _, v := range k {
		s[v] = struct{}{}
	}
}

func (s Set) Include(another Set) {
	for k := range another {
		s[k] = struct{}{}
	}
}

func (s Set) Has(k string) bool {
	_, ok := s[k]
	return ok
}

func (s Set) Slice() []string {
	var ans []string
	for k := range s {
		ans = append(ans, k)
	}
	sort.Strings(ans)
	return ans
}

func (s Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Slice())
}
