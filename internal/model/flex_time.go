package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// FlexTime aceita time.Time (Postgres) ou string (SQLite) no Scan — evita erro "unsupported Scan, storing driver.Value type string into type *time.Time"
type FlexTime struct {
	time.Time
}

// Scan implementa sql.Scanner para aceitar time.Time e string (SQLite retorna datetime como string)
func (t *FlexTime) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case []byte:
		return t.parse(string(v))
	case string:
		return t.parse(v)
	default:
		return fmt.Errorf("FlexTime: tipo não suportado %T", value)
	}
}

func (t *FlexTime) parse(s string) error {
	s = trimSpace(s)
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02",
	}
	for _, f := range formats {
		if parsed, err := time.Parse(f, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("FlexTime: não foi possível parsear %q", s)
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// Value implementa driver.Valuer para INSERT/UPDATE
func (t FlexTime) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time, nil
}
