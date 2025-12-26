package wzeromark

import (
	"bytes"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const (
	// MarkSize is the length of the watermark, in bits
	MarkSize           = 83 * 8
	markByteLen        = 83
	version1    uint8  = 0b10_000_000
	context     string = "watermark_zero/v1"
)

var (
	ErrInvalidMarkLength = errors.New("invalid mark length")
	ErrInvalidVersion    = errors.New("invalid version")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrInvalidOrgCode    = errors.New("invalid organization code")
)

type WZeroMark struct {
	hmacKeyGen    keyGen
	ed25519KeyGen keyGen
	orgBytes      []byte
	now           func() time.Time
}

// New creates a new WZeroMark instance.
// New returns a new WZeroMark instance for watermark encoding/decoding.
// orgMasterKey and systemSolt are used for key generation, orgCode is a 4-digit hex string representing the organization.
func New(orgMasterKey, systemSolt []byte, orgCode string) (*WZeroMark, error) {
	orgBytes, err := hex.DecodeString(orgCode)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidOrgCode, err)
	}
	if len(orgBytes) != 2 {
		return nil, fmt.Errorf("%w: orgCode must decode to 2 bytes (4 hex chars)", ErrInvalidOrgCode)
	}
	return &WZeroMark{
		hmacKeyGen:    newHmacKeygen(orgMasterKey, systemSolt),
		ed25519KeyGen: newEd25519Keygen(orgMasterKey, systemSolt),
		orgBytes:      orgBytes,
		now:           time.Now,
	}, nil
}

// Encode converts the input string into a watermark byte slice.
// Only the watermark bytes are returned; hash, timestamp, and nonce are not exposed.
func (m *WZeroMark) Encode(src string) (mark []byte, err error) {
	mark = make([]byte, markByteLen)
	err = m.encode(src, mark, nil, nil, nil)
	return
}

// FullEncode converts the input string into a watermark byte slice,
// and also returns the hash (hex), timestamp, and nonce used in the watermark.
func (m *WZeroMark) FullEncode(src string) (mark []byte, hash string, timestamp time.Time, nonce string, err error) {
	mark = make([]byte, markByteLen)
	err = m.encode(src, mark, &hash, &timestamp, &nonce)
	return
}

// Decode returns the hash (hex) from the watermark byte slice.
// Returns the hash if decoding succeeds.
func (m *WZeroMark) Decode(mark []byte) (hash string, err error) {
	err = m.decode(mark, &hash, nil, nil)
	return
}

// FullDecode returns the hash, timestamp, and nonce from the watermark byte slice.
// Returns all decoded values if successful.
func (m *WZeroMark) FullDecode(mark []byte) (hash string, timestamp time.Time, nonce string, err error) {
	err = m.decode(mark, &hash, &timestamp, &nonce)
	return
}

// Verify checks if the watermark byte slice matches the expected hash for the given source string.
// Returns true if the hash matches the hash generated from src and timestamp.
// Also returns the timestamp and nonce.
func (m *WZeroMark) Verify(mark []byte, src string) (ok bool, timestamp time.Time, nonce string, err error) {
	var decoded string
	err = m.decode(mark, &decoded, &timestamp, &nonce)
	if err != nil {
		if errors.Is(err, ErrInvalidSignature) {
			err = nil
			return
		}
		if errors.Is(err, ErrInvalidOrgCode) {
			err = nil
			return
		}
		if errors.Is(err, ErrInvalidVersion) {
			err = nil
			return
		}
		err = fmt.Errorf("failed to decode mark: %w", err)
		return
	}
	_, exp, err := m.encodeSrc(timestamp, src)
	if err != nil {
		err = fmt.Errorf("failed to encode source: %w", err)
		return
	}
	ok = decoded == exp
	return
}

// EqualHash checks if the provided hash matches the hash generated from the source string and timestamp.
// Returns true if hashes match.
func (m *WZeroMark) EqualHash(hash, src string, timestamp time.Time) (ok bool, err error) {
	_, exp, err := m.encodeSrc(timestamp, src)
	if err != nil {
		err = fmt.Errorf("failed to encode source: %w", err)
		return
	}
	ok = hash == exp
	return
}

// PublicKeyAt returns the Ed25519 public key for the given timestamp.
// Note: The public key rotates hourly based on the timestamp.
func (m *WZeroMark) PublicKeyAt(timestamp time.Time) (ed25519.PublicKey, error) {
	edKeySeed, err := m.ed25519KeyGen.Generate(timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key seed: %w", err)
	}
	priv := ed25519.NewKeyFromSeed(edKeySeed)
	pub := priv.Public().(ed25519.PublicKey)
	return pub, nil
}

// encode is an internal method that converts the source string into the watermark byte slice.
// Optionally returns the hash, timestamp, and nonce used in the watermark.
func (m *WZeroMark) encode(src string, mark []byte, hash *string, timestamp *time.Time, nonce *string) error {
	now := m.now()

	// 1. Generate HMAC Hash
	h, hexHash, err := m.encodeSrc(now, src)
	if err != nil {
		return fmt.Errorf("failed to generate HMAC hash: %w", err)
	}

	// 2. Generate Ed25519 Private Key
	edKeySeed, err := m.ed25519KeyGen.Generate(now)
	if err != nil {
		return fmt.Errorf("failed to generate Ed25519 key seed: %w", err)
	}
	priv := ed25519.NewKeyFromSeed(edKeySeed)

	// 3. Create Payload and Sign
	payload := make([]byte, markByteLen)
	payload[0] = version1
	binary.BigEndian.PutUint64(payload[1:9], uint64(now.UnixMilli()<<16))
	_, _ = rand.Read(payload[7:9])
	copy(payload[9:11], m.orgBytes)
	copy(payload[11:19], h)

	sig, err := priv.Sign(nil, payload[:19], &ed25519.Options{
		Context: context,
	})
	if err != nil {
		return fmt.Errorf("failed to sign payload: %w", err)
	}
	copy(payload[19:], sig)

	if len(mark) == markByteLen {
		copy(mark, payload)
	}
	if hash != nil {
		*hash = hexHash
	}
	if timestamp != nil {
		*timestamp = now
	}
	if nonce != nil {
		*nonce = hex.EncodeToString(payload[7:9])
	}
	return nil
}

// encodeSrc generates the HMAC hash for the source string and timestamp.
// Returns the hash bytes and its hex string.
func (m *WZeroMark) encodeSrc(keyClock time.Time, src string) ([]byte, string, error) {
	macKey, err := m.hmacKeyGen.Generate(keyClock)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate HMAC key: %w", err)
	}
	mac := hmac.New(sha256.New, macKey)
	_, _ = mac.Write([]byte(src))
	h := mac.Sum(nil)
	return h[:8], hex.EncodeToString(h[:8]), nil
}

// decode is an internal method that returns the hash, timestamp, and nonce from the watermark byte slice.
// Optionally returns these values.
func (m *WZeroMark) decode(mark []byte, hash *string, timestamp *time.Time, nonce *string) error {
	if len(mark) != markByteLen {
		return fmt.Errorf("%w: %d", ErrInvalidMarkLength, len(mark))
	}

	msec := int64(binary.BigEndian.Uint64(mark[1:9])) >> 16
	rectimestamp := time.UnixMilli(msec)

	// 1. Generate Ed25519 Private Key
	edKeySeed, err := m.ed25519KeyGen.Generate(rectimestamp)
	if err != nil {
		return fmt.Errorf("failed to generate Ed25519 key seed: %w", err)
	}
	priv := ed25519.NewKeyFromSeed(edKeySeed)
	pub := priv.Public().(ed25519.PublicKey)

	// 2. Verify Signature
	if err := ed25519.VerifyWithOptions(pub, mark[:19], mark[19:], &ed25519.Options{
		Context: context,
	}); err != nil {
		return ErrInvalidSignature
	}
	if mark[0] != version1 {
		return fmt.Errorf("%w: %d", ErrInvalidVersion, mark[0])
	}
	if orgBytes := mark[9:11]; !bytes.Equal(m.orgBytes, orgBytes) {
		return fmt.Errorf("%w: got %x, want %x", ErrInvalidOrgCode, orgBytes, m.orgBytes)
	}

	if timestamp != nil {
		*timestamp = rectimestamp
	}
	if nonce != nil {
		*nonce = hex.EncodeToString(mark[7:9])
	}
	if hash != nil {
		*hash = hex.EncodeToString(mark[11:19])
	}
	return nil
}
