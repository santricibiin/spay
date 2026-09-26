package key

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/hirotomasato/paygatemeapp/internal/secrets"
)

const prefix = "pg_live_"

// Generate creates a new server key, returning the plaintext (shown once), its
// SHA-256 hash (stored for incoming API auth), and the AES-GCM-encrypted
// plaintext (stored for signing outgoing webhooks).
func Generate(encryptKeyHex string) (plain, hash string, enc []byte, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", nil, err
	}
	plain = prefix + base64.RawURLEncoding.EncodeToString(raw)
	enc, err = secrets.Encrypt(encryptKeyHex, []byte(plain))
	if err != nil {
		return "", "", nil, err
	}
	return plain, Hash(plain), enc, nil
}

// Hash returns the SHA-256 hex of a server key.
func Hash(k string) string {
	h := sha256.Sum256([]byte(k))
	return fmt.Sprintf("%x", h)
}
