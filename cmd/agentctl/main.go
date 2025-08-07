package main

import (
	"flag"
	"fmt"
	"os"
)

const version = "v0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		handleInit()
	case "version":
		handleVersion()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("agentctl - CLI for Go AI Agent framework %s\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  agentctl init [project-name]  Initialize a new agent project")
	fmt.Println("  agentctl version              Show version information")
	fmt.Println("  agentctl help                 Show this help message")
}

func handleInit() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	projectType := fs.String("type", "basic", "Project type (basic, rag, multi-agent)")
	fs.Parse(os.Args[2:])

	projectName := "my-agent"
	if fs.NArg() > 0 {
		projectName = fs.Arg(0)
	}

	fmt.Printf("Initializing new %s agent project: %s\n", *projectType, projectName)
	
	if err := initProject(projectName, *projectType); err != nil {
		fmt.Printf("Error initializing project: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Project %s initialized successfully!\n", projectName)
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Printf("  go mod tidy\n")
	fmt.Printf("  go run main.go\n")
}

func handleVersion() {
	fmt.Printf("agentctl version %s\n", version)
	fmt.Printf("Go AI Agent framework CLI\n")
}