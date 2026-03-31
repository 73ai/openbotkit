package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/73ai/openbotkit/internal/browser"
	"github.com/73ai/openbotkit/source/twitter/client"
)

func main() {
	outPath := client.DefaultEndpointsPath()
	if len(os.Args) > 1 {
		outPath = os.Args[1]
	}

	fmt.Println("Fetching X endpoint query IDs...")

	httpClient := browser.NewClient()
	endpoints, err := extractEndpoints(httpClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(endpoints) == 0 {
		fmt.Fprintln(os.Stderr, "No endpoints found. Using defaults.")
		endpoints = make(map[string]client.Endpoint)
		for k, v := range client.DefaultEndpoints() {
			endpoints[k] = v
		}
	}

	data, err := endpointsToYAML(endpoints)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling YAML: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %d endpoints to %s\n", len(endpoints), outPath)
	for name, ep := range endpoints {
		fmt.Printf("  %s: %s (%s)\n", name, ep.QueryID, ep.Method)
	}
}
