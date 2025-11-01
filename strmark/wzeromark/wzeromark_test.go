package wzeromark

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestWZeroMark contains nested subtests (in one top-level function) to exercise
// New, FullEncode/FullDecode/Decode, Verify, tampering detection, and invalid
// input error cases. Table-driven style is used where helpful.
func TestWZeroMark(t *testing.T) {
	t.Run("New invalid inputs", func(t *testing.T) {
		test := []struct {
			name    string
			seed    []byte
			org     string
			wantErr error
		}{
			{"short seed", make([]byte, 16), "0a0b", ErrInvalidCryptoSeedLength},
			{"bad org hex", make([]byte, 32), "zzzz", ErrInvalidOrgCode},
			{"org wrong length", make([]byte, 32), "0a0b0c", ErrInvalidOrgCode},
		}
		for _, tt := range test {
			t.Run(tt.name, func(t *testing.T) {
				_, err := New(tt.seed, tt.org)
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "error should wrap expected")
			})
		}
	})

	t.Run("private encode/decode happy path", func(t *testing.T) {
		seed := make([]byte, 32)
		for i := range 32 {
			seed[i] = byte(i)
		}
		m, err := New(seed, "0a0b")
		assert.NoError(t, err)
		fixed := time.Unix(1234567890, 42)
		m.now = func() time.Time { return fixed }

		var (
			mark    = make([]bool, MarkLen)
			gotHash string
			gotTs   time.Time
		)
		src := "hello, watermark"
		err = m.encode(src, mark, &gotHash, &gotTs)
		assert.NoError(t, err)
		assert.Len(t, mark, MarkLen)
		// decode via private decode
		var (
			decodedHash string
			decodedTs   time.Time
		)
		err = m.decode(mark, &decodedHash, &decodedTs)
		assert.NoError(t, err)
		assert.Equal(t, gotHash, decodedHash)
		assert.True(t, decodedTs.Equal(fixed))
	})

	t.Run("public Encode/Decode minimal checks", func(t *testing.T) {
		seed := make([]byte, 32)
		for i := range 32 {
			seed[i] = byte(i)
		}
		m, err := New(seed, "0a0b")
		assert.NoError(t, err)

		{
			mark, err := m.Encode("test")
			assert.NoError(t, err)
			assert.Len(t, mark, MarkLen)
		}
		{
			mark, hash, timestamp, err := m.FullEncode("test")
			assert.NoError(t, err)
			assert.Len(t, mark, MarkLen)
			assert.NotEmpty(t, hash)
			assert.NotZero(t, timestamp)
		}
		{
			mark, _ := m.Encode("test")
			hash, err := m.Decode(mark)
			assert.NoError(t, err)
			assert.NotEmpty(t, hash)
		}
		{
			mark, _ := m.Encode("test")
			hash, timestamp, err := m.FullDecode(mark)
			assert.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotZero(t, timestamp)
		}
		{
			mark, hash, timestamp, _ := m.FullEncode("test")
			ok, gotTimestamp, err := m.Verify(mark, hash)
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, timestamp.UnixNano(), gotTimestamp.UnixNano())
		}
		{
			mark, _, _, _ := m.FullEncode("test")
			_, hash, _, _ := m.FullEncode("test2")
			ok, _, err := m.Verify(mark, hash)
			assert.NoError(t, err)
			assert.False(t, ok)
		}
	})

	t.Run("tampering detection and invalid length", func(t *testing.T) {
		seed := make([]byte, 32)
		for i := range 32 {
			seed[i] = byte(i)
		}
		m, err := New(seed, "0a0b")
		assert.NoError(t, err)

		mark, _, _, err := m.FullEncode("data")
		assert.NoError(t, err)
		mark[0] = !mark[0]
		_, err = m.Decode(mark)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidSignature))

		short := make([]bool, MarkLen-1)
		_, err = m.Decode(short)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidMarkLength))
	})
}
