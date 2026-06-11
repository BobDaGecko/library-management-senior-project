package db

import (
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SqlStringList []string

func (bstr *SqlStringList) Scan(value any) error {
	if value == nil {
		*bstr = nil
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("unable to convert %v of %T to string", value, value)
	}

	*bstr = strings.Split(str, "\x98") // 0x98 = Start of String
	return nil
}

func (bstr SqlStringList) Value() (driver.Value, error) {
	if len(bstr) == 0 {
		return nil, nil
	}
	return strings.Join(bstr, "\x98"), nil // 0x98 = Start of String
}

type SqlUUID struct {
	uuid.UUID
}

func (buuid *SqlUUID) Scan(value any) error {
	if value == nil {
		*buuid = SqlUUID{}
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("unable to convert %v of %T to string", value, value)
	}
	if str == "" {
		*buuid = SqlUUID{}
		return nil
	}

	parsed, err := FromString(str)
	if err != nil {
		return err
	}
	*buuid = parsed

	return nil
}

func (buuid SqlUUID) Value() (driver.Value, error) {
	return buuid.String(), nil
}

func (buuid SqlUUID) IsEmpty() bool {
	for _, b := range buuid.UUID {
		if b != 0 {
			return false
		}
	}
	return true
}

// Short returns the UUID encoded as a 22-character URL-safe base64 string
// using the alphabet A-Za-z0-9-_ (raw, no padding). This is the "compressed"
// form, much shorter than the 36-character hexadecimal String().
//
// This is the compressor for producing YouTube-style short IDs from SqlUUID.
func (buuid SqlUUID) Short() string {
	if buuid.IsEmpty() {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buuid.UUID[:])
}

// ParseShort decodes a string produced by Short() back into a SqlUUID.
// This is the "decompressor".
//
// It does NOT accept standard hex UUID strings (use uuid.Parse or FromString for those).
// Empty input, invalid length, or invalid characters result in error — a zero
// UUID must never be silently produced, because zero-value conditions are
// dropped by GORM struct queries and match the all-zeros UUID in raw ones.
func ParseShort(s string) (SqlUUID, error) {
	if s == "" {
		return SqlUUID{}, fmt.Errorf("empty short uuid")
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return SqlUUID{}, fmt.Errorf("base64 decode failed for short uuid: %w", err)
	}
	if len(b) != 16 {
		return SqlUUID{}, fmt.Errorf("short uuid decodes to %d bytes (expected 16)", len(b))
	}
	var u uuid.UUID
	copy(u[:], b)
	return SqlUUID{UUID: u}, nil
}

// FromString parses a UUID string that may be either:
//   - the standard 36-char (or 32-char) hexadecimal form (with/without hyphens), or
//   - our Short() compressed 22-char base64 form (with - and _).
//
// This provides a single entrypoint for parsing IDs from URLs, forms, etc.
func FromString(s string) (SqlUUID, error) {
	if u, err := uuid.Parse(s); err == nil {
		return SqlUUID{u}, nil
	}
	return ParseShort(s)
}

func (buuid SqlUUID) Thumb() []string {
	hex := strings.ReplaceAll(buuid.String(), "-", "")
	if len(hex) < 6 {
		hex = "000000"
	} else {
		hex = hex[len(hex)-6:]
	}
	return []string{hex[:3], hex[3:]}
}

var NilTime = time.Time{}

type BaseModel struct {
	gorm.Model
	ID SqlUUID `gorm:"type:text;primaryKey"`
}

func (m *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID.IsEmpty() {
		u := uuid.New()
		m.ID = SqlUUID{u}
	}
	return nil
}
