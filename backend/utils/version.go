package utils

import (
	"errors"
	"strconv"
	"strings"
)

var (
	ErrInvalidVersion = errors.New("invalid version format")
	ErrEmptyString    = errors.New("empty string")
)

// Version represents a semantic version
type Version struct {
	major, minor, patch int
}

// ParseVersion parses a version string into a Version struct
func ParseVersion(v string) (*Version, error) {
	if v == "" {
		return nil, ErrEmptyString
	}
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidVersion
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, ErrInvalidVersion
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, ErrInvalidVersion
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, ErrInvalidVersion
	}

	return &Version{
		major: major,
		minor: minor,
		patch: patch,
	}, nil
}

// Compare compares two versions
// Returns:
//
//	 1 if v > other
//	 0 if v == other
//	-1 if v < other
func (v *Version) Compare(other *Version) int {
	if v.major != other.major {
		return compareInt(v.major, other.major)
	}
	if v.minor != other.minor {
		return compareInt(v.minor, other.minor)
	}
	return compareInt(v.patch, other.patch)
}

// String returns the string representation of the version
func (v *Version) String() string {
	return strconv.Itoa(v.major) + "." + strconv.Itoa(v.minor) + "." + strconv.Itoa(v.patch)
}

// compareInt is a helper function to compare two integers
func compareInt(a, b int) int {
	if a > b {
		return 1
	}
	if a < b {
		return -1
	}
	return 0
}
