package proxy

import (
	"net/http"
	"net/url"
	"sync"
)

var (
    once     sync.Once
    instance *ProxyManager
)

type ProxyManager struct {
    proxyURL *url.URL
}

func GetInstance() *ProxyManager {
    once.Do(func() {
        instance = &ProxyManager{}
    })
    return instance
}

func (pm *ProxyManager) SetProxy(proxyAddr string) error {
    if proxyAddr == "" {
        pm.proxyURL = nil
        return nil
    }

    proxyURL, err := url.Parse(proxyAddr)
    if err != nil {
        return err
    }

    pm.proxyURL = proxyURL
    return nil
}

func (pm *ProxyManager) GetProxyURL() *url.URL {
    return pm.proxyURL
}

func (pm *ProxyManager) GetProxyFunc() func(*http.Request) (*url.URL, error) {
    return func(*http.Request) (*url.URL, error) {
        return pm.proxyURL, nil
    }
}

// GetDefaultTransport get default transport
func (pm *ProxyManager) GetDefaultTransport() *http.Transport {
    return &http.Transport{
        Proxy: pm.GetProxyFunc(),
    }
}

// GetDefaultClient get default client
func (pm *ProxyManager) GetDefaultClient() *http.Client {
    return &http.Client{
        Transport: pm.GetDefaultTransport(),
    }
}