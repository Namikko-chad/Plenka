package indexers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// Download tokens are self-contained: base64url(JSON payload) + "." + base64url(HMAC).
// The HMAC covers the indexer id, so a token only resolves through the indexer that
// produced it and cannot be forged to make the server fetch arbitrary URLs.

type tokenPayload struct {
	U string `json:"u"` // details or download URL
}

var errBadToken = errors.New("invalid or tampered download token")

func signToken(key []byte, indexerID, link string) string {
	payload, _ := json.Marshal(tokenPayload{U: link})
	p := base64.RawURLEncoding.EncodeToString(payload)
	return p + "." + base64.RawURLEncoding.EncodeToString(tokenMAC(key, indexerID, p))
}

func verifyToken(key []byte, indexerID, token string) (string, error) {
	p, sig, ok := strings.Cut(token, ".")
	if !ok {
		return "", errBadToken
	}
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(got, tokenMAC(key, indexerID, p)) {
		return "", errBadToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(p)
	if err != nil {
		return "", errBadToken
	}
	var tp tokenPayload
	if err := json.Unmarshal(raw, &tp); err != nil || tp.U == "" {
		return "", errBadToken
	}
	return tp.U, nil
}

func tokenMAC(key []byte, indexerID, payload string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(indexerID))
	m.Write([]byte{0})
	m.Write([]byte(payload))
	return m.Sum(nil)[:18]
}
