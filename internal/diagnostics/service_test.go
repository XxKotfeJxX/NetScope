package diagnostics

import (
	"errors"
	"reflect"
	"testing"
)

func TestNormalizePorts(t *testing.T) {
	t.Parallel()

	got, err := normalizePorts([]int{443, 80, 443})
	if err != nil {
		t.Fatalf("normalizePorts() error = %v", err)
	}
	if want := []int{80, 443}; !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizePorts() = %v, want %v", got, want)
	}
}

func TestNormalizePortsRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	_, err := normalizePorts([]int{0})
	if !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("normalizePorts() error = %v, want ErrInvalidOptions", err)
	}
}
