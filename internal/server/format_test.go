package server

import "testing"

func TestCleanDate(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"*23-08-2019", "23-08-2019"},
		{"06-12-2019", "06-12-2019"},
		{"", ""},
		{"*", ""},
		{"**05-12-2019", "*05-12-2019"},
	}

	for _, tt := range tests {
		if got := cleanDate(tt.in); got != tt.want {
			t.Errorf("cleanDate(%q) = %q, ожидалось %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatLocation(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"playa_del_carmen-mexico", "Playa Del Carmen, Mexico"},
		{"georgia-usa", "Georgia, Usa"},
		{"north_carolina-usa", "North Carolina, Usa"},
		{"dunedin-new_zealand", "Dunedin, New Zealand"},
		{"papeete-french_polynesia", "Papeete, French Polynesia"},
		{"losangeles", "Losangeles"},
	}

	for _, tt := range tests {
		if got := formatLocation(tt.in); got != tt.want {
			t.Errorf("formatLocation(%q) = %q, ожидалось %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatLocationDoesNotPanic(t *testing.T) {
	inputs := []string{
		"",
		"-",
		"--",
		"a--b",
		"_",
		"   ",
		"-usa",
		"city-",
		"日本-tokyo",
	}

	for _, in := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("formatLocation(%q) паникует: %v", in, r)
				}
			}()
			formatLocation(in)
		}()
	}
}
