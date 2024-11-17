package stringutil

import (
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func FirstUpper(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// FirstLower
func FirstLower(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func OllamaHost(host string) *url.URL {
	// default url
	if host == "" || host == "://:" {
		host = "http://127.0.0.1:11434"
	}

	// default port
	defaultPort := "11434"

	// default scheme
	scheme, hostport, ok := strings.Cut(host, "://")
	switch {
	case !ok:
		scheme, hostport = "http", host
	case scheme == "http":
		defaultPort = "80"
	case scheme == "https":
		defaultPort = "443"
	}

	// trim trailing slashes
	hostport = strings.TrimRight(hostport, "/")

	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		host, port = "127.0.0.1", defaultPort
		if ip := net.ParseIP(strings.Trim(hostport, "[]")); ip != nil {
			host = ip.String()
		} else if hostport != "" {
			host = hostport
		}
	}

	if n, err := strconv.ParseInt(port, 10, 32); err != nil || n > 65535 || n < 0 {
		log.Println("invalid port, using default", "port", port, "default", defaultPort)
		return &url.URL{
			Scheme: scheme,
			Host:   net.JoinHostPort(host, defaultPort),
		}
	}

	return &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
	}
}

func SanitizeFileName(fileName string) string {
	// define illegal characters
	illegalChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}

	result := fileName
	// replace
	for _, char := range illegalChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	// trim space
	result = strings.TrimSpace(result)

	// if file name is empty, use default name
	if result == "" {
		result = "untitled"
	}

	// limit file name length (optional, Windows max path length is 255 characters)
	if len(result) > 200 {
		result = result[:200]
	}

	return result
}
