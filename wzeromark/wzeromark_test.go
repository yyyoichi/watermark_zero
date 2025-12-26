package wzeromark

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type keyGenMock struct {
	calls []time.Time
	key   []byte
}

func (k *keyGenMock) Generate(timestamp time.Time) ([]byte, error) {
	k.calls = append(k.calls, timestamp)
	return k.key, nil
}

func TestWZeroMark(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		test := []struct {
			name    string
			org     string
			wantErr error
		}{
			{"valid", "1a2b", nil},
			{"bad org hex", "zzzz", ErrInvalidOrgCode},
			{"org wrong length", "0a0b0c", ErrInvalidOrgCode},
		}
		for _, tt := range test {
			t.Run(tt.name, func(t *testing.T) {
				_, err := New(nil, nil, tt.org)
				if tt.wantErr == nil {
					assert.NoError(t, err)
					return
				}
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "error should wrap expected")
			})
		}
	})

	t.Run("encode", func(t *testing.T) {
		key := make([]byte, 32)
		_, _ = rand.Read(key)
		now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
		m := &WZeroMark{
			hmacKeyGen:    &keyGenMock{key: key},
			ed25519KeyGen: &keyGenMock{key: key},
			orgBytes:      []byte{0x0a, 0x0b},
			now:           func() time.Time { return now },
		}
		test := []struct {
			name      string
			src       string
			mark      []byte
			hash      *string
			timestamp *time.Time
			nonce     *string
			assert    func(t *testing.T, mark []byte, hash *string, timestamp *time.Time, nonce *string)
		}{
			{name: "basic", src: "data",
				assert: func(t *testing.T, mark []byte, hash *string, timestamp *time.Time, nonce *string) {
					assert.Zero(t, mark)
					assert.Nil(t, hash)
					assert.Nil(t, timestamp)
					assert.Nil(t, nonce)
				},
			},
			{name: "mark", src: "data",
				mark: make([]byte, markByteLen),
				assert: func(t *testing.T, mark []byte, hash *string, timestamp *time.Time, nonce *string) {
					assert.Len(t, mark, markByteLen)
					assert.Nil(t, hash)
					assert.Nil(t, timestamp)
					assert.Nil(t, nonce)
				},
			},
			{name: "empty mark", src: "data",
				mark: make([]byte, 0),
				assert: func(t *testing.T, mark []byte, hash *string, timestamp *time.Time, nonce *string) {
					assert.Len(t, mark, 0)
					assert.Nil(t, hash)
					assert.Nil(t, timestamp)
					assert.Nil(t, nonce)
				},
			},
			{name: "hash", src: "data",
				hash: new(string),
				assert: func(t *testing.T, mark []byte, hash *string, timestamp *time.Time, nonce *string) {
					assert.Zero(t, mark)
					assert.NotNil(t, hash)
					assert.Len(t, *hash, 8*2)
					assert.Nil(t, timestamp)
					assert.Nil(t, nonce)
				},
			},
			{name: "timestamp", src: "data",
				timestamp: new(time.Time),
				assert: func(t *testing.T, mark []byte, hash *string, timestamp *time.Time, nonce *string) {
					assert.Zero(t, mark)
					assert.Nil(t, hash)
					assert.NotNil(t, timestamp)
					assert.Equal(t, now.UnixMilli(), timestamp.UnixMilli())
					assert.Nil(t, nonce)
				},
			},
			{name: "nonce", src: "data",
				nonce: new(string),
				assert: func(t *testing.T, mark []byte, hash *string, timestamp *time.Time, nonce *string) {
					assert.Zero(t, mark)
					assert.Nil(t, hash)
					assert.Nil(t, timestamp)
					assert.NotNil(t, nonce)
					assert.Len(t, *nonce, 2*2)
				},
			},
			{name: "full", src: "data",
				mark:      make([]byte, markByteLen),
				hash:      new(string),
				timestamp: new(time.Time),
				nonce:     new(string),
				assert: func(t *testing.T, mark []byte, hash *string, timestamp *time.Time, nonce *string) {
					assert.Len(t, mark, markByteLen)
					assert.NotNil(t, hash)
					assert.Len(t, *hash, 8*2)
					assert.NotNil(t, timestamp)
					assert.Equal(t, now.UnixMilli(), timestamp.UnixMilli())
					assert.NotNil(t, nonce)
					assert.Len(t, *nonce, 2*2)
				},
			},
		}

		for _, tt := range test {
			t.Run(tt.name, func(t *testing.T) {
				err := m.encode(tt.src, tt.mark, tt.hash, tt.timestamp, tt.nonce)
				require.NoError(t, err)
				tt.assert(t, tt.mark, tt.hash, tt.timestamp, tt.nonce)
			})
		}
	})

	t.Run("decode", func(t *testing.T) {
		key := make([]byte, 32)
		_, _ = rand.Read(key)
		now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
		m := &WZeroMark{
			hmacKeyGen:    &keyGenMock{key: key},
			ed25519KeyGen: &keyGenMock{key: key},
			orgBytes:      []byte{0x0a, 0x0b},
			now:           func() time.Time { return now },
		}
		var (
			src           = "data"
			testmark      = make([]byte, markByteLen)
			testhash      string
			testtimestamp time.Time
			testnonce     string
		)
		err := m.encode(src, testmark, &testhash, &testtimestamp, &testnonce)
		require.NoError(t, err)

		test := []struct {
			name      string
			mark      []byte
			hash      *string
			timestamp *time.Time
			nonce     *string
			assert    func(t *testing.T, hash *string, timestamp *time.Time, nonce *string)
		}{
			{name: "basic",
				assert: func(t *testing.T, hash *string, timestamp *time.Time, nonce *string) {
					assert.Nil(t, hash)
					assert.Nil(t, timestamp)
					assert.Nil(t, nonce)
				},
			},
			{name: "hash",
				hash: new(string),
				assert: func(t *testing.T, hash *string, timestamp *time.Time, nonce *string) {
					assert.NotNil(t, hash)
					assert.Equal(t, testhash, *hash)
					assert.Nil(t, timestamp)
					assert.Nil(t, nonce)
				},
			},
			{name: "timestamp",
				timestamp: new(time.Time),
				assert: func(t *testing.T, hash *string, timestamp *time.Time, nonce *string) {
					assert.Nil(t, hash)
					assert.NotNil(t, timestamp)
					assert.Equal(t, testtimestamp.UnixMilli(), timestamp.UnixMilli())
					assert.Nil(t, nonce)
				},
			},
			{name: "nonce",
				nonce: new(string),
				assert: func(t *testing.T, hash *string, timestamp *time.Time, nonce *string) {
					assert.Nil(t, hash)
					assert.Nil(t, timestamp)
					assert.NotNil(t, nonce)
					assert.Equal(t, testnonce, *nonce)
				},
			},
			{name: "full",
				hash:      new(string),
				timestamp: new(time.Time),
				nonce:     new(string),
				assert: func(t *testing.T, hash *string, timestamp *time.Time, nonce *string) {
					assert.NotNil(t, hash)
					assert.Equal(t, testhash, *hash)
					assert.NotNil(t, timestamp)
					assert.Equal(t, testtimestamp.UnixMilli(), timestamp.UnixMilli())
					assert.NotNil(t, nonce)
					assert.Equal(t, testnonce, *nonce)
				},
			},
		}
		for _, tt := range test {
			t.Run(tt.name, func(t *testing.T) {
				err := m.decode(testmark, tt.hash, tt.timestamp, tt.nonce)
				require.NoError(t, err)
				tt.assert(t, tt.hash, tt.timestamp, tt.nonce)
			})
		}

	})

	t.Run("invalid decode", func(t *testing.T) {
		key := make([]byte, 32)
		_, _ = rand.Read(key)
		timestamp := time.Date(2024, 11, 30, 19, 45, 0, 0, time.UTC)
		m := &WZeroMark{
			hmacKeyGen:    newHmacKeygen(key, key),
			ed25519KeyGen: newEd25519Keygen(key, key),
			orgBytes:      []byte{0x11, 0x22},
			now:           func() time.Time { return timestamp },
		}
		test := []struct {
			name   string
			edit   func(t *testing.T, mark *[]byte)
			expErr error
		}{
			{name: "invalid length",
				edit: func(t *testing.T, mark *[]byte) {
					*mark = (*mark)[:markByteLen-1]
				},
				expErr: ErrInvalidMarkLength,
			},
			{name: "invalid version",
				edit: func(t *testing.T, mark *[]byte) {
					(*mark)[0] = 0xff
					// Re sign with invalid signature
					edKeySeed, err := m.ed25519KeyGen.Generate(timestamp)
					require.NoError(t, err)
					priv := ed25519.NewKeyFromSeed(edKeySeed)
					sig, err := priv.Sign(nil, (*mark)[:19], &ed25519.Options{
						Context: context,
					})
					require.NoError(t, err)
					copy((*mark)[19:], sig)
				},
				expErr: ErrInvalidVersion,
			},
			{
				name: "invalid org code",
				edit: func(t *testing.T, mark *[]byte) {
					(*mark)[9] = 0xff
					// Re sign with invalid signature
					edKeySeed, err := m.ed25519KeyGen.Generate(timestamp)
					require.NoError(t, err)
					priv := ed25519.NewKeyFromSeed(edKeySeed)
					sig, err := priv.Sign(nil, (*mark)[:19], &ed25519.Options{
						Context: context,
					})
					require.NoError(t, err)
					copy((*mark)[19:], sig)
				},
				expErr: ErrInvalidOrgCode,
			},
			{
				name: "invalid signature",
				edit: func(t *testing.T, mark *[]byte) {
					// Flip last bit of the last byte
					(*mark)[markByteLen-1] ^= 0x01
				},
				expErr: ErrInvalidSignature,
			},
		}
		for _, tt := range test {
			t.Run(tt.name, func(t *testing.T) {
				mark := make([]byte, markByteLen)
				err := m.encode("data", mark, nil, nil, nil)
				require.NoError(t, err)

				tt.edit(t, &mark)

				var decoded string
				err = m.decode(mark, &decoded, nil, nil)
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expErr)
			})
		}
	})

	t.Run("Verify", func(t *testing.T) {
		type params struct {
			master, solt []byte
			orgCode      string
			src          string
		}
		var defaultParams = params{
			master:  make([]byte, 32),
			solt:    make([]byte, 32),
			orgCode: "1b2c",
			src:     "test-source-string-for-verification",
		}
		_, _ = rand.Read(defaultParams.master)
		_, _ = rand.Read(defaultParams.solt)
		var newMark = func(p params) *WZeroMark {
			m, err := New(p.master, p.solt, p.orgCode)
			require.NoError(t, err)
			return m
		}
		test := []struct {
			name     string
			encoder  params
			decorder params
			assert   func(t *testing.T, ok bool, timestamp time.Time, nonce string, err error)
		}{
			{
				name:     "valid same params",
				encoder:  defaultParams,
				decorder: defaultParams,
				assert: func(t *testing.T, ok bool, timestamp time.Time, nonce string, err error) {
					require.NoError(t, err)
					assert.True(t, ok)
					assert.NotZero(t, timestamp)
					assert.NotEmpty(t, nonce)
				},
			},
			{
				name:    "invalid src",
				encoder: defaultParams,
				decorder: func() params {
					p := defaultParams
					p.src = "different-source-string"
					return p
				}(),
				assert: func(t *testing.T, ok bool, timestamp time.Time, nonce string, err error) {
					require.NoError(t, err)
					assert.False(t, ok)
				},
			},
			{
				name:    "invalid orgCode",
				encoder: defaultParams,
				decorder: func() params {
					p := defaultParams
					p.orgCode = "ffff"
					return p
				}(),
				assert: func(t *testing.T, ok bool, timestamp time.Time, nonce string, err error) {
					require.NoError(t, err)
					assert.False(t, ok)
				},
			},
			{
				name:    "invalid master key",
				encoder: defaultParams,
				decorder: func() params {
					p := defaultParams
					p.master = make([]byte, 32)
					_, _ = rand.Read(p.master)
					return p
				}(),
				assert: func(t *testing.T, ok bool, timestamp time.Time, nonce string, err error) {
					require.NoError(t, err)
					assert.False(t, ok)
				},
			},
			{
				name:    "invalid solt",
				encoder: defaultParams,
				decorder: func() params {
					p := defaultParams
					p.solt = make([]byte, 32)
					_, _ = rand.Read(p.solt)
					return p
				}(),
				assert: func(t *testing.T, ok bool, timestamp time.Time, nonce string, err error) {
					require.NoError(t, err)
					assert.False(t, ok)
				},
			},
		}
		for _, tt := range test {
			t.Run(tt.name, func(t *testing.T) {
				encoder := newMark(tt.encoder)
				mark, err := encoder.Encode(tt.encoder.src)
				require.NoError(t, err)

				decorder := newMark(tt.decorder)
				ok, ts, nonce, err := decorder.Verify(mark, tt.decorder.src)
				tt.assert(t, ok, ts, nonce, err)
			})
		}
	})

	t.Run("Public methods", func(t *testing.T) {
		key := make([]byte, 32)
		_, _ = rand.Read(key)
		m, err := New(key, key, "1a2b")
		require.NoError(t, err)
		m.now = func() time.Time {
			return time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		}

		src := "public-methods-test-string"
		mark, hash, timestamp, nonce, err := m.FullEncode(src)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotZero(t, timestamp)
		assert.NotEmpty(t, nonce)

		decHash, decTimestamp, decNonce, err := m.FullDecode(mark)
		require.NoError(t, err)
		assert.Equal(t, hash, decHash)
		assert.Equal(t, timestamp.UnixMilli(), decTimestamp.UnixMilli())
		assert.Equal(t, nonce, decNonce)

		mark2, err := m.Encode(src)
		require.NoError(t, err)

		hash2, err := m.Decode(mark)
		require.NoError(t, err)
		assert.Equal(t, hash, hash2)
		hash3, err := m.Decode(mark2)
		require.NoError(t, err)
		assert.Equal(t, hash, hash3)

		ok, err := m.EqualHash(hash, src, timestamp)
		require.NoError(t, err)
		assert.True(t, ok)

		ok2, err := m.EqualHash(hash, src, timestamp.Add(time.Hour))
		require.NoError(t, err)
		assert.False(t, ok2)

		ok3, err := m.EqualHash("ffffffffffffffff", src, timestamp)
		require.NoError(t, err)
		assert.False(t, ok3)

		ok4, err := m.EqualHash(hash, "different-src-string", timestamp)
		require.NoError(t, err)
		assert.False(t, ok4)
	})
	t.Run("PublicKeyAt", func(t *testing.T) {
		key := make([]byte, 32)
		_, _ = rand.Read(key)
		m, err := New(key, key, "1a2b")
		require.NoError(t, err)
		timestamp := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

		pub1, err := m.PublicKeyAt(timestamp)
		require.NoError(t, err)
		pub2, err := m.PublicKeyAt(timestamp)
		require.NoError(t, err)
		assert.Equal(t, pub1, pub2, "PublicKeyAt should return the same key for the same timestamp")

		// Check that the returned key is a valid ed25519 public key (32 bytes)
		assert.Len(t, pub1, ed25519.PublicKeySize)

		// Different timestamps should generate different keys
		pub3, err := m.PublicKeyAt(timestamp.Add(time.Hour))
		require.NoError(t, err)
		assert.NotEqual(t, pub1, pub3, "PublicKeyAt should return different keys for different timestamps")
	})
}
