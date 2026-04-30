package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// String accepts JSON strings, numbers, and booleans while preserving a stable string form.
type String string

func (s *String) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		*s = ""
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = String(str)
		return nil
	}
	var raw any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return err
	}
	*s = String(fmt.Sprint(raw))
	return nil
}

func (s String) MarshalJSON() ([]byte, error) { return json.Marshal(string(s)) }
func (s String) String() string               { return string(s) }

// StringSlice accepts JSON arrays, JSON-encoded string arrays, and comma-separated strings.
type StringSlice []string

func (s *StringSlice) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) || bytes.Equal(data, []byte(`""`)) {
		*s = nil
		return nil
	}
	var values []string
	if err := json.Unmarshal(data, &values); err == nil {
		*s = values
		return nil
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		*s = nil
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &values); err == nil {
			*s = values
			return nil
		}
	}
	parts := strings.Split(raw, ",")
	values = values[:0]
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			values = append(values, part)
		}
	}
	*s = values
	return nil
}

func (s StringSlice) MarshalJSON() ([]byte, error) { return json.Marshal([]string(s)) }

// Float64Slice accepts JSON number arrays and JSON-encoded string arrays.
type Float64Slice []Float64

func (s *Float64Slice) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) || bytes.Equal(data, []byte(`""`)) {
		*s = nil
		return nil
	}
	var values []Float64
	if err := json.Unmarshal(data, &values); err == nil {
		*s = values
		return nil
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		*s = nil
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &values); err == nil {
			*s = values
			return nil
		}
	}
	parts := strings.Split(raw, ",")
	values = values[:0]
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var value Float64
		if err := value.UnmarshalJSON([]byte(part)); err != nil {
			return err
		}
		values = append(values, value)
	}
	*s = values
	return nil
}

func (s Float64Slice) MarshalJSON() ([]byte, error) { return json.Marshal([]Float64(s)) }

// Int accepts either a JSON number or a quoted base-10 integer.
type Int int

// Int64 accepts either a JSON number or a quoted base-10 integer.
type Int64 int64

// Uint64 accepts either a JSON number or a quoted base-10 unsigned integer.
type Uint64 uint64

// Float64 accepts either a JSON number or a quoted floating-point number.
type Float64 float64

func (n *Int) UnmarshalJSON(data []byte) error {
	v, err := parseInt64JSON(data)
	*n = Int(v)
	return err
}

func (n Int) MarshalJSON() ([]byte, error) { return []byte(strconv.FormatInt(int64(n), 10)), nil }

func (n *Int64) UnmarshalJSON(data []byte) error {
	v, err := parseInt64JSON(data)
	*n = Int64(v)
	return err
}

func (n Int64) MarshalJSON() ([]byte, error) { return []byte(strconv.FormatInt(int64(n), 10)), nil }

func (n *Uint64) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) || bytes.Equal(data, []byte(`""`)) {
		*n = 0
		return nil
	}
	var s string
	if data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
	} else {
		s = string(data)
	}
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	*n = Uint64(v)
	return err
}

func (n Uint64) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatUint(uint64(n), 10)), nil
}

func (n *Float64) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) || bytes.Equal(data, []byte(`""`)) {
		*n = 0
		return nil
	}
	var s string
	if data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
	} else {
		s = string(data)
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	*n = Float64(v)
	return err
}

func (n Float64) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(n), 'f', -1, 64)), nil
}

func parseInt64JSON(data []byte) (int64, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) || bytes.Equal(data, []byte(`""`)) {
		return 0, nil
	}
	var s string
	if data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return 0, err
		}
	} else {
		s = string(data)
	}
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

// Time accepts RFC3339 strings, date-only strings, Unix seconds, Unix milliseconds, and numeric strings.
type Time time.Time

func (t *Time) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) || bytes.Equal(data, []byte(`""`)) {
		*t = Time(time.Time{})
		return nil
	}
	var s string
	if data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
	} else {
		s = string(data)
	}
	parsed, err := ParseTime(s)
	if err != nil {
		return err
	}
	*t = Time(parsed)
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time().Format(time.RFC3339Nano))
}

func (t Time) Time() time.Time {
	return time.Time(t)
}

func (t Time) IsZero() bool {
	return t.Time().IsZero()
}

// Date accepts a JSON date string in YYYY-MM-DD format.
type Date time.Time

func (d *Date) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) || bytes.Equal(data, []byte(`""`)) {
		*d = Date(time.Time{})
		return nil
	}
	var s string
	if data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
	} else {
		s = string(data)
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return err
	}
	*d = Date(parsed)
	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.Time().Format("2006-01-02"))
}

func (d Date) Time() time.Time {
	return time.Time(d)
}

func (d Date) IsZero() bool {
	return d.Time().IsZero()
}

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,

	// Polymarket / PostgreSQL style:
	// "2024-01-08 22:29:46.138+00"
	"2006-01-02 15:04:05.999999999-07",
	"2006-01-02 15:04:05-07",

	// Optional variants, useful if API returns +0000 or +00:00.
	"2006-01-02 15:04:05.999999999-0700",
	"2006-01-02 15:04:05-0700",
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05-07:00",

	// No-timezone fallbacks.
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// ParseTime parses RFC3339, date-only, Unix seconds, or Unix milliseconds values.
func ParseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		if n > 1_000_000_000_000 {
			return time.UnixMilli(n).UTC(), nil
		}
		return time.Unix(n, 0).UTC(), nil
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("polymarket: unsupported time %q", value)
}
