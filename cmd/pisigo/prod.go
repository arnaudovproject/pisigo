package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func Prod(args []string) {
	target := "."
	out := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o" || arg == "--output":
			if i+1 >= len(args) {
				fatal("missing value for %s", arg)
			}
			out = args[i+1]
			i++
		case strings.HasPrefix(arg, "-o="):
			out = strings.TrimPrefix(arg, "-o=")
		case strings.HasPrefix(arg, "--output="):
			out = strings.TrimPrefix(arg, "--output=")
		case strings.HasPrefix(arg, "-"):
			fatal("unknown flag: %s", arg)
		default:
			target = arg
		}
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		fatal("resolve path: %v", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		fatal("path not found: %s", abs)
	}
	if !info.IsDir() {
		fatal("path must be a directory: %s", abs)
	}

	if out == "" {
		name := filepath.Base(abs)
		if name == "." || name == string(filepath.Separator) {
			wd, _ := os.Getwd()
			name = filepath.Base(wd)
		}
		if name == "" || name == "." {
			name = "app"
		}
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		out = filepath.Join("bin", name)
	}

	if dir := filepath.Dir(out); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fatal("create output dir: %v", err)
		}
	}

	fmt.Printf("pisigo prod — building %s -> %s\n", abs, out)

	cmd := exec.Command(
		"go",
		"build",
		"-trimpath",
		"-ldflags=-s -w",
		"-o",
		out,
		abs,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Dir = findModuleRoot(abs)

	if err := cmd.Run(); err != nil {
		fatal("build failed: %v", err)
	}

	fmt.Printf("pisigo: production binary ready at %s\n", out)
}

func findModuleRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return start
		}
		dir = parent
	}
}
