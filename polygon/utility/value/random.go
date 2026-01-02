package value

import (
	crand "crypto/rand"
	"math/big"
	"math/rand"
	"strings"
	"time"
)

const (
	RandomNumber            = "0123456789"
	RandomMixedCaseAlpha    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	RandomLowercaseAlpha    = "abcdefghijklmnopqrstuvwxyz"
	RandomUppercaseAlpha    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	RandomMixedCaseAlphaNum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	RandomLowercaseAlphaNum = "abcdefghijklmnopqrstuvwxyz0123456789"
	RandomUppercaseAlphaNum = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	RandomHex               = "0123456789abcdef"
	RandomUppercaseHex      = "0123456789ABCDEF"
)

var Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func Random(characters string, number int) *string {
	var generated strings.Builder
	for range number {
		random := Rand.Intn(len(characters))
		randomChar := characters[random]
		generated.WriteString(string(randomChar))
	}

	var str = generated.String()
	return &str
}

func RandomSecure(characters string, number int) *string {
	var generated strings.Builder
	for range number {
		index, _ := crand.Int(crand.Reader, big.NewInt(int64(len(characters))))
		generated.WriteByte(characters[index.Int64()])
	}

	result := generated.String()
	return &result
}
