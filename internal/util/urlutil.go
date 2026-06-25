package util

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateURL 校验 URL 是否合法，不合法返回错误描述
func ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL 格式不正确")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https 链接")
	}

	if u.User != nil {
		return fmt.Errorf("不支持带认证的 URL")
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL 缺少域名")
	}

	if u.Port() != "" {
		return fmt.Errorf("不支持带端口的 URL")
	}

	if net.ParseIP(host) != nil || host == "localhost" {
		return fmt.Errorf("不支持 IP 地址")
	}

	return nil
}

// NormalizeURL 将 URL 归一化：去协议、去 www、去尾部斜杠、去 fragment、域名转小写
func NormalizeURL(rawURL string) (string, error) {
	if err := ValidateURL(rawURL); err != nil {
		return "", err
	}

	u, _ := url.Parse(rawURL)

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

// ExtractDomain 从 URL 中提取归一化后的域名（去 www、转小写）
func ExtractDomain(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("URL 格式不正确")
	}

	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("URL 缺少域名")
	}

	host = strings.ToLower(host)
	host = strings.TrimPrefix(host, "www.")

	return host, nil
}
