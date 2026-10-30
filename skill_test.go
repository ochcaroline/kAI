package main

import (
	"strings"
	"testing"
)

func TestSkillFor(t *testing.T) {
	for addr, want := range map[string]string{
		"127.0.0.1:7777": "http://127.0.0.1:7777/api",
		"localhost:9000": "http://localhost:9000/api",
		":8080":          "http://127.0.0.1:8080/api",
	} {
		got := skillFor(addr)
		if !strings.Contains(got, want) || strings.Contains(got, "7777") != strings.Contains(want, "7777") {
			t.Errorf("skillFor(%q) missing %q or kept default addr", addr, want)
		}
	}
}
