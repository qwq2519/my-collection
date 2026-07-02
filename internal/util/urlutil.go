package util

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// parseAndValidate 解析并校验 URL，返回已解析的 *url.URL 供后续复用
func parseAndValidate(rawURL string) (*url.URL, error) {
	if !strings.Contains(rawURL, "://") {
		return nil, fmt.Errorf("invalid URL format")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("only http/https supported")
	}

	if u.User != nil {
		return nil, fmt.Errorf("URL with credentials not supported")
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("URL missing hostname")
	}

	if u.Port() != "" {
		return nil, fmt.Errorf("URL with port not supported")
	}

	if net.ParseIP(host) != nil || host == "localhost" {
		return nil, fmt.Errorf("IP address not supported")
	}

	return u, nil
}

// normalizeHost 归一化域名：转小写、去 www 前缀
func normalizeHost(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	return strings.TrimPrefix(host, "www.")
}

// ValidateURL 校验 URL 是否合法，不合法返回错误描述。
// 限制：仅允许 http/https、禁止带端口/凭证/IP 地址/localhost。
func ValidateURL(rawURL string) error {
	_, err := parseAndValidate(rawURL)
	return err
}

// NormalizeURL 将 URL 归一化：保留协议、去 www、去尾部斜杠、去 fragment、域名转小写、query 参数排序。
// 输入需通过 ValidateURL 同等校验，不合法时返回 error。
func NormalizeURL(rawURL string) (string, error) {
	u, err := parseAndValidate(rawURL)
	if err != nil {
		return "", err
	}

	result := u.Scheme + "://" + normalizeHost(u) + strings.TrimSuffix(u.Path, "/")
	if u.RawQuery != "" {
		result += "?" + u.Query().Encode()
	}

	return result, nil
}

// ExtractDomain 从 URL 中提取归一化域名（去 www、转小写）。
// 内部包含完整校验，不合法 URL 返回 error。
func ExtractDomain(rawURL string) (string, error) {
	u, err := parseAndValidate(rawURL)
	if err != nil {
		return "", err
	}
	return normalizeHost(u), nil
}
