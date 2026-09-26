// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "dev":
		Dev(os.Args[2:])
	case "prod":
		Prod(os.Args[2:])
	case "new":
		NewProject(os.Args[2:])
	case "generate", "gen":
		Generate(os.Args[2:])
	case "migrate":
		MigrateCmd(os.Args[2:])
	case "install":
		Install(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`pisigo — lightweight Go framework CLI

Usage:
  pisigo dev [path]                 run with auto-reload
  pisigo prod [path] [-o out]       production build
  pisigo new <name>                 scaffold new project
  pisigo generate handler <name>    generate handler stub
  pisigo migrate create <name>      create migration file
  pisigo install agents <target>    install AI agent rules (cursor|claude|codex|all)
  pisigo help                       show this help

Examples:
  pisigo new myapp
  pisigo dev .
  pisigo prod . -o bin/app
  pisigo generate handler users
  pisigo migrate create add_users
  pisigo install agents all
  pisigo install agents cursor claude --force`)
}
