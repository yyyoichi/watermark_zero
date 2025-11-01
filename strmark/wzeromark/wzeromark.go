package wzeromark

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/yyyoichi/watermark_zero/internal/bitconv"
	"github.com/yyyoichi/watermark_zero/strmark"
)

const (
	// version 5bit + hash format 3bit
	version1 = 17
	// Length of the watermark, in bits
	MarkLen = 91 * 8
	// Context is the context string used in Ed25519ctx signatures
	Context string = "watermark_zero/v1"
)

var (
	ErrInvalidCryptoSeedLength = errors.New("invalid crypto seed length")
	ErrInvalidMarkLength       = errors.New("invalid mark length")
	ErrInvalidVersion          = errors.New("invalid version")
	ErrInvalidSignature        = errors.New("invalid signature")
	ErrInvalidOrgCode          = errors.New("invalid organization code")
)

var _ strmark.Mark = (*WZeroMark)(nil)

type WZeroMark struct {
	pub      ed25519.PublicKey
	priv     ed25519.PrivateKey
	orgBytes []byte
	now      func() time.Time
}

// New creates a new WZeroMark instance.
// cryptoSeed must be 32 bytes long, used to generate the Ed25519ctx key pair.
// orgCode is a hexadecimal string representing 2 bytes (4 hex characters) identifying the organization.
func New(cryptoSeed []byte, orgCode string) (*WZeroMark, error) {
	if len(cryptoSeed) != ed25519.SeedSize {
		return nil, fmt.Errorf("%w: size: %d", ErrInvalidCryptoSeedLength, len(cryptoSeed))
	}
	priv := ed25519.NewKeyFromSeed(cryptoSeed)
	pub := priv.Public().(ed25519.PublicKey)

	orgBytes, err := hex.DecodeString(orgCode)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidOrgCode, err)
	}
	if len(orgBytes) != 2 {
		return nil, fmt.Errorf("%w: orgCode must decode to 2 bytes (4 hex chars)", ErrInvalidOrgCode)
	}
	return &WZeroMark{
		pub:      pub,
		priv:     priv,
		orgBytes: orgBytes,
		now:      time.Now,
	}, nil
}

// Encode encodes the input string into a slice of booleans representing bits.
// The encoded format is as follows:
//
//	1byte version + 8bytes unix nano timestamp + 2bytes orgCode + SHA256(32bytes)/2 + Ed25519ctx(64bytes)
//	= 91bytes
func (m *WZeroMark) Encode(src string) (mark []bool, err error) {
	mark = make([]bool, MarkLen)
	err = m.encode(src, mark, nil, nil)
	return
}

// FullEncode encodes the input string into a slice of booleans representing bits.
// It also returns the hexadecimal string of the embedded hash and the timestamp.
// The encoded format is as follows:
//
//	1byte version + 8bytes unix nano timestamp + 2bytes orgCode + SHA256(32bytes)/2 + Ed25519ctx(64bytes)
//	= 91bytes
func (m *WZeroMark) FullEncode(src string) (mark []bool, hash string, timestamp time.Time, err error) {
	mark = make([]bool, MarkLen)
	err = m.encode(src, mark, &hash, &timestamp)
	return
}

// Decode decodes the input slice of booleans back into the original string.
// It returns the hexadecimal string of the embedded hash.
func (m *WZeroMark) Decode(mark []bool) (hash string, err error) {
	err = m.decode(mark, &hash, nil)
	return
}

// FellDecode decodes the input slice of booleans back into the original string and timestamp.
// It returns the hexadecimal string of the embedded hash and the timestamp.
func (m *WZeroMark) FullDecode(mark []bool) (hash string, timestamp time.Time, err error) {
	err = m.decode(mark, &hash, &timestamp)
	return
}

// Verify verifies the integrity of the watermark mark against the provided hash string.
// It returns true if the embedded hash matches the provided hash.
func (m *WZeroMark) Verify(mark []bool, hash string) (ok bool, timestamp time.Time, err error) {
	var decoded string
	err = m.decode(mark, &decoded, &timestamp)
	ok = decoded == hash
	return
}

func (m *WZeroMark) encode(src string, mark []bool, hash *string, timestamp *time.Time) error {
	h := sha256.Sum256([]byte(src))

	payload := make([]byte, MarkLen/8)
	payload[0] = version1
	now := m.now()
	binary.BigEndian.PutUint64(payload[1:9], uint64(now.UnixNano()))
	copy(payload[9:11], m.orgBytes)
	copy(payload[11:27], h[:16])

	sig, err := m.priv.Sign(nil, payload[:27], &ed25519.Options{
		Context: Context,
	})
	if err != nil {
		return fmt.Errorf("failed to sign payload: %w", err)
	}
	copy(payload[27:], sig)

	if len(mark) == MarkLen {
		copy(mark, bitconv.BytesToBools(payload))
	}
	if hash != nil {
		*hash = hex.EncodeToString(h[:16])
	}
	if timestamp != nil {
		*timestamp = now
	}
	return nil
}

func (m *WZeroMark) decode(mark []bool, hash *string, timestamp *time.Time) error {
	if len(mark) != MarkLen {
		return fmt.Errorf("%w: %d", ErrInvalidMarkLength, len(mark))
	}
	payload := bitconv.BoolsToBytes(mark)
	if err := ed25519.VerifyWithOptions(m.pub, payload[:27], payload[27:], &ed25519.Options{
		Context: Context,
	}); err != nil {
		return ErrInvalidSignature
	}
	if payload[0] != version1 {
		return fmt.Errorf("%w: %d", ErrInvalidVersion, payload[0])
	}
	if orgBytes := payload[9:11]; !bytes.Equal(m.orgBytes, orgBytes) {
		return fmt.Errorf("%w: got %x, want %x", ErrInvalidOrgCode, orgBytes, m.orgBytes)
	}

	if timestamp != nil {
		tm := int64(binary.BigEndian.Uint64(payload[1:9]))
		*timestamp = time.Unix(0, tm)
	}
	if hash != nil {
		*hash = hex.EncodeToString(payload[11:27])
	}
	return nil
}
