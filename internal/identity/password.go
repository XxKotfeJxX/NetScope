package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultPasswordParams() PasswordParams {
	return PasswordParams{
		Memory: 64 * 1024, Iterations: 3, Parallelism: 2,
		SaltLength: 16, KeyLength: 32,
	}
}

type PasswordHasher struct {
	params PasswordParams
}

func NewPasswordHasher(params PasswordParams) PasswordHasher {
	return PasswordHasher{params: params}
}

func (h PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.params.Iterations,
		h.params.Memory,
		h.params.Parallelism,
		h.params.KeyLength,
	)
	encoding := base64.RawStdEncoding
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.params.Memory,
		h.params.Iterations,
		h.params.Parallelism,
		encoding.EncodeToString(salt),
		encoding.EncodeToString(hash),
	), nil
}

func (h PasswordHasher) Verify(encoded string, password string) (bool, error) {
	params, salt, expected, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		uint32(len(expected)),
	)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func parsePasswordHash(encoded string) (PasswordParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return PasswordParams{}, nil, nil, errors.New("invalid password hash")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil ||
		version != argon2.Version {
		return PasswordParams{}, nil, nil, errors.New("unsupported password hash version")
	}
	var params PasswordParams
	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&params.Memory,
		&params.Iterations,
		&params.Parallelism,
	); err != nil {
		return PasswordParams{}, nil, nil, errors.New("invalid password hash parameters")
	}
	if params.Memory < 8*1024 || params.Memory > 256*1024 ||
		params.Iterations < 1 || params.Iterations > 10 ||
		params.Parallelism < 1 || params.Parallelism > 8 {
		return PasswordParams{}, nil, nil, errors.New("unsafe password hash parameters")
	}
	encoding := base64.RawStdEncoding
	salt, err := encoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 || len(salt) > 64 {
		return PasswordParams{}, nil, nil, errors.New("invalid password hash salt")
	}
	expected, err := encoding.DecodeString(parts[5])
	if err != nil || len(expected) < 16 || len(expected) > 64 {
		return PasswordParams{}, nil, nil, errors.New("invalid password hash value")
	}
	return params, salt, expected, nil
}
