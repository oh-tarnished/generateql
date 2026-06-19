// URL construction shared by all client types.
package network

import (
	"fmt"
	"net/url"
)

// buildFullURL constructs a full URL from URLOptions using the path at pathIndex.
// It validates the scheme (http, https, ws, wss), a non-empty host, a non-empty
// paths list, and that pathIndex is in range. Query parameters from Params are
// appended when present.
func buildFullURL(urlOptions URLOptions, pathIndex int) (string, error) {
	switch urlOptions.Scheme {
	case HTTP, HTTPS, WS, WSS:
	default:
		return "", fmt.Errorf("invalid URL scheme: %s. Must be 'http', 'https', 'ws', or 'wss'", urlOptions.Scheme)
	}
	if urlOptions.Host == "" {
		return "", fmt.Errorf("host cannot be empty")
	}
	if len(urlOptions.Paths) == 0 {
		return "", fmt.Errorf("paths array cannot be empty")
	}
	if pathIndex < 0 || pathIndex >= len(urlOptions.Paths) {
		return "", fmt.Errorf("pathIndex %d out of bounds for paths array of length %d", pathIndex, len(urlOptions.Paths))
	}

	u := url.URL{Scheme: string(urlOptions.Scheme), Host: urlOptions.Host}

	// Ensure the path starts with a forward slash without re-encoding it.
	path := urlOptions.Paths[pathIndex]
	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}
	u.Path = path

	if len(urlOptions.Params) > 0 {
		query := u.Query()
		for key, value := range urlOptions.Params {
			query.Set(key, value)
		}
		u.RawQuery = query.Encode()
	}

	return u.String(), nil
}

// websocketURL derives a ws/wss URLOptions copy from an http/https URLOptions,
// used to open a subscription transport against the same host and path.
func websocketURL(in URLOptions) URLOptions {
	out := in
	switch in.Scheme {
	case HTTPS, WSS:
		out.Scheme = WSS
	default:
		out.Scheme = WS
	}
	return out
}
