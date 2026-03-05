package weixinmp

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
)

// Signature computes the SHA1 signature used by WeChat for server validation.
//
// It sorts (token, timestamp, nonce) lexicographically, concatenates them,
// then returns sha1 hex digest.
func Signature(token, timestamp, nonce string) string {
	parts := []string{token, timestamp, nonce}
	sort.Strings(parts)
	sum := sha1.Sum([]byte(parts[0] + parts[1] + parts[2]))
	return hex.EncodeToString(sum[:])
}

// VerifySignature compares the computed signature with the provided one.
// Comparison is case-insensitive to tolerate hex casing differences.
func VerifySignature(signature, token, timestamp, nonce string) bool {
	return strings.EqualFold(Signature(token, timestamp, nonce), signature)
}

