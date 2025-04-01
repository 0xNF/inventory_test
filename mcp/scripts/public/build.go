package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

func getScriptDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get script directory")
	}
	return filepath.Dir(filename)
}

// Prints a colored message to stdout
func printColored(color, message string) {
	// Only print colors if we're in a terminal
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		fmt.Print(color + message + colorReset)
	} else {
		fmt.Print(message)
	}
}

// Prints a colored error message to stderr
func printError(message string) {
	// Only print colors if we're in a terminal
	if fileInfo, _ := os.Stderr.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		fmt.Fprint(os.Stderr, colorRed+message+colorReset)
	} else {
		fmt.Fprint(os.Stderr, message)
	}
}

func main() {
	// Get current script directory
	scriptDir := getScriptDir()

	// Define default main.go path
	defaultMainPath := filepath.Join(scriptDir, "..", "..", "cmd", "mcpserver", "main.go")

	// Parse command line arguments
	mainPath := flag.String("main", defaultMainPath, "Path to main.go file")
	version := flag.String("version", "1.0.0.0", "Version number for the build")

	// Define default output directory based on OS and architecture
	defaultOutputDir := filepath.Join(scriptDir, "..", "..", "dist",
		fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
	outputDir := flag.String("output", defaultOutputDir, "Output directory for build artifacts")

	flag.Parse()

	// Print build parameters
	printColored(colorGreen, "Build parameters:\n")
	printColored(colorYellow, fmt.Sprintf("  Main file: %s\n", *mainPath))
	printColored(colorYellow, fmt.Sprintf("  Version: %s\n", *version))
	printColored(colorYellow, fmt.Sprintf("  Output directory: %s\n", *outputDir))
	fmt.Println()

	// Check if main.go exists
	if _, err := os.Stat(*mainPath); os.IsNotExist(err) {
		printError(fmt.Sprintf("Error: main.go not found at %s\n", *mainPath))
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		printError(fmt.Sprintf("Error creating output directory: %v\n", err))
		os.Exit(1)
	}

	// Get the base directory of the project to run the build from
	projectDir := filepath.Dir(*mainPath)

	// Determine output binary name
	binaryName := "wtinventorymcpserver"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	outputPath := filepath.Join(*outputDir, binaryName)

	// Build the ldflags for version information
	ldflags := fmt.Sprintf("-X 'main.Version=%s'", *version)

	// Execute go build command
	printColored(colorGreen, "Building project...\n")
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", outputPath, *mainPath)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		printError(fmt.Sprintf("Build failed: %v\n", err))
		os.Exit(1)
	}

	printColored(colorGreen, fmt.Sprintf("Build successful! Binary saved to: %s\n", outputPath))
}
