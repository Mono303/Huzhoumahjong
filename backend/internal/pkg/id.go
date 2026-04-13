package pkg

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

const roomAlphabet = "23456789"

func NewID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s_fallback", prefix)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf))
}

func NewToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return NewID("token")
	}
	return hex.EncodeToString(buf)
}

func NewRoomCode(length int) string {
	if length < 4 {
		length = 6
	}

	var builder strings.Builder
	builder.Grow(length)
	for range length {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(roomAlphabet))))
		if err != nil {
			builder.WriteByte(roomAlphabet[0])
			continue
		}
		builder.WriteByte(roomAlphabet[index.Int64()])
	}
	return builder.String()
}
