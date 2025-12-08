package wzeromark

import (
	"crypto/hkdf"
	"crypto/sha256"
	"fmt"
	"time"
)

type keyGen interface {
	Generate(timestamp time.Time) ([]byte, error)
}

var _ keyGen = (*hkdfKeyGen)(nil)

type hkdfKeyGen struct {
	ikm        []byte
	salt       []byte
	infoPrefix string
}

const (
	hmacKey    = "W-ZeroAPI-HMAC-Key-V1"
	ed25519Key = "W-ZeroAPI-Ed25519-Seed-V1"
	keyLen     = 32
)

func newHmacKeygen(orgMasterKey, systemSolt []byte) *hkdfKeyGen {
	return &hkdfKeyGen{
		ikm:        orgMasterKey,
		salt:       systemSolt,
		infoPrefix: hmacKey,
	}
}

func newEd25519Keygen(orgMasterKey, systemSolt []byte) *hkdfKeyGen {
	return &hkdfKeyGen{
		ikm:        orgMasterKey,
		salt:       systemSolt,
		infoPrefix: ed25519Key,
	}
}

func (k *hkdfKeyGen) Generate(timestamp time.Time) ([]byte, error) {
	info := fmt.Sprintf("%s-%s", k.infoPrefix, timestamp.UTC().Format("2006010215"))
	return hkdf.Key(sha256.New, k.ikm, k.salt, info, keyLen)
}
