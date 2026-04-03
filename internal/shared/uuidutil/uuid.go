package uuidutil

import (
	"fmt"

	"github.com/gofrs/uuid/v5"
)

func GenerateV7() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("uuidutil: failed to generate UUIDv7: %v", err))
	}
	return id
}

func Parse(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, fmt.Errorf("uuidutil: cannot parse empty string")
	}
	id, err := uuid.FromString(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("uuidutil: invalid UUID format %q: %w", s, err)
	}
	return id, nil
}

func MustParse(s string) uuid.UUID {
	id, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}

func Nil() uuid.UUID {
	return uuid.Nil
}
