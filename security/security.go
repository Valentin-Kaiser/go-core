// Package security provides cryptographic utilities and PGP and AES encryption/decryption functionality.
//
// It offers functions for generating cryptographically secure random bytes and passwords,
// hashing data using SHA256, MD5, and xxHash algorithms, and deriving keys from strings.
//
// The package also includes the PGPCipher struct, which leverages the gopenpgp library
// to perform PGP encryption and decryption with both public/private key pairs and passphrase-based methods.
//
// Key features:
//   - Secure random byte generation and base64 encoding
//   - Password generation with customizable character sets
//   - Hashing functions: SHA256, MD5, and xxHash
//   - Reading and saving passphrases securely to files
//   - PGP encryption/decryption with support for:
//   - Public key encryption
//   - Private key decryption
//   - Passphrase-based encryption and decryption
//   - Key pair generation with user identity and high security settings
//   - Convenient methods to load keys from files or strings
//
// This package is designed for use cases requiring cryptographically secure operations
// and OpenPGP-compatible message encryption in Go applications.
package security

import (
	"crypto/md5" //nolint:gosec
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/Valentin-Kaiser/go-core/apperror"
	"github.com/cespare/xxhash/v2"
)

// GetRandomBytes Generates random Bytes using crypto/rand which is significantly
// slower than math/rand but cryptographically secure. This Should be used whenever
// cryptographic keys should be generated.
func GetRandomBytes(size uint) ([]byte, error) {
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, apperror.Wrap(err)
	}
	return bytes, nil
}

// SHA256 returns the SHA256 hash of the input as a hex string
func SHA256(inp []byte) string {
	hash := sha256.Sum256(inp)
	return hex.EncodeToString(hash[:])
}

// XX returns the xxhash hash of the input as a uint64
func XX(inp []byte) uint64 {
	return xxhash.Sum64(inp)
}

// Md5 returns the MD5 hash of the input as a hex string
func Md5(v any) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%v", v))) //nolint:gosec
	return hex.EncodeToString(hash[:])
}

// StringToKey32Bytes produces a 32byte slice from the input string using SHA256
func StringToKey32Bytes(inp string) []byte {
	hash := sha256.Sum256([]byte(inp))
	return hash[:]
}

// StringToKey32 produces a 32byte slice from the input string using SHA256
func StringToKey32(inp string) string {
	hash := sha256.Sum256([]byte(inp))
	return hex.EncodeToString(hash[:])[:32]
}

// GetRandomBytesBase64 Returns the Base64 Encoded Equivalent of calling GetRandomBytes.
func GetRandomBytesBase64(size uint) (string, error) {
	bytes, err := GetRandomBytes(size)
	if err != nil {
		return "", apperror.Wrap(err)
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// GenerateRandomPassword generates a random password of the specified length
func GenerateRandomPassword(length int) (string, error) {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.-#!+/")
	gen := make([]rune, length)

	for i := range gen {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", apperror.Wrap(err)
		}
		gen[i] = letters[num.Int64()]
	}

	return string(gen), nil
}

// ReadOrSavePassphrase generates a random passphrase of the specified length and saves it if the file does not exist.
func ReadOrSavePassphrase(file string, length int) ([]byte, error) {
	_, err := os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		passphrase, err := GenerateRandomPassword(length)
		if err != nil {
			return nil, apperror.Wrap(err)
		}
		err = os.WriteFile(file, []byte(passphrase), 0600)
		if err != nil {
			return nil, apperror.Wrap(err)
		}
		return []byte(passphrase), nil
	}
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	passphrase, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	if len(passphrase) < length {
		return nil, apperror.NewError("passphrase is too short")
	}

	return passphrase[:length], nil
}
