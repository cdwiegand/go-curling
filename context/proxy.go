package context

import (
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	curlerrors "github.com/cdwiegand/go-curling/errors"
	"golang.org/x/net/http/httpproxy"
)

func (ctx *CurlContext) configureProxy(transport *http.Transport) *curlerrors.CurlError {
	if ctx.UnixSocket != "" {
		transport.Proxy = nil
		return nil
	}
	if !ctx.ProxySet && !ctx.NoProxySet {
		return nil
	}
	config := httpproxy.FromEnvironment()
	if ctx.ProxySet {
		proxy := ctx.Proxy
		if proxy != "" {
			if !strings.Contains(proxy, "://") {
				proxy = "http://" + proxy
			}
			u, err := url.Parse(proxy)
			if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
				return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "--proxy requires an HTTP or HTTPS proxy URL")
			}
		}
		config.HTTPProxy, config.HTTPSProxy = proxy, proxy
		// An explicit proxy is trusted user input even in a CGI environment.
		config.CGI = false
	}
	if ctx.NoProxySet {
		config.NoProxy = ctx.NoProxy
	}
	bypass := config.NoProxy
	config.NoProxy = ""
	proxyFunc := config.ProxyFunc()
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		if bypassProxy(req.URL.Hostname(), bypass) {
			return nil, nil
		}
		// Go implicitly bypasses loopback even with an empty NO_PROXY. An
		// explicit curl override must also work for localhost, so use this
		// helper only to choose the proxy for the request's scheme.
		return proxyFunc(&url.URL{Scheme: req.URL.Scheme, Host: "proxy-target.invalid"})
	}
	return nil
}

func bypassProxy(host, rules string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	ip, _ := netip.ParseAddr(host)
	for _, rule := range strings.Split(rules, ",") {
		rule = strings.ToLower(strings.TrimSpace(rule))
		if rule == "*" {
			return true
		}
		if rule == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(rule); err == nil {
			if ip.IsValid() && prefix.Contains(ip.Unmap()) {
				return true
			}
			continue
		}
		if ruleIP, err := netip.ParseAddr(strings.Trim(rule, "[]")); err == nil {
			if ip.IsValid() && ip.Unmap() == ruleIP.Unmap() {
				return true
			}
			continue
		}
		rule = strings.TrimSuffix(strings.TrimPrefix(rule, "."), ".")
		if host == rule || strings.HasSuffix(host, "."+rule) {
			return true
		}
	}
	return false
}
