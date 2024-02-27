package adguard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

var ErrBadStatus = errors.New("response code is not 2xx")

type Record struct {
	Domain  string `json:"domain"`
	Address string `json:"answer"`
}

type Config struct {
	URL      string        `long:"url" env:"URL" description:"AdGuard URL" required:"true"`
	User     string        `long:"user" env:"USER" description:"Username" required:"true"`
	Password string        `long:"password" env:"PASSWORD" description:"Password" required:"true"`
	Timeout  time.Duration `long:"timeout" env:"TIMEOUT" description:"Single operation timeout" default:"5s"`
}

func New(cfg Config) (*AdGuard, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	return &AdGuard{
		config: cfg,
		base:   u,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

type AdGuard struct {
	config Config
	base   *url.URL
	client *http.Client
}

func (adg *AdGuard) List(ctx context.Context) ([]Record, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adg.makeURL("/control/rewrite/list"), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth(adg.config.User, adg.config.Password)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %w", res.StatusCode, ErrBadStatus)
	}

	var ans []Record
	err = json.NewDecoder(res.Body).Decode(&ans)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return ans, nil
}

func (adg *AdGuard) Delete(ctx context.Context, record Record) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, adg.makeURL("/control/rewrite/delete"), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(adg.config.User, adg.config.Password)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", res.StatusCode)
	}

	return nil
}

func (adg *AdGuard) Add(ctx context.Context, record Record) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, adg.makeURL("/control/rewrite/add"), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(adg.config.User, adg.config.Password)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", res.StatusCode)
	}

	return nil
}

func (adg *AdGuard) makeURL(subPath string) string {
	cp := *adg.base
	cp.Path = path.Join(cp.Path, subPath)
	return cp.String()
}
