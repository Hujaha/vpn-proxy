package xray

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
)

// RealityKeys generates an X25519 keypair encoded in URL-safe base64
// (matching the format produced by `xray x25519`).
type RealityKeys struct {
	Private string `json:"private_key"`
	Public  string `json:"public_key"`
}

func GenerateRealityKeys() (RealityKeys, error) {
	curve := ecdh.X25519()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return RealityKeys{}, err
	}
	enc := base64.RawURLEncoding
	return RealityKeys{
		Private: enc.EncodeToString(priv.Bytes()),
		Public:  enc.EncodeToString(priv.PublicKey().Bytes()),
	}, nil
}
