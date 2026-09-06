package utils

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	percentOfPattern = regexp.MustCompile(`^([-+]?\d+(?:[.,]\d+)?)\s*(?:%|percent)\s+(of|off)\s+([-+]?\d+(?:[.,]\d+)?)$`)
	changePattern    = regexp.MustCompile(`^(increase|decrease)\s+([-+]?\d+(?:[.,]\d+)?)\s+by\s+([-+]?\d+(?:[.,]\d+)?)\s*(?:%|percent)$`)
	relativePercent  = regexp.MustCompile(`^([-+]?\d+(?:[.,]\d+)?)\s*([+-])\s*([-+]?\d+(?:[.,]\d+)?)\s*%$`)
	relativePattern  = regexp.MustCompile(`^(?:in\s+)?(\d+)\s+(day|days|week|weeks|month|months|year|years)(?:\s+from\s+now)?$`)
	agoPattern       = regexp.MustCompile(`^(\d+)\s+(day|days|week|weeks|month|months|year|years)\s+ago$`)
)

func InstantAnswer(query string) (string, bool) {
	return instantAnswerAt(query, time.Now())
}

func instantAnswerAt(query string, now time.Time) (string, bool) {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(query)), " "))
	normalized = strings.TrimPrefix(normalized, "what is ")
	if answer, ok := percentageAnswer(normalized); ok {
		return answer, true
	}
	if answer := ConvertCurrency(normalized); answer != "" {
		return answer, true
	}
	if answer, ok := dateTimeAnswer(normalized, now); ok {
		return answer, true
	}
	return "", false
}

func percentageAnswer(query string) (string, bool) {
	if match := percentOfPattern.FindStringSubmatch(query); match != nil {
		percent, ok1 := parseNumber(match[1])
		value, ok2 := parseNumber(match[3])
		if !ok1 || !ok2 {
			return "", false
		}
		result := value * percent / 100
		if match[2] == "off" {
			result = value - result
		}
		return formatNumber(result), true
	}
	if match := changePattern.FindStringSubmatch(query); match != nil {
		value, ok1 := parseNumber(match[2])
		percent, ok2 := parseNumber(match[3])
		if !ok1 || !ok2 {
			return "", false
		}
		change := value * percent / 100
		if match[1] == "decrease" {
			change = -change
		}
		return formatNumber(value + change), true
	}
	if match := relativePercent.FindStringSubmatch(query); match != nil {
		value, ok1 := parseNumber(match[1])
		percent, ok2 := parseNumber(match[3])
		if !ok1 || !ok2 {
			return "", false
		}
		change := value * percent / 100
		if match[2] == "-" {
			change = -change
		}
		return formatNumber(value + change), true
	}
	return "", false
}

func dateTimeAnswer(query string, now time.Time) (string, bool) {
	switch query {
	case "time", "current time", "local time":
		return now.Format("15:04, Monday, 2 January 2006"), true
	case "date", "today", "today's date":
		return now.Format("Monday, 2 January 2006"), true
	case "tomorrow":
		return now.AddDate(0, 0, 1).Format("Monday, 2 January 2006"), true
	case "yesterday":
		return now.AddDate(0, 0, -1).Format("Monday, 2 January 2006"), true
	}

	if match := relativePattern.FindStringSubmatch(query); match != nil {
		return relativeDate(now, match[1], match[2], 1)
	}
	if match := agoPattern.FindStringSubmatch(query); match != nil {
		return relativeDate(now, match[1], match[2], -1)
	}
	return "", false
}

func relativeDate(now time.Time, amountText, unit string, direction int) (string, bool) {
	amount, err := strconv.Atoi(amountText)
	if err != nil || amount > 10000 {
		return "", false
	}
	amount *= direction
	var result time.Time
	switch strings.TrimSuffix(unit, "s") {
	case "day":
		result = now.AddDate(0, 0, amount)
	case "week":
		result = now.AddDate(0, 0, amount*7)
	case "month":
		result = now.AddDate(0, amount, 0)
	case "year":
		result = now.AddDate(amount, 0, 0)
	default:
		return "", false
	}
	return result.Format("Monday, 2 January 2006"), true
}

func parseNumber(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
	return parsed, err == nil
}

func formatNumber(value float64) string {
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'g', 12, 64)
}
