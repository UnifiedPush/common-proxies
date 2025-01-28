package vapid

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
)

func GenerateKey(rand io.Reader) (private *ecdsa.PrivateKey, err error) {
	private, err = ecdsa.GenerateKey(elliptic.P256(), rand)
	return
}

func EncodePriv(private ecdsa.PrivateKey) (encoded string, err error) {
	marshalled, err := x509.MarshalECPrivateKey(&private)
	bEncoded := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: marshalled})
	encoded = string(bEncoded)
	return
}

func DecodePriv(encoded []byte) (private *ecdsa.PrivateKey, err error) {
	block, _ := pem.Decode(encoded)
	private, err = x509.ParseECPrivateKey(block.Bytes)
	return
}

// Returns the public key encoded in the uncompressed form,
// base64 url safe encoded
func EncodePub(public ecdsa.PublicKey) (encoded string, err error) {
	ecdh, err := public.ECDH()
	encoded = base64.RawURLEncoding.EncodeToString(ecdh.Bytes())
	return
}

// Used to test with jwt.io
func EncodePubPEM(public ecdsa.PublicKey) (pemstr string, err error) {
	raw, err := x509.MarshalPKIXPublicKey(&public)
	encoded := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: raw})
	pemstr = string(encoded)
	return
}

func Sign(rand io.Reader, private ecdsa.PrivateKey, toSign []byte) (signature string, err error) {
	hasher := sha256.Sum256(toSign)

	r, s, err := ecdsa.Sign(rand, &private, hasher[:])
	if err != nil {
		return "", err
	}

	raw_signature := r.Bytes()
	raw_signature = append(raw_signature, s.Bytes()...)
	signature = base64.RawURLEncoding.EncodeToString(raw_signature)
	return
}

func GenAuth(rand io.Reader, private ecdsa.PrivateKey, aud string, exp int) (out string, err error) {
	header := map[string]interface{}{
		"alg": "ES256",
		"typ": "JWT",
	}
	body := map[string]interface{}{
		"aud": aud,
		"exp": exp,
		"sub": "https://codeberg.org/UnifiedPush/common-proxies",
	}
	header_str, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	body_str, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	jwt := fmt.Sprintf("%s.%s", base64.RawURLEncoding.EncodeToString(header_str), base64.RawURLEncoding.EncodeToString(body_str))

	signature, err := Sign(rand, private, []byte(jwt))
	jwt = fmt.Sprintf("%s.%s", jwt, signature)
	pubkey, err := EncodePub(private.PublicKey)
	out = fmt.Sprintf("vapid t=%s,k=%s", jwt, pubkey)
	return
}
