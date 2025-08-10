package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
	case "graph":
		handleGraph()
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
	fmt.Println("  agentctl init [project-name]  Initialize a new agent project (use --type minimal|basic|rag|multi-agent)")
	fmt.Println("  agentctl graph --name <workflow> [--host localhost:8080] [--dir TD|LR] [--conds]")
	fmt.Println("  agentctl version              Show version information")
	fmt.Println("  agentctl help                 Show this help message")
}

func handleInit() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	projectType := fs.String("type", "minimal", "Project type (minimal, basic, rag, multi-agent)")
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

func handleGraph() {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	name := fs.String("name", "", "Workflow name registered in the running server")
	host := fs.String("host", "localhost:8080", "Host of the running server")
	dir := fs.String("dir", "", "Mermaid direction (TD, LR, BT, RL)")
	conds := fs.Bool("conds", false, "Show generic condition indicators on edges")
	fs.Parse(os.Args[2:])

	if *name == "" {
		fmt.Println("--name is required")
		os.Exit(1)
	}
	q := fmt.Sprintf("name=%s", urlQueryEscape(*name))
	if *dir != "" {
		q += fmt.Sprintf("&dir=%s", urlQueryEscape(*dir))
	}
	if *conds {
		q += "&conds=1"
	}
	url := fmt.Sprintf("http://%s/debug/workflows/mermaid?%s", *host, q)
	resp, err := httpGet(url)
	if err != nil {
		fmt.Printf("request error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(resp)
}

// tiny helpers without adding new deps
func urlQueryEscape(s string) string {
	// very small subset sufficient for names we expect
	r := strings.ReplaceAll(s, " ", "+")
	r = strings.ReplaceAll(r, "\n", "")
	r = strings.ReplaceAll(r, "\t", "")
	return r
}

func httpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
