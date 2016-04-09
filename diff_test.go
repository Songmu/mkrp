package main

import (
	"reflect"
	"testing"
)

func TestSliceDiff(t *testing.T) {
	a := []string{"d", "c", "b", "a"}
	b := []string{"a", "c", "e", "f"}
	d := sliceDiff(a, b)

	if !reflect.DeepEqual(d.deleted, []string{"b", "d"}) {
		t.Errorf("deleted: something went wrong")
	}
	if !reflect.DeepEqual(d.added, []string{"e", "f"}) {
		t.Errorf("added: something went wrong")
	}
}
