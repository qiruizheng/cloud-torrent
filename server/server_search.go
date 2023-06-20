package server

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/jpillora/backoff"
)

const searchConfigURL = "https://gist.githubusercontent.com/jpillora/4d945b46b3025843b066adf3d685be6b/raw/scraper-config.json"

func (s *Server) fetchSearchConfigLoop() {
	b := backoff.Backoff{Max: 30 * time.Minute}
	for {
		if err := s.fetchSearchConfig(); err != nil {
			//ignore error
			time.Sleep(b.Duration())
		} else {
			//no errror - check again in half hour
			time.Sleep(30 * time.Minute)
			b.Reset()
		}
	}
}

var fetches = 0
var currentConfig, _ = normalize(defaultSearchConfig)

func (s *Server) fetchSearchConfig() error {
	resp, err := http.Get(searchConfigURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	newConfig, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	newConfig, err = normalize(newConfig)
	if err != nil {
		return err
	}
	fetches++
	if bytes.Equal(currentConfig, newConfig) {
		return nil //skip
	}
	if err := s.scraper.LoadConfig(newConfig); err != nil {
		return err
	}
	s.state.SearchProviders = s.scraper.Config
	s.state.Push()
	currentConfig = newConfig
	log.Printf("Loaded new search providers")
	return nil
}

func normalize(input []byte) ([]byte, error) {
	output := bytes.Buffer{}
	if err := json.Indent(&output, input, "", "  "); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

//see github.com/jpillora/scraper for config specification
//cloud-torrent uses "<id>-item" handlers
var defaultSearchConfig = []byte(`{
"jackett": {
  "name": "Jackett",
  "url": "http://192.168.2.130:9117/api/v2.0/indexers/all/results/torznab/api?apikey=vp5cwbgbcmsl1xg4h4xg3me72ouhaibu&t=search&q={{query}}&page={{page:1}}",
  "list": ".results > div.result-item",
  "result": {
    "name": ".release-title",
    "url": [".release-title a", "@href"],
    "magnet": ["a[href^='magnet:?']", "@href"],
    "seeds": ".release-peers .label-seeders",
    "peers": ".release-peers .label-leechers"
  }
}
}`)
