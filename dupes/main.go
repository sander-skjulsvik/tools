package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	producerConsumer "github.com/sander-skjulsvik/tools/dupes/lib/producerConsumer"
	singleThread "github.com/sander-skjulsvik/tools/dupes/lib/singleThread"
)

func main() {

	var (
		method string
		path   string
	)

	flag.StringVar(&method, "m", "", "Method (single or producerConsumer)")
	flag.StringVar(&method, "method", "", "Method (single or producerConsumer)")
	flag.StringVar(&path, "path", "", "File path")
	flag.StringVar(&path, "p", "", "File path")
	flag.StringVar(&path, "", "", "File path")

	// Parse the command-line arguments
	flag.Parse()

	// Check if the method flag is provided
	if method == "" {
		method = "single"
	}

	// Check if the method is one of the allowed values
	if method != "single" && method != "producerConsumer" {
		fmt.Println("Invalid method. Allowed values are 'single' and 'producerConsumer'.")
		os.Exit(1)
	}

	// Check if the path flag is provided
	if path == "" {
		fmt.Println("Path is required. Please provide a path using -path, -p or flag or without a flag.")
		os.Exit(1)
	}
	// At this point, you have valid values for method and path
	fmt.Printf("Method: %s\n", method)
	fmt.Printf("Path: %s\n", path)

	switch {
	case method == "single":
		singleThread.Run(path)
	case method == "producerConsumer":
		producerConsumer.Run(path)
}

func Run(path, method string) {
	// Use a switch statement to handle different cases
	switch method {
	case "single":
		singlethread.Run(path)
	case "producerConsumer":
		log.Fatal("producerConsumer not implemented yet")
		// producerConsumer.Run(path)
	default:
		fmt.Println("Invalid method. Allowed values are 'single' and 'producerConsumer'.")
		os.Exit(1)
	}
}
