package file

import (
	"reflect"
	"testing"
)

func TestNormalizeAllowedTypes(t *testing.T) {
	want := []string{"png", "jpg"}
	got := normalize_allowed_types([]string{".PNG", "*.jpg", "png", "*", ""})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalize_allowed_types() = %v, want %v", got, want)
	}
}
