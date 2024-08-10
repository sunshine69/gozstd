package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/DataDog/zstd"
)

// compressData compresses the input data using zstd and returns the compressed data.
func compressData(input []byte, level, thread int) ([]byte, error) {
	var compressedData bytes.Buffer
	encoder := zstd.NewWriterLevel(&compressedData, level)

	defer encoder.Close()
	// If more than 1 it crashed famous double free or corruption memory coming from C :P
	encoder.SetNbWorkers(thread)

	_, err := encoder.Write(input)
	if err != nil {
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	err = encoder.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close encoder: %w", err)
	}

	return compressedData.Bytes(), nil
}

// decompressData decompresses the input data using zstd and returns the decompressed data.
func decompressData(input []byte) ([]byte, error) {
	var decompressedData bytes.Buffer
	reader := zstd.NewReader(bytes.NewReader(input))

	defer reader.Close()

	_, err := io.Copy(&decompressedData, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	return decompressedData.Bytes(), nil
}

func main() {
	// Define command-line flags
	level := flag.Int("level", 15, "Compression level (0-22)")
	thread := flag.Int("T", 2, "Thread count")
	outputFile := flag.String("o", "", "Output file name (default: stdout)")
	stdout := flag.Bool("c", false, "Write compressed data to stdout")
	decompress := flag.Bool("d", false, "Decompress mode")
	flag.Parse()

	// Determine input file from remaining arguments
	var inputFile string
	if len(flag.Args()) > 0 {
		inputFile = flag.Args()[0]
	}

	// Read input file or from stdin
	var data []byte
	var err error
	if inputFile != "" {
		data, err = os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read file: %v\n", err)
			os.Exit(1)
		}
	} else {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read from stdin: %v\n", err)
			os.Exit(1)
		}
	}

	if *decompress {
		// Decompress the data
		decompressedData, err := decompressData(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Decompression failed: %v\n", err)
			os.Exit(1)
		}

		// Write decompressed data to output file or stdout
		if *stdout {
			_, err = os.Stdout.Write(decompressedData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write to stdout: %v\n", err)
				os.Exit(1)
			}
		} else if *outputFile != "" {
			err = os.WriteFile(*outputFile, decompressedData, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write to file: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "No output destination specified. Use -c for stdout or -o for a file.\n")
			os.Exit(1)
		}
	} else {
		// Compress the data
		compressedData, err := compressData(data, *level, *thread)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Compression failed: %v\n", err)
			os.Exit(1)
		}

		// Write compressed data to output file or stdout
		if *stdout {
			_, err = os.Stdout.Write(compressedData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write to stdout: %v\n", err)
				os.Exit(1)
			}
		} else if *outputFile != "" {
			err = os.WriteFile(*outputFile, compressedData, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write to file: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "No output destination specified. Use -c for stdout or -o for a file.\n")
			os.Exit(1)
		}
	}
}
