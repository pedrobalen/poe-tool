// Package id generates short, URL-safe, collision-resistant identifiers.
//
// A local generator is preferred over a UUID dependency: the IDs never leave
// the machine, so their only requirement is uniqueness within the local
// SQLite database ("a little copying is better than a little dependency").
package id

import (
	"crypto/rand"
	"encoding/base32"
)

// encoding is base32 without padding, lowercased at call sites, producing IDs
// that are safe in URLs, filenames, and SQLite text columns alike.
var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// New returns a random 128-bit identifier encoded as 26 base32 characters.
func New() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// crypto/rand only fails when the OS entropy source is unavailable,
		// which is unrecoverable for an application that needs stable IDs.
		panic("id: reading random bytes: " + err.Error())
	}

	return encoding.EncodeToString(buf[:])
}
