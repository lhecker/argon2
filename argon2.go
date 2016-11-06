// Copyright (c) 2016 Leonard Hecker
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package argon2 provides fast and easy to use bindings for Argon2:
// A very secure, modern password hashing algorithm - Winner of the
// Password Hashing Competition (PHC).
package argon2

/*
#include <stdint.h>
#include <argon2.h>
#include <core.h>

// This is structurally the same as the Config struct below
typedef struct bindings_argon2_config {
	uint32_t HashLength;
	uint32_t SaltLength;
	uint32_t TimeCost;
	uint32_t MemoryCost;
	uint32_t Parallelism;
	uint32_t Mode;
	uint32_t Version;
} bindings_argon2_config;

// A simplified version of argon2_hash()
int bindings_argon2_hash(const bindings_argon2_config* cfg, void* pwd, const uint32_t pwdlen, void* salt, const uint32_t saltlen, void* hash, const uint32_t hashlen) {
	argon2_context c = {
		.out = hash,
		.outlen = hashlen,
		.pwd = pwd,
		.pwdlen = pwdlen,
		.salt = salt,
		.saltlen = saltlen,
		.t_cost = cfg->TimeCost,
		.m_cost = cfg->MemoryCost,
		.lanes = cfg->Parallelism,
		.threads = cfg->Parallelism,
		.flags = ARGON2_DEFAULT_FLAGS,
		.version = cfg->Version,
	};

	const int rc = argon2_ctx(&c, cfg->Mode);

	if (rc != ARGON2_OK) {
		secure_wipe_memory(hash, hashlen);
	}

	return rc;
}
*/
import "C"
import (
	"crypto/rand"
	"crypto/subtle"
	"unsafe"
)

// Mode exists for type check purposes. See Config.
type Mode uint32

const (
	// ModeArgon2d is faster and uses data-depending memory access,
	// which makes it highly resistant against GPU cracking attacks and
	// suitable for applications with no (!) threats from
	// side-channel timing attacks (eg. cryptocurrencies).
	ModeArgon2d = Mode(C.Argon2_d)

	// ModeArgon2i uses data-independent memory access, which is
	// preferred for password hashing and password-based key derivation
	// (e.g. hard drive encryption), but it's slower as it makes
	// more passes over the memory to protect from TMTO attacks.
	ModeArgon2i = Mode(C.Argon2_i)

	// ModeArgon2id is a hybrid of Argon2i and Argon2d, using a
	// combination of data-depending and data-independent memory accesses,
	// which gives some of Argon2i's resistance to side-channel cache timing
	// attacks and much of Argon2d's resistance to GPU cracking attacks.
	ModeArgon2id = Mode(C.Argon2_id)
)

// String simply maps a ModeArgon{d,i,id} constant to a "Argon{d,i,id}" string
// or returns "unknown" if `m` does not match one of the constants.
func (m Mode) String() string {
	switch m {
	case ModeArgon2d:
		return "Argon2d"
	case ModeArgon2i:
		return "Argon2i"
	case ModeArgon2id:
		return "Argon2id"
	default:
		return "unknown"
	}
}

// Version exists for type check purposes. See Config.
type Version uint32

const (
	// Version10 of the Argon2 algorithm. Deprecated: Use Version13 instead.
	Version10 = Version(C.ARGON2_VERSION_10)

	// Version13 of the Argon2 algorithm. Recommended.
	Version13 = Version(C.ARGON2_VERSION_13)
)

// String simply maps a Version{10,13} constant to a "{10,13}" string
// or returns "unknown" if `v` does not match one of the constants.
func (v Version) String() string {
	switch v {
	case Version10:
		return "10"
	case Version13:
		return "13"
	default:
		return "unknown"
	}
}

// NOTE: Keep `Config` in sync with the C code at the beginning of this file.

// Config contains all configuration parameters for the Argon2 hash function.
//
// You MUST ensure that a Config instance is not changed after creation,
// otherwise you risk race conditions. If you do need to change it during
// runtime use a Mutex and simply create a by-value copy of your shared Config
// instance in the critical section and store it on your local stack.
// That way your critical section is very short, while allowing you to safely
// call all the member methods on your local "immutable" copy.
type Config struct {
	// HashLength specifies the length of the resulting hash in Bytes.
	//
	// Must be > 0.
	HashLength uint32

	// SaltLength specifies the length of the resulting salt in Bytes,
	// if one of the helper methods is used.
	//
	// Must be > 0.
	SaltLength uint32

	// TimeCost specifies the number of iterations of argon2.
	//
	// Must be > 0.
	// If you use ModeArgon2i this should *always* be >= 3 due to TMTO attacks.
	// Additionally if you can afford it you might set it to >= 10.
	TimeCost uint32

	// MemoryCost specifies the amount of memory to use in Kibibytes.
	//
	// Must be > 0.
	MemoryCost uint32

	// Parallelism specifies the amount of threads to use.
	//
	// Must be > 0.
	Parallelism uint32

	// Mode specifies the hashing method used by argon2.
	//
	// If you're writing a server and unsure what to choose,
	// use ModeArgon2i with a TimeCost >= 3.
	Mode Mode

	// Version specifies the argon2 version to be used.
	Version Version
}

// DefaultConfig returns a Config struct suitable for most servers.
//
// These default settings result in around 7ms of computation time while using 4 MiB of memory.
// (Tested on an i7 3770 @ 3.4 GHz & G.Skill F3-14900CL9D @ 1886 MHz).
func DefaultConfig() Config {
	return Config{
		HashLength:  32,
		SaltLength:  16,
		TimeCost:    3,
		MemoryCost:  1 << 12,
		Parallelism: 1,
		Mode:        ModeArgon2i,
		Version:     Version13,
	}
}

// Hash takes a password and optionally a salt and returns an Argon2 hash.
//
// If salt is nil a appropriate salt of Config.SaltLength bytes is generated for you.
// It is recommended to use SecureZeroMemory(pwd) afterwards.
func (c *Config) Hash(pwd []byte, salt []byte) (Raw, error) {
	if pwd == nil {
		return Raw{}, ErrPwdTooShort
	}

	if salt == nil {
		salt = make([]byte, c.SaltLength)
		_, err := rand.Read(salt)

		if err != nil {
			return Raw{}, err
		}
	}

	pwdptr := unsafe.Pointer(nil)
	pwdlen := C.uint32_t(len(pwd))
	saltptr := unsafe.Pointer(nil)
	saltlen := C.uint32_t(len(salt))
	hashptr := unsafe.Pointer(nil)
	hashlen := C.uint32_t(c.HashLength)

	hash := make([]byte, hashlen)

	if pwdlen > 0 {
		pwdptr = unsafe.Pointer(&pwd[0])
	}

	if saltlen > 0 {
		saltptr = unsafe.Pointer(&salt[0])
	}

	if hashlen > 0 {
		hashptr = unsafe.Pointer(&hash[0])
	}

	rc := C.bindings_argon2_hash(
		(*C.struct_bindings_argon2_config)(unsafe.Pointer(c)),
		pwdptr,
		pwdlen,
		saltptr,
		saltlen,
		hashptr,
		hashlen,
	)

	if rc != C.ARGON2_OK {
		return Raw{}, Error(rc)
	}

	return Raw{
		Config: *c,
		Salt:   salt,
		Hash:   hash,
	}, nil
}

// HashRaw is a helper function around Hash()
// which automatically generates a salt for you.
//
// It is recommended to use SecureZeroMemory(pwd) afterwards.
func (c *Config) HashRaw(pwd []byte) (Raw, error) {
	return c.Hash(pwd, nil)
}

// HashEncoded is a helper function around Hash() which automatically
// generates a salt and encodes the result for you.
//
// It is recommended to use SecureZeroMemory(pwd) afterwards.
func (c *Config) HashEncoded(pwd []byte) (encoded []byte, err error) {
	r, err := c.Hash(pwd, nil)
	if err == nil {
		encoded = r.Encode()
	}
	return
}

// Raw wraps a salt and hash pair including the Config with which it was generated.
//
// A Raw struct is generated using Decode() or the Hash*() methods above.
//
// You MUST ensure that a Raw instance is not changed after creation,
// otherwise you risk race conditions. If you do need to change it during
// runtime use a Mutex and simply create a copy of your shared Raw
// instance in the critical section and store it on your local stack.
// That way your critical section is very short, while allowing you to safely
// call all the member methods on your local "immutable" copy.
type Raw struct {
	Config Config
	Salt   []byte
	Hash   []byte
}

// Verify returns true if `pwd` matches the hash in `raw` and otherwise false.
func (raw *Raw) Verify(pwd []byte) (bool, error) {
	r, err := raw.Config.Hash(pwd, raw.Salt)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(r.Hash, raw.Hash) == 1, nil
}

// VerifyEncoded returns true if `pwd` matches the encoded hash `encoded` and otherwise false.
func VerifyEncoded(pwd []byte, encoded []byte) (bool, error) {
	r, err := Decode(encoded)
	if err != nil {
		return false, err
	}
	return r.Verify(pwd)
}

// SecureZeroMemory is a helper method which as securely as possible sets all
// bytes in `b` (up to it's capacity) to `0x00`, erasing it's contents.
//
// Using this method DOES NOT make secrets impossible to recover from memory,
// it's just a good start and generally recommended to use.
//
// This method uses SecureZeroMemory() on Windows, memset_s() if available,
// explicit_bzero() on OpenBSD, or a plain memset() as a fallback.
func SecureZeroMemory(b []byte) {
	c := cap(b)
	if c > 0 {
		b = b[:c:c]
		C.secure_wipe_memory(unsafe.Pointer(&b[0]), C.size_t(c))
	}
}
