// Copyright (c) 2016 Leonard Hecker
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package argon2

import (
	"bytes"
	"encoding/base64"
	"strconv"
)

// A helper for Decode(). Every operation below increases the off(set).
type parser struct {
	buf []byte
	off int
}

// Ensures that the next len(b) bytes match b
func (p *parser) check(b []byte) int {
	l := len(b)
	i := p.off
	j := i + l

	if j <= len(p.buf) {
		p.off = j
		return bytes.Compare(b, p.buf[i:j])
	}

	return 0
}

// Reads a single byte or returns 0
func (p *parser) readByte() byte {
	if p.off < len(p.buf) {
		i := p.off
		p.off = i + 1
		return p.buf[i]
	}

	return 0
}

// Parses a stringified integer until the next non-numeric character or
// returns 0 in case of an integer overflow.
func (p *parser) parseUint32() uint32 {
	i := p.off
	j := len(p.buf)
	r := uint32(0)

	for ; i < j; i++ {
		d := p.buf[i]

		if '0' <= d && d <= '9' {
			rb := r
			r = r*10 + uint32(d-'0')

			if r < rb {
				return 0 // integer overflow
			}
		} else {
			break
		}
	}

	p.off = i
	return r
}

// Skips 0 or more bytes until delim is found (the skip includes delim).
func (p *parser) skipUntil(delim byte) {
	i := p.off
	idx := bytes.IndexByte(p.buf[i:], delim)

	if idx >= 0 {
		p.off = i + idx + 1
	}
}

// Does the same as skipUntil(delim), but returns a slice of the skipped
// bytes (without delim). Returns nil if the slice length is less than 1.
func (p *parser) readSlice(delim byte) []byte {
	i := p.off
	idx := bytes.IndexByte(p.buf[i:], delim) // TODO: len()

	if idx > 0 {
		j := i + idx
		p.off = j + 1
		return p.buf[i:j]
	}

	return nil
}

// Returns the rest of the parser buffer as a slice, or nil
// if the length of the slice is less than 1.
func (p *parser) readRest() []byte {
	i := p.off
	j := len(p.buf)

	if i < j {
		p.off = j
		return p.buf[i:j]
	}

	return nil
}

// appendBase64 works like a combination of base64.Encode() and append(),
// while preventing additional allocations.
func appendBase64(dst []byte, src []byte, encLen int) []byte {
	l := len(dst)
	c := cap(dst)

	if encLen <= 0 {
		encLen = enc64.EncodedLen(len(src))
	}

	newl := l + encLen

	if newl < l {
		panic("integer overflow")
	}

	if newl > c {
		newc := c * c
		if newl > newc {
			newc = newl
		}

		dst = append(make([]byte, 0, newc), dst...)
		c = newc
	}

	enc64.Encode(dst[l:newl:c], src)
	return dst[:newl]
}

var (
	enc64 = base64.RawStdEncoding

	decChunk1 = []byte("$argon2")
	decChunk2 = []byte("v=")
	decChunk3 = []byte("$m=")
	decChunk4 = []byte(",t=")
	decChunk5 = []byte(",p=")
	encTypD   = []byte("d$v=")
	encTypI   = []byte("i$v=")
	encTypID  = []byte("id$v=")
)

// Encode turns a Raw struct into the official stringified/encoded argon2 representation.
//
// The resulting byte slice can safely be turned into a string.
func (raw *Raw) Encode() []byte {
	c := raw.Config
	saltLen64 := enc64.EncodedLen(len(raw.Salt))
	hashLen64 := enc64.EncodedLen(len(raw.Hash))

	// 36 is a good estimate for the maximal likely static overhead, based on:
	//     7 ("$argon2") + 2 (mode)
	//   + 3 ("$v=") + 2 (version)
	//   + 3 ("$m=") + 7 (memory)
	//   + 3 (",t=") + 2 (time)
	//   + 3 (",p=") + 2 (parallelism)
	//   + 1 ("$") + saltLen64 (salt)
	//   + 1 ("$") + hashLen64 (hash)
	buf := make([]byte, 0, saltLen64+hashLen64+36)
	var encTyp []byte

	switch c.Mode {
	case ModeArgon2d:
		encTyp = encTypD
	case ModeArgon2i:
		encTyp = encTypI
	case ModeArgon2id:
		encTyp = encTypID
	}

	buf = append(buf, decChunk1...)
	buf = append(buf, encTyp...)
	buf = strconv.AppendUint(buf, uint64(c.Version), 10)
	buf = append(buf, decChunk3...)
	buf = strconv.AppendUint(buf, uint64(c.MemoryCost), 10)
	buf = append(buf, decChunk4...)
	buf = strconv.AppendUint(buf, uint64(c.TimeCost), 10)
	buf = append(buf, decChunk5...)
	buf = strconv.AppendUint(buf, uint64(c.Parallelism), 10)
	buf = append(buf, '$')
	buf = appendBase64(buf, raw.Salt, saltLen64)
	buf = append(buf, '$')
	buf = appendBase64(buf, raw.Hash, hashLen64)

	return buf
}

// Decode takes a stringified/encoded argon2 hash and turns it back into a Raw struct.
//
// This decoder ignores "data" attributes as they are likely to be deprecated.
func Decode(encoded []byte) (raw Raw, err error) {
	pa := parser{buf: encoded}
	err = ErrDecodingFail

	if pa.check(decChunk1) != 0 {
		return
	}

	typ1 := pa.readByte()
	typ2 := pa.readByte()
	var mode Mode

	if typ1 == 'i' {
		if typ2 == 'd' {
			r := pa.readByte()

			if r == '$' {
				mode = ModeArgon2id
			} else {
				return
			}
		} else if typ2 == '$' {
			mode = ModeArgon2i
		}
	} else if typ1 == 'd' {
		mode = ModeArgon2d
	} else {
		return
	}

	ok := pa.check(decChunk2)
	v := pa.parseUint32()
	ok |= pa.check(decChunk3)
	m := pa.parseUint32()
	ok |= pa.check(decChunk4)
	t := pa.parseUint32()
	ok |= pa.check(decChunk5)
	p := pa.parseUint32()
	pa.skipUntil('$')
	s := pa.readSlice('$')
	h := pa.readRest()

	if ok != 0 || v == 0 || v > 255 || m == 0 || t == 0 || p == 0 || s == nil || h == nil {
		return
	}

	salt := make([]byte, enc64.DecodedLen(len(s)))
	hash := make([]byte, enc64.DecodedLen(len(h)))
	sl, se := enc64.Decode(salt, s)
	hl, he := enc64.Decode(hash, h)

	if se != nil || he != nil {
		return
	}

	c := &Config{}
	c.HashLength = uint32(hl)
	c.SaltLength = uint32(sl)
	c.MemoryCost = m
	c.TimeCost = t
	c.Parallelism = p
	c.Mode = mode
	c.Version = Version(v)

	raw.Config = c
	raw.Salt = salt[0:sl]
	raw.Hash = hash[0:hl]
	err = nil
	return
}
