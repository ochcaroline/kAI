package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed skills/todo/SKILL.md
var skillMD string

func main() {
	home, _ := os.UserHomeDir()
	def := os.Getenv("KAINBAN_FILE")
	if def == "" {
		def = filepath.Join(home, ".kai", "board.json")
	}
	file := flag.String("file", def, "board file (env KAINBAN_FILE); point it into OneDrive to sync")
	addr := flag.String("addr", "127.0.0.1:7777", "web server listen address")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: kai [-file board.json] [-addr host:port] [serve]\n       kai [-addr host:port] skill    print the Claude Code /todo skill (SKILL.md) for that address")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 1 && flag.Arg(0) == "skill" {
		fmt.Print(skillFor(*addr))
		return
	}
	if flag.NArg() > 1 || flag.NArg() == 1 && flag.Arg(0) != "serve" {
		flag.Usage()
		os.Exit(2)
	}

	st, err := NewStore(*file)
	if err == nil {
		err = serve(st, *addr)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "kainban:", err)
		os.Exit(1)
	}
}

// skillFor points the embedded skill at addr; ":7777" listens on all interfaces, so use loopback.
func skillFor(addr string) string {
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	return strings.ReplaceAll(skillMD, "127.0.0.1:7777", addr)
}
