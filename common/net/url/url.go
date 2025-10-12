// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package url

import (
	"errors"
	"fmt"
	"net/netip"
	neturl "net/url"
	"strings"
	_ "unsafe"
)

//go:linkname setFragment net/url.(*URL).setFragment
func setFragment(u *neturl.URL, fragment string) error

//go:linkname setPath net/url.(*URL).setPath
func setPath(u *neturl.URL, fragment string) error

func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	default:
		panic("invalid hex character")
	}
}

type encoding int

const (
	encodePath encoding = 1 + iota
	encodePathSegment
	encodeHost
	encodeZone
	encodeUserPassword
	encodeQueryComponent
	encodeFragment
)

func shouldEscape(c byte, mode encoding) bool {
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
		return false
	}

	if mode == encodeHost || mode == encodeZone {
		switch c {
		case '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=', ':', '[', ']', '<', '>', '"':
			return false
		}
	}

	switch c {
	case '-', '_', '.', '~':
		return false

	case '$', '&', '+', ',', '/', ':', ';', '=', '?', '@':
		switch mode {
		case encodePath:
			return c == '?'

		case encodePathSegment:
			return c == '/' || c == ';' || c == ',' || c == '?'

		case encodeUserPassword:
			return c == '@' || c == '/' || c == '?' || c == ':'

		case encodeQueryComponent:
			return true

		case encodeFragment:
			return false
		}
	}

	if mode == encodeFragment {
		switch c {
		case '!', '(', ')', '*':
			return false
		}
	}

	return true
}

func unescape(s string, mode encoding) (string, error) {
	n := 0
	hasPlus := false
	for i := 0; i < len(s); {
		switch s[i] {
		case '%':
			n++
			if i+2 >= len(s) || !ishex(s[i+1]) || !ishex(s[i+2]) {
				s = s[i:]
				if len(s) > 3 {
					s = s[:3]
				}
				return "", neturl.EscapeError(s)
			}
			if mode == encodeHost && unhex(s[i+1]) < 8 && s[i:i+3] != "%25" {
				return "", neturl.EscapeError(s[i : i+3])
			}
			if mode == encodeZone {
				v := unhex(s[i+1])<<4 | unhex(s[i+2])
				if s[i:i+3] != "%25" && v != ' ' && shouldEscape(v, encodeHost) {
					return "", neturl.EscapeError(s[i : i+3])
				}
			}
			i += 3
		case '+':
			hasPlus = mode == encodeQueryComponent
			i++
		default:
			if (mode == encodeHost || mode == encodeZone) && s[i] < 0x80 && shouldEscape(s[i], mode) {
				return "", neturl.InvalidHostError(s[i : i+1])
			}
			i++
		}
	}

	if n == 0 && !hasPlus {
		return s, nil
	}

	var t strings.Builder
	t.Grow(len(s) - 2*n)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '%':
			t.WriteByte(unhex(s[i+1])<<4 | unhex(s[i+2]))
			i += 2
		case '+':
			if mode == encodeQueryComponent {
				t.WriteByte(' ')
			} else {
				t.WriteByte('+')
			}
		default:
			t.WriteByte(s[i])
		}
	}
	return t.String(), nil
}

func getScheme(rawURL string) (scheme, path string, err error) {
	for i := 0; i < len(rawURL); i++ {
		c := rawURL[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				return "", rawURL, nil
			}
		case c == ':':
			if i == 0 {
				return "", "", errors.New("missing protocol scheme")
			}
			return rawURL[:i], rawURL[i+1:], nil
		default:
			return "", rawURL, nil
		}
	}
	return "", rawURL, nil
}

func Parse(rawURL string) (*neturl.URL, error) {
	u, frag, _ := strings.Cut(rawURL, "#")
	url, err := parse(u)
	if err != nil {
		return nil, &neturl.Error{Op: "parse", URL: u, Err: err}
	}
	if frag == "" {
		return url, nil
	}
	if err = setFragment(url, frag); err != nil {
		return nil, &neturl.Error{Op: "parse", URL: rawURL, Err: err}
	}
	return url, nil
}

func parse(rawURL string) (*neturl.URL, error) {
	var rest string
	var err error

	if stringContainsCTLByte(rawURL) {
		return nil, errors.New("net/url: invalid control character in URL")
	}

	url := new(neturl.URL)

	if rawURL == "*" {
		url.Path = "*"
		return url, nil
	}

	if url.Scheme, rest, err = getScheme(rawURL); err != nil {
		return nil, err
	}
	url.Scheme = strings.ToLower(url.Scheme)

	if strings.HasSuffix(rest, "?") && strings.Count(rest, "?") == 1 {
		url.ForceQuery = true
		rest = rest[:len(rest)-1]
	} else {
		rest, url.RawQuery, _ = strings.Cut(rest, "?")
	}

	if !strings.HasPrefix(rest, "/") {
		if url.Scheme != "" {
			url.Opaque = rest
			return url, nil
		}

		if segment, _, _ := strings.Cut(rest, "/"); strings.Contains(segment, ":") {
			return nil, errors.New("first path segment in URL cannot contain colon")
		}
	}

	if !strings.HasPrefix(rest, "///") && strings.HasPrefix(rest, "//") {
		var authority string
		authority, rest = rest[2:], ""
		if i := strings.Index(authority, "/"); i >= 0 {
			authority, rest = authority[:i], authority[i:]
		}
		url.User, url.Host, err = parseAuthority(authority)
		if err != nil {
			return nil, err
		}
	} else if url.Scheme != "" && strings.HasPrefix(rest, "/") {
		url.OmitHost = true
	}

	if err := setPath(url, rest); err != nil {
		return nil, err
	}
	return url, nil
}

func parseAuthority(authority string) (user *neturl.Userinfo, host string, err error) {
	i := strings.LastIndex(authority, "@")
	if i < 0 {
		host, err = parseHost(authority)
	} else {
		host, err = parseHost(authority[i+1:])
	}
	if err != nil {
		return nil, "", err
	}
	if i < 0 {
		return nil, host, nil
	}
	userinfo := authority[:i]
	if !validUserinfo(userinfo) {
		return nil, "", errors.New("net/url: invalid userinfo")
	}
	if !strings.Contains(userinfo, ":") {
		if userinfo, err = unescape(userinfo, encodeUserPassword); err != nil {
			return nil, "", err
		}
		user = neturl.User(userinfo)
	} else {
		username, password, _ := strings.Cut(userinfo, ":")
		if username, err = unescape(username, encodeUserPassword); err != nil {
			return nil, "", err
		}
		if password, err = unescape(password, encodeUserPassword); err != nil {
			return nil, "", err
		}
		user = neturl.UserPassword(username, password)
	}
	return user, host, nil
}

func parseHost(host string) (string, error) {
	if openBracketIdx := strings.LastIndex(host, "["); openBracketIdx != -1 {
		closeBracketIdx := strings.LastIndex(host, "]")
		if closeBracketIdx < 0 {
			return "", errors.New("missing ']' in host")
		}

		colonPort := host[closeBracketIdx+1:]
		if !validOptionalPort(colonPort) {
			return "", fmt.Errorf("invalid port %q after host", colonPort)
		}
		unescapedColonPort, err := unescape(colonPort, encodeHost)
		if err != nil {
			return "", err
		}

		hostname := host[openBracketIdx+1 : closeBracketIdx]
		var unescapedHostname string
		zoneIdx := strings.Index(hostname, "%25")
		if zoneIdx >= 0 {
			hostPart, err := unescape(hostname[:zoneIdx], encodeHost)
			if err != nil {
				return "", err
			}
			zonePart, err := unescape(hostname[zoneIdx:], encodeZone)
			if err != nil {
				return "", err
			}
			unescapedHostname = hostPart + zonePart
		} else {
			var err error
			unescapedHostname, err = unescape(hostname, encodeHost)
			if err != nil {
				return "", err
			}
		}

		addr, err := netip.ParseAddr(unescapedHostname)
		if err != nil {
			return "", fmt.Errorf("invalid host: %w", err)
		}
		if addr.Is4() {
			return "", errors.New("invalid IP-literal")
		}
		return "[" + unescapedHostname + "]" + unescapedColonPort, nil
	} else if i := strings.LastIndex(host, ":"); i != -1 {
		colonPort := host[i:]
		if !validOptionalPort(colonPort) {
			return "", fmt.Errorf("invalid port %q after host", colonPort)
		}
	}

	var err error
	if host, err = unescape(host, encodeHost); err != nil {
		return "", err
	}
	return host, nil
}

func validOptionalPort(port string) bool {
	if port == "" {
		return true
	}
	if port[0] != ':' {
		return false
	}
	for _, b := range port[1:] {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

func validUserinfo(s string) bool {
	for _, r := range s {
		if 'A' <= r && r <= 'Z' {
			continue
		}
		if 'a' <= r && r <= 'z' {
			continue
		}
		if '0' <= r && r <= '9' {
			continue
		}
		switch r {
		case '-', '.', '_', ':', '~', '!', '$', '&', '\'',
			'(', ')', '*', '+', ',', ';', '=', '%':
			continue
		case '@':
			continue
		default:
			return false
		}
	}
	return true
}

func stringContainsCTLByte(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < ' ' || b == 0x7f {
			return true
		}
	}
	return false
}
