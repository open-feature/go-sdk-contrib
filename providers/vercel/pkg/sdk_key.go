package vercel

import (
	"net/url"
	"regexp"
	"strings"
)

var sdkKeyRegex = regexp.MustCompile(`^vf_(?:server|client)_`)

// IsValidSDKKey reports whether value looks like a Vercel Flags SDK key.
func IsValidSDKKey(value string) bool {
	return sdkKeyRegex.MatchString(value)
}

// ParseSDKKey accepts either a raw Vercel Flags SDK key or a FLAGS connection
// string such as "flags:edgeConfigId=...&edgeConfigToken=...&sdkKey=vf_server_...".
func ParseSDKKey(value string) (string, bool) {
	if IsValidSDKKey(value) {
		return value, true
	}

	if !strings.HasPrefix(value, "flags:") {
		return "", false
	}

	params, err := url.ParseQuery(strings.TrimPrefix(value, "flags:"))
	if err != nil {
		return "", false
	}

	sdkKey := params.Get("sdkKey")
	if !IsValidSDKKey(sdkKey) {
		return "", false
	}

	return sdkKey, true
}
