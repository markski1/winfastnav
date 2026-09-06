package utils

import (
	"encoding/xml"
	"testing"
	"time"
)

func TestPercentageAnswers(t *testing.T) {
	tests := map[string]string{
		"20% of 250":         "50",
		"25 percent off 80":  "60",
		"increase 50 by 10%": "55",
		"decrease 50 by 10%": "45",
		"50 + 10%":           "55",
		"50 - 10%":           "45",
	}
	for query, want := range tests {
		got, ok := percentageAnswer(query)
		if !ok || got != want {
			t.Errorf("percentageAnswer(%q) = %q, %v; want %q, true", query, got, ok, want)
		}
	}
}

func TestDateAnswers(t *testing.T) {
	now := time.Date(2026, time.September, 6, 14, 30, 0, 0, time.UTC)
	tests := map[string]string{
		"today":           "Sunday, 6 September 2026",
		"tomorrow":        "Monday, 7 September 2026",
		"3 days ago":      "Thursday, 3 September 2026",
		"in 2 weeks":      "Sunday, 20 September 2026",
		"1 year from now": "Monday, 6 September 2027",
	}
	for query, want := range tests {
		got, ok := instantAnswerAt(query, now)
		if !ok || got != want {
			t.Errorf("instantAnswerAt(%q) = %q, %v; want %q, true", query, got, ok, want)
		}
	}
}

func TestDirectedUnitConversions(t *testing.T) {
	tests := map[string]string{
		"10 km to miles": "6.21371192237 mi",
		"32 f to c":      "0 °C",
		"1 GiB to MiB":   "1024 MiB",
		"2 hours in min": "120 min",
	}
	for query, want := range tests {
		if got := ConvertUnit(query); got != want {
			t.Errorf("ConvertUnit(%q) = %q, want %q", query, got, want)
		}
	}
}

func TestCurrencyConversionUsesCachedRates(t *testing.T) {
	currencyMu.Lock()
	oldRates, oldUpdated := currencyRates, currencyUpdated
	currencyRates = map[string]float64{"EUR": 1, "USD": 1.2, "GBP": 0.8}
	currencyUpdated = time.Now()
	currencyMu.Unlock()
	t.Cleanup(func() {
		currencyMu.Lock()
		currencyRates, currencyUpdated = oldRates, oldUpdated
		currencyMu.Unlock()
	})

	got := ConvertCurrency("120 usd to gbp")
	if got != "80 GBP" {
		t.Fatalf("currency conversion = %q, want 80 GBP", got)
	}
}

func TestECBRatesDocumentParsing(t *testing.T) {
	document := `<Envelope><Cube><Cube time="2026-09-04"><Cube currency="USD" rate="1.2"/><Cube currency="GBP" rate="0.8"/></Cube></Cube></Envelope>`
	var envelope ecbEnvelope
	if err := xml.Unmarshal([]byte(document), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Cube.Day.Rates) != 2 || envelope.Cube.Day.Rates[0].Currency != "USD" {
		t.Fatalf("parsed rates = %#v", envelope.Cube.Day.Rates)
	}
}

func TestArithmeticSupportsGroupingAndPowers(t *testing.T) {
	tests := map[string]string{
		"(2 + 3) * 4": "20",
		"2^3^2":       "512",
		"10 / -2":     "-5",
		"-2^2":        "-4",
	}
	for expression, want := range tests {
		got, err := EvalMath(expression)
		if err != nil || got != want {
			t.Errorf("EvalMath(%q) = %q, %v; want %q", expression, got, err, want)
		}
	}
}

func BenchmarkInstantPercentageAnswer(b *testing.B) {
	for b.Loop() {
		_, _ = InstantAnswer("20% of 250")
	}
}

func BenchmarkArithmeticAnswer(b *testing.B) {
	for b.Loop() {
		_, _ = EvalMath("(2 + 3)^4 / 5")
	}
}
