package main

import (
	"os"
	"testing"
)

func TestRun(t *testing.T) {
	got, err := run(dirSource{"testdata/repo"}, "1.28", "golang", "go", "master")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/want.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("proposal mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
