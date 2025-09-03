package utils

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// uuidRegex is a regular expression for validating UUID format
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// UUIDv7 utilities for optimal performance and cursor-based pagination

// GenerateUUIDv7 generates a new UUIDv7
func GenerateUUIDv7() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

// GenerateUUIDv7WithTime generates UUIDv7 with specific time according to RFC 9562
// The first 48 bits contain the timestamp in milliseconds since Unix epoch
// The next 16 bits contain version (7) and variant
// The remaining 74 bits contain random data
func GenerateUUIDv7WithTime(t time.Time) uuid.UUID {
	var id uuid.UUID

	// Convert time to milliseconds since Unix epoch
	timestamp := uint64(t.UnixMilli())

	// Set the first 6 bytes (48 bits) to the timestamp
	// We need to handle the 48-bit timestamp properly
	// The timestamp goes into the first 6 bytes, with proper byte ordering
	id[0] = byte(timestamp >> 40)
	id[1] = byte(timestamp >> 32)
	id[2] = byte(timestamp >> 24)
	id[3] = byte(timestamp >> 16)
	id[4] = byte(timestamp >> 8)
	id[5] = byte(timestamp)

	// Set version (7) and variant bits in the 7th byte
	// Version 7: 0b0111xxxx, Variant: 0b10xxxxxx
	id[6] = 0x70 | (id[6] & 0x0F) // Set version 7
	id[8] = 0x80 | (id[8] & 0x3F) // Set variant

	// Fill the remaining bytes with random data
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)

	// Copy random data to the remaining bytes, preserving version and variant
	id[7] = randomBytes[0]
	id[9] = randomBytes[1]
	id[10] = randomBytes[2]
	id[11] = randomBytes[3]
	id[12] = randomBytes[4]
	id[13] = randomBytes[5]
	id[14] = randomBytes[6]
	id[15] = randomBytes[7]

	return id
}

// ParseUUIDv7Time extracts timestamp from UUIDv7 according to RFC 9562
// Returns the timestamp embedded in the first 48 bits
func ParseUUIDv7Time(id uuid.UUID) (time.Time, error) {
	if !IsUUIDv7(id) {
		return time.Time{}, fmt.Errorf("not a UUIDv7: %s", id.String())
	}

	// Extract the first 6 bytes (48 bits) which contain the timestamp
	// Reconstruct the 48-bit timestamp from the first 6 bytes
	timestamp := uint64(id[0])<<40 | uint64(id[1])<<32 | uint64(id[2])<<24 |
		uint64(id[3])<<16 | uint64(id[4])<<8 | uint64(id[5])

	// Convert milliseconds to time.Time
	return time.UnixMilli(int64(timestamp)), nil
}

// GenerateUUIDv7WithOffset generates UUIDv7 with time offset from current time
func GenerateUUIDv7WithOffset(offset time.Duration) uuid.UUID {
	return GenerateUUIDv7WithTime(time.Now().Add(offset))
}

// GenerateUUIDv7WithUnixTimestamp generates UUIDv7 from Unix timestamp in milliseconds
func GenerateUUIDv7WithUnixTimestamp(timestamp int64) uuid.UUID {
	return GenerateUUIDv7WithTime(time.UnixMilli(timestamp))
}

// IsUUIDv7 checks if a UUID is version 7
func IsUUIDv7(id uuid.UUID) bool {
	return id.Version() == 7
}

// IsValidUUIDv7 checks if a string is a valid UUIDv7
func IsValidUUIDv7(s string) bool {
	// Check for empty string
	if s == "" {
		return false
	}

	// Check for leading/trailing whitespace
	if strings.TrimSpace(s) != s {
		return false
	}

	// Check for valid length (32 for compact format, 36 for standard format)
	if len(s) != 32 && len(s) != 36 {
		return false
	}

	// For 36-character format, check for exact format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	// where x is a hexadecimal digit (case insensitive)
	if len(s) == 36 && !uuidRegex.MatchString(strings.ToLower(s)) {
		return false
	}

	// For 32-character format, check that all characters are hexadecimal digits
	if len(s) == 32 {
		for _, c := range s {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}

	// Parse the UUID
	id, err := uuid.Parse(s)
	if err != nil {
		return false
	}

	return IsUUIDv7(id)
}

// ParseUUIDv7 parses a string into UUIDv7, returns error if not valid UUIDv7
func ParseUUIDv7(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID format: %w", err)
	}
	if !IsUUIDv7(id) {
		return uuid.Nil, fmt.Errorf("not a UUIDv7: %s", s)
	}
	return id, nil
}

// MustParseUUIDv7 parses a string into UUIDv7, panics if not valid
func MustParseUUIDv7(s string) uuid.UUID {
	id, err := ParseUUIDv7(s)
	if err != nil {
		panic(err)
	}
	return id
}

// UUIDv7FromBytes creates UUIDv7 from byte slice
func UUIDv7FromBytes(b []byte) (uuid.UUID, error) {
	if len(b) != 16 {
		return uuid.Nil, fmt.Errorf("invalid UUID byte length: %d", len(b))
	}
	var id uuid.UUID
	copy(id[:], b)
	if !IsUUIDv7(id) {
		return uuid.Nil, fmt.Errorf("not a UUIDv7")
	}
	return id, nil
}

// UUIDv7ToBytes converts UUIDv7 to byte slice
func UUIDv7ToBytes(id uuid.UUID) ([]byte, error) {
	if !IsUUIDv7(id) {
		return nil, fmt.Errorf("not a UUIDv7")
	}
	return id[:], nil
}

// UUIDv7FromTimeRange generates UUIDv7 within a time range
func UUIDv7FromTimeRange(start, end time.Time) (uuid.UUID, error) {
	if start.After(end) {
		return uuid.Nil, fmt.Errorf("start time must be before end time")
	}

	// Generate UUIDv7 with a time within the range
	midTime := start.Add(end.Sub(start) / 2)
	return GenerateUUIDv7WithTime(midTime), nil
}

// UUIDv7TimeRange returns the time range for UUIDv7 generation
func UUIDv7TimeRange() (time.Time, time.Time) {
	// UUIDv7 supports timestamps from 1970 to 2106
	start := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2106, 2, 7, 6, 28, 15, 0, time.UTC)
	return start, end
}

// IsUUIDv7TimeValid checks if the time embedded in UUIDv7 is within valid range
func IsUUIDv7TimeValid(id uuid.UUID) bool {
	if !IsUUIDv7(id) {
		return false
	}

	t, err := ParseUUIDv7Time(id)
	if err != nil {
		return false
	}

	start, end := UUIDv7TimeRange()
	return !t.Before(start) && !t.After(end)
}

// UUIDv7TimeDifference calculates the time difference between two UUIDv7s
func UUIDv7TimeDifference(id1, id2 uuid.UUID) (time.Duration, error) {
	if !IsUUIDv7(id1) || !IsUUIDv7(id2) {
		return 0, fmt.Errorf("both UUIDs must be UUIDv7")
	}

	t1, err := ParseUUIDv7Time(id1)
	if err != nil {
		return 0, fmt.Errorf("failed to parse time from first UUID: %w", err)
	}

	t2, err := ParseUUIDv7Time(id2)
	if err != nil {
		return 0, fmt.Errorf("failed to parse time from second UUID: %w", err)
	}

	return t2.Sub(t1), nil
}

// UUIDv7Sortable returns true if UUIDv7s are sortable by time
func UUIDv7Sortable(id1, id2 uuid.UUID) bool {
	if !IsUUIDv7(id1) || !IsUUIDv7(id2) {
		return false
	}

	t1, err1 := ParseUUIDv7Time(id1)
	t2, err2 := ParseUUIDv7Time(id2)

	if err1 != nil || err2 != nil {
		return false
	}

	return t1.Before(t2) == (id1.String() < id2.String())
}

// UUIDv7Cursor creates a cursor string from UUIDv7 for pagination
func UUIDv7Cursor(id uuid.UUID) string {
	if !IsUUIDv7(id) {
		return ""
	}
	return id.String()
}

// UUIDv7FromCursor creates UUIDv7 from cursor string
func UUIDv7FromCursor(cursor string) (uuid.UUID, error) {
	if cursor == "" {
		return uuid.Nil, nil
	}
	return ParseUUIDv7(cursor)
}

// UUIDv7Batch generates a batch of UUIDv7s
func UUIDv7Batch(count int) ([]uuid.UUID, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}
	if count > 10000 {
		return nil, fmt.Errorf("count too large: %d", count)
	}

	ids := make([]uuid.UUID, count)
	for i := 0; i < count; i++ {
		ids[i] = GenerateUUIDv7()
	}

	return ids, nil
}

// UUIDv7BatchWithTime generates a batch of UUIDv7s with specific time
func UUIDv7BatchWithTime(t time.Time, count int) ([]uuid.UUID, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}
	if count > 10000 {
		return nil, fmt.Errorf("count too large: %d", count)
	}

	ids := make([]uuid.UUID, count)
	for i := 0; i < count; i++ {
		// Add small time increments to ensure uniqueness
		increment := time.Duration(i) * time.Microsecond
		ids[i] = GenerateUUIDv7WithTime(t.Add(increment))
	}

	return ids, nil
}

// UUIDv7Validator validates UUIDv7 properties
type UUIDv7Validator struct {
	AllowNil  bool
	CheckTime bool
	TimeRange time.Duration
}

// NewUUIDv7Validator creates a new UUIDv7 validator
func NewUUIDv7Validator() *UUIDv7Validator {
	return &UUIDv7Validator{
		AllowNil:  false,
		CheckTime: true,
		TimeRange: 24 * time.Hour, // Allow 24 hour range
	}
}

// Validate validates a UUIDv7
func (v *UUIDv7Validator) Validate(id uuid.UUID) error {
	if id == uuid.Nil {
		if !v.AllowNil {
			return fmt.Errorf("UUID cannot be nil")
		}
		return nil
	}

	if !IsUUIDv7(id) {
		return fmt.Errorf("not a UUIDv7: %s", id.String())
	}

	if v.CheckTime {
		t, err := ParseUUIDv7Time(id)
		if err != nil {
			return fmt.Errorf("invalid UUIDv7 time: %w", err)
		}

		now := time.Now()
		if t.After(now.Add(v.TimeRange)) || t.Before(now.Add(-v.TimeRange)) {
			return fmt.Errorf("UUIDv7 time out of range: %v", t)
		}
	}

	return nil
}

// ValidateString validates a UUIDv7 string
func (v *UUIDv7Validator) ValidateString(s string) error {
	id, err := ParseUUIDv7(s)
	if err != nil {
		return err
	}
	return v.Validate(id)
}
