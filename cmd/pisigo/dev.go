// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

func Dev(args []string) {
	target := "."
	if len(args) > 0 {
		target = args[0]
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

	moduleRoot := findModuleRoot(abs)

	tmpDir, err := os.MkdirTemp("", "pisigo-dev-*")
	if err != nil {
		fatal("temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binName := "app"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(tmpDir, binName)

	watchRoot := moduleRoot
	fmt.Printf("pisigo dev — target %s (watching %s)\n", abs, watchRoot)

	var (
		mu      sync.Mutex
		cmd     *exec.Cmd
		running bool
	)

	stop := func() {
		mu.Lock()
		defer mu.Unlock()
		if !running || cmd == nil || cmd.Process == nil {
			return
		}
		_ = killProcess(cmd)
		done := make(chan struct{})
		go func() {
			_, _ = cmd.Process.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
		running = false
		cmd = nil
	}

	start := func() {
		stop()
		fmt.Println("pisigo: building...")
		build := exec.Command("go", "build", "-o", binPath, abs)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		build.Dir = moduleRoot
		if err := build.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "pisigo: build failed: %v\n", err)
			return
		}

		mu.Lock()
		defer mu.Unlock()
		cmd = exec.Command(binPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = abs
		cmd.Env = os.Environ()
		if runtime.GOOS != "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		}
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "pisigo: start failed: %v\n", err)
			return
		}
		running = true
		fmt.Println("pisigo: running")
		go func(c *exec.Cmd) {
			_ = c.Wait()
			mu.Lock()
			if cmd == c {
				running = false
			}
			mu.Unlock()
		}(cmd)
	}

	start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	last := snapshot(watchRoot)
	var pending bool
	var fireAt time.Time

	for {
		select {
		case <-sigCh:
			fmt.Println("\npisigo: shutting down")
			stop()
			return
		case <-ticker.C:
			now := snapshot(watchRoot)
			if !mapsEqual(last, now) {
				last = now
				pending = true
				fireAt = time.Now().Add(400 * time.Millisecond)
			}
			if pending && time.Now().After(fireAt) {
				pending = false
				fmt.Println("pisigo: reload")
				start()
			}
		}
	}
}

func snapshot(root string) map[string]time.Time {
	out := make(map[string]time.Time)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		out[path] = info.ModTime()
		return nil
	})
	return out
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "vendor", "node_modules", "bin", "tmp", "dist":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func mapsEqual(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok || !v.Equal(bv) {
			return false
		}
	}
	return true
}

func killProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		return cmd.Process.Kill()
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
		return nil
	}
	return cmd.Process.Signal(syscall.SIGTERM)
}
