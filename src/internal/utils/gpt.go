package utils

import (
	"net/url"
)

func MakeGPTReq(prompt string) string {
	data, err := HttpGet("https://markski.ar/gpt-req.php?prompt=" + url.QueryEscape(prompt))

	if err != nil {
		return "Sorry, there was an error making the request."
	}

	return WrapTextByWords(data, 64)
}
