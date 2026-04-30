package shared

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFlexibleNumericTypes(t *testing.T) {
	var got struct {
		ID    String  `json:"id"`
		Size  Float64 `json:"size"`
		Count Int     `json:"count"`
		U     Uint64  `json:"u"`
	}
	if err := json.Unmarshal([]byte(`{"id":12345,"size":"12.34","count":"7","u":"99"}`), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID.String() != "12345" {
		t.Fatalf("id = %q", got.ID)
	}
	if float64(got.Size) != 12.34 || int(got.Count) != 7 || uint64(got.U) != 99 {
		t.Fatalf("unexpected numerics: %+v", got)
	}
}

func TestFlexibleSliceTypes(t *testing.T) {
	var got struct {
		Outcomes []string     `json:"outcomes"`
		Prices   Float64Slice `json:"prices"`
		IDs      StringSlice  `json:"ids"`
	}
	if err := json.Unmarshal([]byte(`{
		"outcomes":["Yes","No"],
		"prices":"[\"0.42\",0.58]",
		"ids":"1,2,3"
	}`), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Prices) != 2 || float64(got.Prices[0]) != 0.42 || len(got.IDs) != 3 || got.IDs[2] != "3" {
		t.Fatalf("unexpected slices: %+v", got)
	}
}

func TestFlexibleTimeTypes(t *testing.T) {
	tests := []struct {
		name string
		json string
		want time.Time
	}{
		{
			name: "unix_seconds_string",
			json: `"1713398400"`,
			want: time.Date(2024, 4, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "unix_seconds_number",
			json: `1713398400`,
			want: time.Date(2024, 4, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "unix_millis_string",
			json: `"1713398400000"`,
			want: time.Date(2024, 4, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "unix_millis_number",
			json: `1713398400000`,
			want: time.Date(2024, 4, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "rfc3339",
			json: `"2026-04-25T10:11:12Z"`,
			want: time.Date(2026, 4, 25, 10, 11, 12, 0, time.UTC),
		},
		{
			name: "rfc3339_nano",
			json: `"2026-04-25T10:11:12.123456789Z"`,
			want: time.Date(2026, 4, 25, 10, 11, 12, 123456789, time.UTC),
		},
		{
			name: "postgres_offset_hour",
			json: `"2024-01-08 22:29:46.138+00"`,
			want: time.Date(2024, 1, 8, 22, 29, 46, 138000000, time.UTC),
		},
		{
			name: "postgres_offset_hour_no_fraction",
			json: `"2024-01-08 22:29:46+00"`,
			want: time.Date(2024, 1, 8, 22, 29, 46, 0, time.UTC),
		},
		{
			name: "postgres_offset_colon",
			json: `"2024-01-08 22:29:46.138+00:00"`,
			want: time.Date(2024, 1, 8, 22, 29, 46, 138000000, time.UTC),
		},
		{
			name: "postgres_offset_compact",
			json: `"2024-01-08 22:29:46.138+0000"`,
			want: time.Date(2024, 1, 8, 22, 29, 46, 138000000, time.UTC),
		},
		{
			name: "datetime_without_timezone",
			json: `"2026-04-25 10:11:12"`,
			want: time.Date(2026, 4, 25, 10, 11, 12, 0, time.UTC),
		},
		{
			name: "datetime_without_timezone_fraction",
			json: `"2026-04-25 10:11:12.123"`,
			want: time.Date(2026, 4, 25, 10, 11, 12, 123000000, time.UTC),
		},
		{
			name: "iso_without_timezone",
			json: `"2026-04-25T10:11:12"`,
			want: time.Date(2026, 4, 25, 10, 11, 12, 0, time.UTC),
		},
		{
			name: "iso_without_timezone_fraction",
			json: `"2026-04-25T10:11:12.123"`,
			want: time.Date(2026, 4, 25, 10, 11, 12, 123000000, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Time
			if err := json.Unmarshal([]byte(tt.json), &got); err != nil {
				t.Fatalf("UnmarshalJSON(%s) error = %v", tt.json, err)
			}

			if !got.Time().Equal(tt.want) {
				t.Fatalf("time = %s, want %s", got.Time().Format(time.RFC3339Nano), tt.want.Format(time.RFC3339Nano))
			}
		})
	}
}

func TestFlexibleTimeTypesNullAndEmpty(t *testing.T) {
	tests := []string{
		`null`,
		`""`,
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			var got Time
			if err := json.Unmarshal([]byte(input), &got); err != nil {
				t.Fatalf("UnmarshalJSON(%s) error = %v", input, err)
			}
			if !got.IsZero() {
				t.Fatalf("time = %s, want zero", got.Time().Format(time.RFC3339Nano))
			}
		})
	}
}

func TestFlexibleDateType(t *testing.T) {
	var got Date
	if err := json.Unmarshal([]byte(`"2026-04-25"`), &got); err != nil {
		t.Fatal(err)
	}

	want := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	if !got.Time().Equal(want) {
		t.Fatalf("date = %s, want %s", got.Time().Format("2006-01-02"), want.Format("2006-01-02"))
	}
}
