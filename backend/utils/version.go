package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidVersion = errors.New("invalid version format")
	ErrEmptyString    = errors.New("empty string")
)

// VersionType represents different version formats
type VersionType int

const (
	VersionTypeSemantic VersionType = iota // 1.2.3
	VersionTypeDate                        // 2025.03.31
	VersionTypeSnapshot                    // 119886-g52441bd4cd or N-119886-g52441bd4cd-tessus
	VersionTypeUnknown
)

// Version represents a version with flexible format support
type Version struct {
	versionType VersionType
	original    string
	// For semantic versions
	major, minor, patch int
	suffix              int  // 新增：用于存储后缀数字（如 -6 中的 6）
	hasSuffix           bool // 新增：标记是否有后缀
	// For date versions
	date time.Time
	// For snapshot versions
	commitNumber int
	commitHash   string
	prefix       string // "N" for nightly builds
	builder      string // "tessus" for builder identifier
}

// ParseVersion parses a version string into a Version struct
// Supports multiple formats:
// - Semantic: 1.2.3, v1.2.3
// - Date: 2025.03.31, 2025.3.31
// - Snapshot: 119886-g52441bd4cd, N-119886-g52441bd4cd-tessus
func ParseVersion(v string) (*Version, error) {
	if v == "" {
		return nil, ErrEmptyString
	}

	v = strings.TrimPrefix(v, "v")
	version := &Version{original: v}

	// Try semantic version first (x.y.z)
	if semanticVersion, err := parseSemanticVersion(v); err == nil {
		*version = *semanticVersion
		return version, nil
	}

	// Try date version (yyyy.mm.dd)
	if dateVersion, err := parseDateVersion(v); err == nil {
		*version = *dateVersion
		return version, nil
	}

	// Try snapshot version (number-ghash or extended format)
	if snapshotVersion, err := parseSnapshotVersion(v); err == nil {
		*version = *snapshotVersion
		return version, nil
	}

	return nil, ErrInvalidVersion
}

// parseSemanticVersion parses semantic version (x.y.z or x.y.z-suffix)
func parseSemanticVersion(v string) (*Version, error) {
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

	// 处理可能包含后缀的 patch 版本（如 "1-6"）
	patchPart := parts[2]
	var patch, suffix int
	var hasSuffix bool
	
	if dashIndex := strings.Index(patchPart, "-"); dashIndex != -1 {
		// 有后缀的情况
		patch, err = strconv.Atoi(patchPart[:dashIndex])
		if err != nil {
			return nil, ErrInvalidVersion
		}
		
		suffix, err = strconv.Atoi(patchPart[dashIndex+1:])
		if err != nil {
			return nil, ErrInvalidVersion
		}
		hasSuffix = true
	} else {
		// 无后缀的情况
		patch, err = strconv.Atoi(patchPart)
		if err != nil {
			return nil, ErrInvalidVersion
		}
	}

	return &Version{
		versionType: VersionTypeSemantic,
		original:    v,
		major:       major,
		minor:       minor,
		patch:       patch,
		suffix:      suffix,
		hasSuffix:   hasSuffix,
	}, nil
}

// parseDateVersion parses date version (yyyy.mm.dd)
func parseDateVersion(v string) (*Version, error) {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidVersion
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil || year < 2000 || year > 3000 {
		return nil, ErrInvalidVersion
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return nil, ErrInvalidVersion
	}

	day, err := strconv.Atoi(parts[2])
	if err != nil || day < 1 || day > 31 {
		return nil, ErrInvalidVersion
	}

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	return &Version{
		versionType: VersionTypeDate,
		original:    v,
		date:        date,
	}, nil
}

// parseSnapshotVersion parses snapshot version with extended format support
// Supports formats:
// - Simple: 119886-g52441bd4cd
// - Extended: N-119886-g52441bd4cd-tessus
func parseSnapshotVersion(v string) (*Version, error) {
	// Try extended format first: N-number-ghash-builder
	extendedRe := regexp.MustCompile(`^([A-Z]+)-(\d+)-g([a-f0-9]+)-([a-zA-Z0-9]+)$`)
	if matches := extendedRe.FindStringSubmatch(v); len(matches) == 5 {
		commitNumber, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil, ErrInvalidVersion
		}

		return &Version{
			versionType:  VersionTypeSnapshot,
			original:     v,
			commitNumber: commitNumber,
			commitHash:   matches[3],
			prefix:       matches[1], // "N"
			builder:      matches[4], // "tessus"
		}, nil
	}

	// Try simple format: number-ghash
	simpleRe := regexp.MustCompile(`^(\d+)-g([a-f0-9]+)$`)
	if matches := simpleRe.FindStringSubmatch(v); len(matches) == 3 {
		commitNumber, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, ErrInvalidVersion
		}

		return &Version{
			versionType:  VersionTypeSnapshot,
			original:     v,
			commitNumber: commitNumber,
			commitHash:   matches[2],
		}, nil
	}

	return nil, ErrInvalidVersion
}

// Compare compares two versions
// Returns:
//
//	 1 if v > other
//	 0 if v == other
//	-1 if v < other
func (v *Version) Compare(other *Version) int {
	// Same type comparison
	if v.versionType == other.versionType {
		switch v.versionType {
		case VersionTypeSemantic:
			return v.compareSemanticVersion(other)
		case VersionTypeDate:
			return v.compareDateVersion(other)
		case VersionTypeSnapshot:
			return v.compareSnapshotVersion(other)
		}
	}

	// Different types: prioritize Semantic > Date > Snapshot
	typeOrder := map[VersionType]int{
		VersionTypeSemantic: 3,
		VersionTypeDate:     2,
		VersionTypeSnapshot: 1,
	}

	return compareInt(typeOrder[v.versionType], typeOrder[other.versionType])
}

func (v *Version) compareSemanticVersion(other *Version) int {
	if v.major != other.major {
		return compareInt(v.major, other.major)
	}
	if v.minor != other.minor {
		return compareInt(v.minor, other.minor)
	}
	if v.patch != other.patch {
		return compareInt(v.patch, other.patch)
	}
	
	// 比较后缀
	if v.hasSuffix && other.hasSuffix {
		// 两个都有后缀，比较后缀数字
		return compareInt(v.suffix, other.suffix)
	} else if v.hasSuffix && !other.hasSuffix {
		// 只有 v 有后缀，v 被认为是预发布版本，小于正式版本
		return -1
	} else if !v.hasSuffix && other.hasSuffix {
		// 只有 other 有后缀，other 被认为是预发布版本，v 大于 other
		return 1
	}
	
	// 两个都没有后缀，版本相等
	return 0
}

func (v *Version) compareDateVersion(other *Version) int {
	if v.date.Before(other.date) {
		return -1
	}
	if v.date.After(other.date) {
		return 1
	}
	return 0
}

func (v *Version) compareSnapshotVersion(other *Version) int {
	// 首先比较提交计数
	if v.commitNumber != other.commitNumber {
		return compareInt(v.commitNumber, other.commitNumber)
	}

	// 如果提交计数相同，比较提交哈希
	if v.commitHash != other.commitHash {
		return strings.Compare(v.commitHash, other.commitHash)
	}

	// 如果核心版本信息相同，则认为版本相等
	// 不管是否有额外的前缀或构建者信息
	return 0
}

// String returns the string representation of the version
func (v *Version) String() string {
	return v.original
}

// GetType returns the version type
func (v *Version) GetType() VersionType {
	return v.versionType
}

// IsSemanticVersion checks if this is a semantic version
func (v *Version) IsSemanticVersion() bool {
	return v.versionType == VersionTypeSemantic
}

// IsDateVersion checks if this is a date version
func (v *Version) IsDateVersion() bool {
	return v.versionType == VersionTypeDate
}

// IsSnapshotVersion checks if this is a snapshot version
func (v *Version) IsSnapshotVersion() bool {
	return v.versionType == VersionTypeSnapshot
}

// GetSemanticParts returns major, minor, patch for semantic versions
func (v *Version) GetSemanticParts() (int, int, int, error) {
	if !v.IsSemanticVersion() {
		return 0, 0, 0, errors.New("not a semantic version")
	}
	return v.major, v.minor, v.patch, nil
}

// GetDate returns the date for date versions
func (v *Version) GetDate() (time.Time, error) {
	if !v.IsDateVersion() {
		return time.Time{}, errors.New("not a date version")
	}
	return v.date, nil
}

// GetSnapshotParts returns commit number and hash for snapshot versions
func (v *Version) GetSnapshotParts() (int, string, error) {
	if !v.IsSnapshotVersion() {
		return 0, "", errors.New("not a snapshot version")
	}
	return v.commitNumber, v.commitHash, nil
}

// GetSnapshotPartsExtended returns all snapshot version components
func (v *Version) GetSnapshotPartsExtended() (int, string, string, string, error) {
	if !v.IsSnapshotVersion() {
		return 0, "", "", "", errors.New("not a snapshot version")
	}
	return v.commitNumber, v.commitHash, v.prefix, v.builder, nil
}

// IsSameCommit 检查两个快照版本是否指向同一个提交
func (v *Version) IsSameCommit(other *Version) bool {
	if !v.IsSnapshotVersion() || !other.IsSnapshotVersion() {
		return false
	}
	return v.commitNumber == other.commitNumber && v.commitHash == other.commitHash
}

// GetCoreVersion 返回核心版本信息（去除前缀和构建者信息）
func (v *Version) GetCoreVersion() string {
	if v.IsSnapshotVersion() {
		return fmt.Sprintf("%d-g%s", v.commitNumber, v.commitHash)
	}
	return v.original
}

// HasBuildInfo 检查快照版本是否包含构建信息
func (v *Version) HasBuildInfo() bool {
	return v.IsSnapshotVersion() && (v.prefix != "" || v.builder != "")
}

// IsNightlyBuild checks if this is a nightly build (has "N" prefix)
func (v *Version) IsNightlyBuild() bool {
	return v.IsSnapshotVersion() && v.prefix == "N"
}

// GetBuilder returns the builder identifier for snapshot versions
func (v *Version) GetBuilder() string {
	return v.builder
}

// GetPrefix returns the prefix for snapshot versions
func (v *Version) GetPrefix() string {
	return v.prefix
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
