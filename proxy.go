package pisigo

import (
	"fmt"
	"net"
	"strings"
)

type ProxyConfig struct {
	TrustedCIDRs []string
	Headers      []string
}

func DefaultProxyConfig() ProxyConfig {
	return ProxyConfig{
		Headers: []string{"X-Forwarded-For", "X-Real-IP"},
	}
}

func (a *App) SetTrustedProxies(cidrs ...string) error {
	parsed := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		if strings.Contains(cidr, "/") {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				return err
			}
			parsed = append(parsed, network)
			continue
		}
		ip := net.ParseIP(cidr)
		if ip == nil {
			return fmt.Errorf("pisigo: invalid proxy ip/cidr %q", cidr)
		}
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		parsed = append(parsed, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}
	a.trustedProxies = parsed
	return nil
}

func (a *App) SetProxyHeaders(headers ...string) {
	a.proxyHeaders = headers
}

func (c *Context) clientIP(trusted []*net.IPNet, headers []string) string {
	remote := remoteIP(c.request.RemoteAddr)
	if len(trusted) == 0 || !ipTrusted(remote, trusted) {
		return remote
	}
	if len(headers) == 0 {
		headers = DefaultProxyConfig().Headers
	}
	for _, h := range headers {
		val := c.request.Header.Get(h)
		if val == "" {
			continue
		}
		if ip := clientIPFromForwarded(val, trusted); ip != "" {
			return ip
		}
	}
	return remote
}

// clientIPFromForwarded walks X-Forwarded-For from right to left, skipping
// trusted hops, and returns the first untrusted IP (the real client).
func clientIPFromForwarded(header string, trusted []*net.IPNet) string {
	parts := strings.Split(header, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(parts[i])
		if candidate == "" {
			continue
		}
		if net.ParseIP(candidate) == nil {
			continue
		}
		if ipTrusted(candidate, trusted) {
			continue
		}
		return candidate
	}
	// All hops trusted: use the leftmost (original) address if valid.
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if net.ParseIP(candidate) != nil {
			return candidate
		}
	}
	return ""
}

func remoteIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func ipTrusted(ipStr string, trusted []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, network := range trusted {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
