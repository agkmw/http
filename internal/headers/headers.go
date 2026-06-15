package headers

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const crlf = "\r\n"

type Headers map[string]string

func NewHeaders() Headers {
	return make(Headers)
}

func (h Headers) Get(key string) string {
	if value, exists := h[strings.ToLower(key)]; exists {
		return value
	}
	return ""
}

func (h Headers) Add(key, value string) {
	if old, exists := h[strings.ToLower(key)]; exists {
		value = fmt.Sprintf("%s, %s", old, value)
	}
	h.Set(key, value)
}

func (h Headers) Set(key, value string) {
	h[strings.ToLower(key)] = value
}

func (h Headers) Remove(key string) {
	delete(h, strings.ToLower(key))
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	i := strings.Index(string(data), crlf)
	switch i {
	case -1:
		return 0, false, nil
	case 0:
		return 2, true, nil
	default:
		header := string(data[:i])
		key, value, ok := strings.Cut(header, ":")
		if !ok {
			return 0, false, errors.New("malformed field line")
		}
		if !isValidKey(key) {
			return 0, false, errors.New("malformed key")
		}

		key = strings.ToLower(key)
		value = strings.TrimSpace(value)
		if v, exists := h[key]; exists {
			value = fmt.Sprintf("%s, %s", v, value)
		}
		h[key] = value
		return len(header) + 2, false, nil
	}
}

// a field-name must contain only:
//   - uppercase letters: A-Z
//   - lowercase letters: a-z
//   - digits: 0-9
//   - special characters: "!" / "#" / "$" / "%" / "&" / "'" / "*"
//     / "+" / "-" / "." / "^" / "_" / "`" / "|" / "~"
func isValidKey(key string) bool {
	if len(key) < 1 {
		return false
	}
	for _, c := range key {
		if isLetter(c) || isDigit(c) || isAllowed(c) {
			continue
		}
		return false
	}
	return true
}

func isAllowed(c rune) bool {
	allowedChars := []rune{'!', '#', '$', '%', '&', '\'', '*',
		'+', '-', '.', '^', '_', '`', '|', '~'}
	return slices.Contains(allowedChars, c)
}

func isLetter(c rune) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

func isDigit(c rune) bool {
	return '0' <= c && c <= '9'
}
