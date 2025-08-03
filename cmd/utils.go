package main

import "fmt"

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var revIndex = func() [256]byte {
	var rev [256]byte
	for i := range rev {
		rev[i] = 0xFF
	}
	for i, c := range alphabet {
		rev[c] = byte(i)
	}
	return rev
}()

func EncodeInt64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	pos := len(buf)
	for n > 0 {
		pos--
		//remainder := n % 62
		//buf[pos] = alphabet[remainder]
		buf[pos] = alphabet[n%62]
		n /= 62
	}
	return string(buf[pos:])
}

func DecodeString(s string) (int64, error) {
	var result int64
	for i := 0; i < len(s); i++ {
		char := s[i]
		if revIndex[char] == 0xFF {
			return 0, fmt.Errorf("invalid character '%c' in string", char)
		}
		result = result*62 + int64(revIndex[char])
	}
	return result, nil
}
