// Package cxds represents stub for CXO data server.
// The cxds is some kind of client of CX data server.
// In future it will be replaced with real client.
// API of this package close to goal and can be used
package cxds

import (
	"encoding/binary"
	"errors"

	"github.com/skycoin/skycoin/src/cipher"
)

const Version int = 2 // previous is 1

// comon errors
var (
	ErrEmptyValue = errors.New("empty value")

	ErrWrongValueLength = errors.New("wrong value length")

	ErrMissingMetaInfo = errors.New("missing meta information")

	ErrMissingVersion = errors.New("missing version in meta")
	ErrOldVersion     = errors.New("db file of old version")      // cxodbfix
	ErrNewVersion     = errors.New("db file newer then this CXO") // go get
)

//
func getRefsCount(val []byte) (rc uint32) {
	rc = binary.BigEndian.Uint32(val)
	return
}

//
func setRefsCount(val []byte, rc uint32) {
	binary.BigEndian.PutUint32(val, rc)
	return
}

func getHash(val []byte) (key cipher.SHA256) {
	return cipher.SumSHA256(val)
}

// version

func versionBytes() []byte {
	return encodeUint32(uint32(Version))
}

func encodeUint32(u uint32) (ub []byte) {
	ub = make([]byte, 4)
	binary.BigEndian.PutUint32(ub, uint32(Version))
	return
}

func decodeUint32(ub []byte) uint32 {
	return binary.BigEndian.Uint32(ub)
}

// increment slice
func incSlice(b []byte) {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == 0xff {
			b[i] = 0
			continue // increase next byte
		}
		b[i]++
		return
	}
}
