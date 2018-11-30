// Copyright (c) 2016 Leonard Hecker
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package argon2

import (
	"bytes"
	"reflect"
	"strconv"
	"testing"

	xcryptoArgon2 "golang.org/x/crypto/argon2"

	// Add this dependency to get pprof traces below the cgo barrier.
	//_ "github.com/ianlancetaylor/cgosymbolizer"
)

var (
	config = Config{
		HashLength:  32,
		SaltLength:  16,
		TimeCost:    1,
		MemoryCost:  32 * 1024,
		Parallelism: 1,
		Mode:        ModeArgon2id,
		Version:     Version13,
	}

	password = []byte("password")
	salt     = []byte("saltsalt")

	expectedHash    = []byte{139, 118, 66, 92, 63, 17, 51, 11, 184, 106, 68, 37, 211, 16, 139, 244, 189, 217, 38, 53, 116, 148, 139, 173, 176, 3, 182, 239, 235, 210, 75, 155}
	expectedEncoded = []byte("$argon2id$v=19$m=32768,t=1,p=1$c2FsdHNhbHQ$i3ZCXD8RMwu4akQl0xCL9L3ZJjV0lIutsAO27+vSS5s")
)

func isFalsey(obj interface{}) bool {
	if obj == nil {
		return true
	}

	value := reflect.ValueOf(obj)
	kind := value.Kind()

	return kind >= reflect.Chan && kind <= reflect.Slice && value.IsNil()
}

func mustBeFalsey(t *testing.T, name string, obj interface{}) {
	if !isFalsey(obj) {
		t.Errorf("'%s' should be nil, but is: %v", name, obj)
	}
}

func mustBeTruthy(t *testing.T, name string, obj interface{}) {
	if isFalsey(obj) {
		t.Errorf("'%s' should be non nil, but is: %v", name, obj)
	}
}

func TestHashRaw(t *testing.T) {
	r, err := config.HashRaw(password)
	mustBeTruthy(t, "r.Config", r.Config)
	mustBeTruthy(t, "r.Salt", r.Salt)
	mustBeTruthy(t, "r.Hash", r.Hash)
	mustBeFalsey(t, "err", err)
}

func TestHashEncoded(t *testing.T) {
	enc, err := config.HashEncoded(password)
	mustBeTruthy(t, "encoded", enc)
	mustBeFalsey(t, "err", err)

	if len(enc) == 0 {
		t.Error("len(encoded) must be > 0")
	}

	for _, b := range enc {
		if b == 0 {
			t.Error("encoded must not contain 0x00")
		}
	}
}

func TestHashWithSalt(t *testing.T) {
	r, err := config.Hash(password, salt)
	mustBeTruthy(t, "r.Config", r.Config)
	mustBeTruthy(t, "r.Salt", r.Salt)
	mustBeTruthy(t, "r.Hash", r.Hash)
	mustBeFalsey(t, "err", err)

	if !bytes.Equal(r.Hash, expectedHash) {
		t.Logf("ref: %v", expectedHash)
		t.Logf("act: %v", r.Hash)
		t.Error("hashes do not match")
	}

	enc := r.Encode()
	mustBeTruthy(t, "encoded", enc)

	if !bytes.Equal(enc, expectedEncoded) {
		t.Logf("ref: %s", string(expectedEncoded))
		t.Logf("act: %s", string(enc))
		t.Error("encoded strings do not match")
	}
}

func TestVerifyRaw(t *testing.T) {
	r, err := config.HashRaw(password)
	mustBeTruthy(t, "r.Config", r.Config)
	mustBeTruthy(t, "r.Salt", r.Salt)
	mustBeTruthy(t, "r.Hash", r.Hash)
	mustBeFalsey(t, "err1", err)

	ok, err := r.Verify(password)
	mustBeTruthy(t, "ok", ok)
	mustBeFalsey(t, "err2", err)
}

func TestVerifyEncoded(t *testing.T) {
	encoded, err := config.HashEncoded(password)
	mustBeTruthy(t, "encoded", encoded)
	mustBeFalsey(t, "err1", err)

	ok, err := VerifyEncoded(password, encoded)
	mustBeTruthy(t, "ok", ok)
	mustBeFalsey(t, "err2", err)
}

func TestSecureZeroMemory(t *testing.T) {
	pwd := append([]byte(nil), password...)

	// SecureZeroMemory should erase up to cap(pwd) --> let's test that too
	SecureZeroMemory(pwd[0:0])

	for _, b := range pwd {
		if b != 0 {
			t.Error("pwd must only contain 0x00")
		}
	}
}

func BenchmarkHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		config.Hash(password, salt)
	}
}

func BenchmarkHashXCryptoArgon2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xcryptoArgon2.IDKey(password, salt, config.TimeCost, config.MemoryCost, uint8(config.Parallelism), config.HashLength)
	}
}

func BenchmarkVerify(b *testing.B) {
	r, err := config.Hash(password, salt)
	if err != nil {
		b.Error(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Verify(password)
	}
}

func BenchmarkEncode(b *testing.B) {
	r, err := config.Hash(password, salt)
	if err != nil {
		b.Error(err)
	}

	b.SetBytes(int64(len(expectedEncoded)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Encode()
	}
}

func BenchmarkDecode(b *testing.B) {
	b.SetBytes(int64(len(expectedEncoded)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Decode(expectedEncoded)
	}
}

func BenchmarkSecureZeroMemory(b *testing.B) {
	for _, n := range []int{16, 256, 4096, 65536} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			buf := make([]byte, n)

			b.SetBytes(int64(n))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				SecureZeroMemory(buf)
			}
		})
	}
}
