package fetcher

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

type JSONFetcher struct {
	Version  string
	Interval time.Duration
	Updated  bool
}

func (s *JSONFetcher) Init() error {
	if s.Version == "" {
		return errors.New("version must be specified")
	}

	if s.Interval == 0 {
		s.Interval = 60 * time.Second
	}

	return nil
}

func (s *JSONFetcher) Fetch() (io.Reader, error) {
	if s.Updated {
		select {} // only one update per invocation
	}
	s.Updated = true

	time.Sleep(s.Interval)
	resp, err := http.Get("http://localhost:8081/update/?version=" + s.Version)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch update information")
	}

	var updateResponse struct {
		Outdated string `json:"outdated"`
		URL      string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&updateResponse); err != nil {
		return nil, err
	}

	if updateResponse.Outdated == "true" && updateResponse.URL != "" {
		updateResp, err := http.Get(updateResponse.URL)
		if err != nil {
			return nil, err
		}
		if updateResp.StatusCode != http.StatusOK {
			return nil, errors.New("failed to download update")
		}
		return updateResp.Body, nil
	}

	return nil, nil
}
