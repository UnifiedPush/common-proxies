package vapid

import (
	"crypto/rand"
	"log"
	"testing"
	"time"
)

func TestLoadSign(t *testing.T) {
	private, err := GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Cannot generate key: %s", err)
	}
	out, err := EncodePriv(*private)
	if err != nil {
		t.Fatalf("Cannot encode privkey: %s", err)
	}
	log.Println(out)
	out, err = EncodePub(private.PublicKey)
	if err != nil {
		t.Fatalf("Cannot encode pubkey: %s", err)
	}
	log.Println("Pub: " + out)
	out, err = EncodePubPEM(private.PublicKey)
	if err != nil {
		t.Fatalf("Cannot encode pubkey to PEM: %s", err)
	}
	log.Println(out)
}

func TestGenAuth(t *testing.T) {
	private, err := GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Cannot generate key: %s", err)
	}
	out, err := EncodePubPEM(private.PublicKey)
	if err != nil {
		t.Fatalf("Cannot encode pubkey to PEM: %s", err)
	}
	log.Println(out)
	auth, err := GenAuth(rand.Reader, *private, "http://localhost", int(time.Now().Add(2*time.Hour).Unix()))
	if err != nil {
		t.Fatalf("Cannot gen auth: %s", err)
	}
	log.Println(auth)
}
