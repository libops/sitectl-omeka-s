package cmd

import (
	"reflect"
	"testing"
)

func TestIncludedPlugins(t *testing.T) {
	t.Parallel()

	want := []string{}
	if got := IncludedPlugins(); !reflect.DeepEqual(got, want) {
		t.Fatalf("IncludedPlugins() = %v, want %v", got, want)
	}
}
