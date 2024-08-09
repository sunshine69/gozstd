package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
)

var (
	version   string // Will hold the version number
	buildTime string // Will hold the build time
)

func printVersionBuildInfo() {
	fmt.Printf("Version: %s\nBuild time: %s\n", version, buildTime)
}

func compress(input io.Reader, output io.Writer, compressionLevel, numThreads int) error {
	options := []zstd.EOption{zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressionLevel))}

	// Set the number of threads if specified
	if numThreads > 0 {
		options = append(options, zstd.WithEncoderConcurrency(numThreads))
	}

	encoder, err := zstd.NewWriter(output, options...)
	if err != nil {
		return fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	defer encoder.Close()

	_, err = io.Copy(encoder, input)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	return nil
}

func decompress(input io.Reader, output io.Writer) error {
	decoder, err := zstd.NewReader(input)
	if err != nil {
		return fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer decoder.Close()

	_, err = io.Copy(output, decoder)
	if err != nil {
		return fmt.Errorf("failed to decompress data: %w", err)
	}

	return nil
}

func main() {
	// Define flags
	compressMode := flag.Bool("d", false, "Decompress instead of compress")
	outputToStdout := flag.Bool("c", false, "Write output to stdout")
	outputFile := flag.String("o", "", "Output file (default: stdout)")
	compressionLevel := flag.Int("l", 3, "Set compression level (1-19, default: 3. Good tradoff is 9)")
	numThreads := flag.Int("T", 0, "Number of threads (default: library default)")

	flag.Usage = func() {
		printVersionBuildInfo()
		flag.PrintDefaults()
	}
	// Parse flags
	flag.Parse()

	// Determine input source
	// fmt.Fprintf(os.Stderr, "[DEBUG] %v\n", flag.Args())
	// fmt.Fprintf(os.Stderr, "[DEBUG] compressMode %v outputToStdout %v outputFile %v compressionLevel %v numThreads %v\n", *compressMode, *outputToStdout, *outputFile, *compressionLevel, *numThreads)

	var input io.Reader = os.Stdin
	if flag.NArg() > 0 {
		inputFile, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Printf("Failed to open input file: %v\n", err)
			os.Exit(1)
		}
		defer inputFile.Close()
		input = inputFile
	}

	// Determine output destination
	var output io.Writer = os.Stdout
	if !*outputToStdout {
		if *outputFile != "" {
			outFile, err := os.Create(*outputFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create output file: %v\n", err)
				os.Exit(1)
			}
			defer outFile.Close()
			output = outFile
		} else if *outputFile == "" && flag.NArg() == 0 {
			fmt.Fprintf(os.Stderr, "No output file specified and no input provided, writing to stdout.")
		}
	}

	// Handle compression/decompression
	if *compressMode {
		err := decompress(input, output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Decompression failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		err := compress(input, output, *compressionLevel, *numThreads)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Compression failed: %v\n", err)
			os.Exit(1)
		}
	}
}
