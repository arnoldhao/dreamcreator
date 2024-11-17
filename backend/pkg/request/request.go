package request

import (
	"CanMe/backend/pkg/specials/proxy"
	"compress/flate"
	"compress/gzip"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	cookiemonster "github.com/MercuryEngineering/CookieMonster"
	"github.com/pkg/errors"
)

// fakeHeaders fake http headers
var fakeHeaders = map[string]string{
	"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	"Accept-Charset":  "UTF-8,*;q=0.5",
	"Accept-Encoding": "gzip,deflate,sdch",
	"Accept-Language": "en-US,en;q=0.8",
	"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.81 Safari/537.36",
}

type Service struct {
	RetryTimes int
	Cookie     string
	UserAgent  string
	Refer      string
	ProxyFunc  func(*http.Request) (*url.URL, error)
}

func New(retryTimes int, cookie, userAgent, refer string, proxyFunc func(*http.Request) (*url.URL, error)) *Service {
	return &Service{
		RetryTimes: retryTimes,
		Cookie:     cookie,
		UserAgent:  userAgent,
		Refer:      refer,
		ProxyFunc:  proxyFunc,
	}
}

func FastNew() *Service {
	return &Service{
		RetryTimes: 3,
		ProxyFunc:  proxy.GetInstance().GetProxyFunc(),
	}
}

// Request base request
func (s *Service) Request(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	transport := &http.Transport{
		Proxy:               s.ProxyFunc,
		DisableCompression:  true,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Minute,
		Jar:       jar,
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for k, v := range fakeHeaders {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if _, ok := headers["Referer"]; !ok {
		req.Header.Set("Referer", url)
	}
	if s.Cookie != "" {
		// parse cookies in Netscape HTTP cookie format
		cookies, _ := cookiemonster.ParseString(s.Cookie)
		if len(cookies) > 0 {
			for _, c := range cookies {
				req.AddCookie(c)
			}
		} else {
			// cookie is not Netscape HTTP format, set it directly
			// a=b; c=d
			req.Header.Set("Cookie", s.Cookie)
		}
	}

	if s.UserAgent != "" {
		req.Header.Set("User-Agent", s.UserAgent)
	}

	if s.Refer != "" {
		req.Header.Set("Referer", s.Refer)
	}

	var (
		res          *http.Response
		requestError error
	)
	for i := 0; ; i++ {
		res, requestError = client.Do(req)
		if requestError == nil && res.StatusCode < 400 {
			break
		} else if i+1 >= s.RetryTimes {
			var err error
			if requestError != nil {
				err = errors.Errorf("request error: %v", requestError)
			} else {
				err = errors.Errorf("%s request error: HTTP %d", url, res.StatusCode)
			}
			return nil, errors.WithStack(err)
		}
		time.Sleep(1 * time.Second)
	}

	return res, nil
}

// Get get request
func (s *Service) Get(url, refer string, headers map[string]string) (string, error) {
	body, err := s.GetByte(url, refer, headers)
	return string(body), err
}

// GetByte get request
func (s *Service) GetByte(url, refer string, headers map[string]string) ([]byte, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	if refer != "" {
		headers["Referer"] = refer
	}
	res, err := s.Request(http.MethodGet, url, nil, headers)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer res.Body.Close() // nolint

	var reader io.ReadCloser
	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ = gzip.NewReader(res.Body)
	case "deflate":
		reader = flate.NewReader(res.Body)
	default:
		reader = res.Body
	}
	defer reader.Close() // nolint

	body, err := io.ReadAll(reader)
	if err != nil && err != io.EOF {
		return nil, errors.WithStack(err)
	}
	return body, nil
}

// Headers return the HTTP Headers of the url
func (s *Service) Headers(url, refer string) (http.Header, error) {
	headers := map[string]string{
		"Referer": refer,
	}
	res, err := s.Request(http.MethodGet, url, nil, headers)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer res.Body.Close() // nolint
	return res.Header, nil
}

// Size get size of the url
func (s *Service) Size(url, refer string) (int64, error) {
	h, err := s.Headers(url, refer)
	if err != nil {
		return 0, err
	}
	contentLength := h.Get("Content-Length")
	if contentLength == "" {
		return 0, errors.New("Content-Length is not present")
	}
	size, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return 0, err
	}
	return size, nil
}

// ContentType get Content-Type of the url
func (s *Service) ContentType(url, refer string) (string, error) {
	h, err := s.Headers(url, refer)
	if err != nil {
		return "", err
	}
	contentType := h.Get("Content-Type")
	// handle Content-Type like this: "text/html; charset=utf-8"
	return strings.Split(contentType, ";")[0], nil
}
