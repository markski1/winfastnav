package utils

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

var quickAnswerURL = "https://mmip-be.markski.ar/freefastnav"

func QuickAnswer(question string) string {
	req, err := http.NewRequest(http.MethodPost, quickAnswerURL, strings.NewReader(question))
	if err != nil {
		return "Sorry, Quick Answer is unavailable."
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := httpClient.Do(req)

	if err != nil {
		return "Sorry, Quick Answer is unavailable."
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPResponseSize+1))
	if err != nil {
		return "Sorry, Quick Answer returned an unreadable response."
	}
	if len(body) > maxHTTPResponseSize {
		return "Sorry, Quick Answer returned a response that is too large."
	}
	answer := strings.TrimSpace(string(body))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if answer != "" {
			return answer
		}
		return fmt.Sprintf("Quick Answer request failed: %s", resp.Status)
	}
	if answer != "" {
		return answer
	}
	return "Quick Answer returned an empty response."
}
