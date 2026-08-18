package util

import "testing"

func TestCompareVersionsSupportsDateVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		newer   bool
	}{
		{name: "later date", current: "v26081001", latest: "v260817", newer: true},
		{name: "earlier date", current: "260817", latest: "26081001", newer: false},
		{name: "later revision", current: "26081001", latest: "26081002", newer: true},
		{name: "same date", current: "260817", latest: "v260817", newer: false},
		{name: "semantic version", current: "1.0.0", latest: "v1.1.0", newer: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newer, err := CompareVersions(test.current, test.latest)
			if err != nil {
				t.Fatalf("compare versions: %v", err)
			}
			if newer != test.newer {
				t.Fatalf("CompareVersions(%q, %q)=%t, want %t", test.current, test.latest, newer, test.newer)
			}
		})
	}
}

func TestIsValidVersion(t *testing.T) {
	valid_versions := []string{"1.2.3", "v1.2.3-beta.1", "260817", "v26081001"}
	for _, version := range valid_versions {
		if !IsValidVersion(version) {
			t.Errorf("IsValidVersion(%q)=false, want true", version)
		}
	}
	invalid_versions := []string{"", "260832", "1.2", "release"}
	for _, version := range invalid_versions {
		if IsValidVersion(version) {
			t.Errorf("IsValidVersion(%q)=true, want false", version)
		}
	}
}
