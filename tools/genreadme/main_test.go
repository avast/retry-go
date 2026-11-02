package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadmeMatchesGenerated(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the repository root")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	generated, err := generate(repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	committed, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatal(err)
	}

	if strings.ReplaceAll(generated, "\r\n", "\n") != strings.ReplaceAll(string(committed), "\r\n", "\n") {
		t.Error("README.md is out of date - run make generate and commit the result")
	}
}
