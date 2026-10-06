package appinfo

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseVersion(t *testing.T) {
	data, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != Version {
		t.Fatalf("VERSION differs from executable version %s", Version)
	}
}
