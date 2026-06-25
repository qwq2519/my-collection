package util

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateURL 校验 URL 是否合法，不合法返回错误描述
func ValidateURL(rawURL string) error {
	_, err := parseAndValidate(rawURL)
	return err
}

// parseAndValidate 解析并校验 URL，返回已解析的 *url.URL 供后续复用
func parseAndValidate(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("URL 格式不正确")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("仅支持 http/https 链接")
	}

	if u.User != nil {
		return nil, fmt.Errorf("不支持带认证的 URL")
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("URL 缺少域名")
	}

	if u.Port() != "" {
		return nil, fmt.Errorf("不支持带端口的 URL")
	}

	if net.ParseIP(host) != nil || host == "localhost" {
		return nil, fmt.Errorf("不支持 IP 地址")
	}

	return u, nil
}

// NormalizeURL 将 URL 归一化：去协议、去 www、去尾部斜杠、去 fragment、域名转小写
func NormalizeURL(rawURL string) (string, error) {
	u, err := parseAndValidate(rawURL)
	if err != nil {
		return "", err
	}

	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")

	path := u.Path
	path = strings.TrimSuffix(path, "/")

	result := host + path
	if u.RawQuery != "" {
		result += "?" + u.RawQuery
	}

	return result, nil
}

// ExtractDomain 从已校验的 URL 中提取归一化域名（去 www、转小写）
func ExtractDomain(rawURL string) (string, error) {
	u, err := parseAndValidate(rawURL)
	if err != nil {
		return "", err
	}

	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")

	return host, nil
}
