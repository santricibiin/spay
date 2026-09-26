package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// Sign returns the HMAC-SHA256 signature header value for a payload.
// The store verifies this with their own server key.
func Sign(serverKey, payload []byte) string {
	mac := hmac.New(sha256.New, serverKey)
	mac.Write(payload)
	return "sha256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}