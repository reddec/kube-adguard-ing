package static

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Record struct {
	Domain  string   `yaml:"domain"`
	Address []string `yaml:"address"`
}

type Config struct {
	Path string        `long:"path" env:"PATH" description:"Path to yaml file"`
	TTL  time.Duration `long:"ttl" env:"TTL" description:"YAML file cache duration" default:"5s"`
}

type Static struct {
	file    string
	cache   []Record
	ttl     time.Duration
	updated time.Time
	lock    sync.RWMutex
}

func New(cfg Config) *Static {
	return &Static{
		file: cfg.Path,
		ttl:  cfg.TTL,
	}
}

func (s *Static) Load() ([]Record, error) {
	if s.file == "" {
		return nil, nil
	}
	s.lock.RLock()
	state, expired := s.cache, s.expired()
	s.lock.RUnlock()
	if !expired {
		return state, nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if !s.expired() {
		return s.cache, nil
	}

	newState, err := s.parseFile()
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}
	s.cache = newState
	s.updated = time.Now()
	return newState, nil
}

func (s *Static) expired() bool {
	return time.Since(s.updated) > s.ttl
}

func (s *Static) parseFile() ([]Record, error) {
	f, err := os.Open(s.file)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", s.file, err)
	}
	defer f.Close()
	var out []Record
	err = yaml.NewDecoder(f).Decode(&out)
	return out, err
}
