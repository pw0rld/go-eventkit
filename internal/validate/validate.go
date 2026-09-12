// Package validate checks data before it crosses the Objective-C boundary.
package validate

import (
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

func Color(s string) error {
	if s == "" {
		return nil
	}
	if len(s) != 7 || s[0] != '#' {
		return fmt.Errorf("color must be #RRGGBB")
	}
	if _, err := hex.DecodeString(s[1:]); err != nil {
		return fmt.Errorf("invalid color: %w", err)
	}
	return nil
}

func Container(title, name, id, color string) error {
	if err := ID(title); err != nil {
		return err
	}
	if name == "" && id == "" {
		return fmt.Errorf("source name or ID is required")
	}
	if err := Selection(name, id); err != nil {
		return err
	}
	return Color(color)
}

func Text(s string) error {
	if !utf8.ValidString(s) || strings.ContainsRune(s, 0) {
		return fmt.Errorf("text must be valid UTF-8 without NUL")
	}
	return nil
}

func ID(s string) error {
	if err := Text(s); err != nil {
		return err
	}
	if strings.TrimSpace(s) == "" || strings.ContainsAny(s, "\r\n") {
		return fmt.Errorf("identifier or name must be nonempty and single-line")
	}
	return nil
}

func Time(t time.Time) error {
	if t.IsZero() || t.Year() < 1 || t.Year() > 9999 {
		return fmt.Errorf("date must be nonzero with year 1–9999")
	}
	return nil
}

func Range(start, end time.Time) error {
	if err := Time(start); err != nil {
		return err
	}
	if err := Time(end); err != nil {
		return err
	}
	if !end.After(start) {
		return fmt.Errorf("end must be after start")
	}
	return nil
}

func Selection(name, id string) error {
	if name != "" && id != "" {
		return fmt.Errorf("specify a name or an ID, not both")
	}
	if name != "" {
		return ID(name)
	}
	if id != "" {
		return ID(id)
	}
	return nil
}

func Coordinates(lat, lon, radius float64) error {
	for _, v := range []float64{lat, lon, radius} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("coordinates and radius must be finite")
		}
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 || radius < 0 {
		return fmt.Errorf("invalid coordinates or radius")
	}
	return nil
}
