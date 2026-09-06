package utils

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ecbRatesURL      = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
	currencyCacheTTL = 12 * time.Hour
)

var (
	currencyPattern = regexp.MustCompile(`^([-+]?\d+(?:[.,]\d+)?)\s*([a-z]{3})\s+(?:to|in)\s+([a-z]{3})$`)
	currencyMu      sync.RWMutex
	currencyRates   map[string]float64
	currencyUpdated time.Time
)

type currencyCache struct {
	Updated time.Time          `json:"updated"`
	Rates   map[string]float64 `json:"rates"`
}

type ecbEnvelope struct {
	Cube struct {
		Day struct {
			Rates []struct {
				Currency string  `xml:"currency,attr"`
				Rate     float64 `xml:"rate,attr"`
			} `xml:"Cube"`
		} `xml:"Cube"`
	} `xml:"Cube"`
}

func LoadCurrencyRates() {
	path, err := currencyCachePath()
	if err != nil {
		return
	}
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	var cached currencyCache
	if json.NewDecoder(file).Decode(&cached) != nil || len(cached.Rates) == 0 {
		return
	}
	currencyMu.Lock()
	currencyRates = cached.Rates
	currencyUpdated = cached.Updated
	currencyMu.Unlock()
}

func RefreshCurrencyRates() bool {
	currencyMu.RLock()
	fresh := len(currencyRates) > 0 && time.Since(currencyUpdated) < currencyCacheTTL
	currencyMu.RUnlock()
	if fresh {
		return false
	}
	body, err := HttpGet(ecbRatesURL)
	if err != nil {
		return false
	}
	var envelope ecbEnvelope
	if xml.Unmarshal([]byte(body), &envelope) != nil || len(envelope.Cube.Day.Rates) == 0 {
		return false
	}
	rates := make(map[string]float64, len(envelope.Cube.Day.Rates)+1)
	rates["EUR"] = 1
	for _, rate := range envelope.Cube.Day.Rates {
		if rate.Rate > 0 {
			rates[strings.ToUpper(rate.Currency)] = rate.Rate
		}
	}
	now := time.Now()
	currencyMu.Lock()
	currencyRates = rates
	currencyUpdated = now
	currencyMu.Unlock()
	_ = saveCurrencyCache(currencyCache{Updated: now, Rates: rates})
	return true
}

func ConvertCurrency(query string) string {
	match := currencyPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(query)))
	if match == nil {
		return ""
	}
	amount, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", "."), 64)
	if err != nil {
		return ""
	}
	from, to := strings.ToUpper(match[2]), strings.ToUpper(match[3])
	currencyMu.RLock()
	fromRate, fromOK := currencyRates[from]
	toRate, toOK := currencyRates[to]
	currencyMu.RUnlock()
	if !fromOK || !toOK || fromRate == 0 {
		return ""
	}
	return formatNumber(amount/fromRate*toRate) + " " + to
}

func saveCurrencyCache(cached currencyCache) error {
	path, err := currencyCachePath()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	err = json.NewEncoder(file).Encode(cached)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func currencyCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "winfastnav", "currency-rates.json"), nil
}
