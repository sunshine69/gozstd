package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/klauspost/compress/zstd"
)

func compressBlock(input []byte, compressionLevel int) ([]byte, error) {
	var buf bytes.Buffer
	options := []zstd.EOption{zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressionLevel))}
	encoder, err := zstd.NewWriter(&buf, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	defer encoder.Close()

	_, err = encoder.Write(input)
	if err != nil {
		return nil, fmt.Errorf("failed to compress block: %w", err)
	}

	return buf.Bytes(), nil
}

func decompressBlock(input []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer decoder.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress block: %w", err)
	}

	return buf.Bytes(), nil
}

func compressFile(input io.Reader, output io.Writer, compressionLevel, numThreads int) error {
	const blockSize = 1 << 20 // 1 MB blocks

	type result struct {
		index int
		data  []byte
		err   error
	}

	var wg sync.WaitGroup
	inputChunks := make(chan []byte)
	results := make(chan result)

	// Launch worker goroutines
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for chunk := range inputChunks {
				compressedData, err := compressBlock(chunk, compressionLevel)
				results <- result{index: index, data: compressedData, err: err}
			}
		}(i)
	}

	// Read the input file in chunks and send to workers
	go func() {
		defer close(inputChunks)
		buf := make([]byte, blockSize)
		index := 0
		for {
			n, err := input.Read(buf)
			if err != nil && err != io.EOF {
				results <- result{index: index, err: fmt.Errorf("failed to read input file: %w", err)}
				return
			}
			if n == 0 {
				break
			}
			inputChunks <- buf[:n]
			index++
		}
	}()

	// Collect and write the compressed blocks in order
	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		if res.err != nil {
			return res.err
		}
		_, err := output.Write(res.data)
		if err != nil {
			return fmt.Errorf("failed to write compressed data: %w", err)
		}
	}

	return nil
}

func decompressFile(input io.Reader, output io.Writer) error {
	const blockSize = 1 << 20 // 1 MB blocks

	buf := make([]byte, blockSize)
	for {
		n, err := input.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read input file: %w", err)
		}
		if n == 0 {
			break
		}

		decompressedData, err := decompressBlock(buf[:n])
		if err != nil {
			return fmt.Errorf("failed to decompress block: %w", err)
		}

		_, err = output.Write(decompressedData)
		if err != nil {
			return fmt.Errorf("failed to write decompressed data: %w", err)
		}
	}

	return nil
}

var (
	version   string // Will hold the version number
	buildTime string // Will hold the build time
)

func printVersionBuildInfo() {
	fmt.Fprintf(os.Stderr, "Version: %s\nBuild time: %s\n", version, buildTime)
}
func main() {
	// Define flags
	compressMode := flag.Bool("d", false, "Decompress instead of compress")
	outputToStdout := flag.Bool("c", false, "Write output to stdout")
	outputFile := flag.String("o", "", "Output file (default: stdout)")
	compressionLevel := flag.Int("l", 3, "Set compression level (1-19, default: 3)")
	numThreads := flag.Int("T", 4, "Number of threads for block-based compression (default: 4)")
	flag.Usage = func() {
		printVersionBuildInfo()
		flag.PrintDefaults()
	}
	// Parse flags
	flag.Parse()

	// Determine input source
	var input io.Reader = os.Stdin
	if flag.NArg() > 0 {
		inputFile, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open input file: %v\n", err)
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
		}
	}

	// Handle compression/decompression
	if *compressMode {
		err := decompressFile(input, output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Decompression failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		err := compressFile(input, output, *compressionLevel, *numThreads)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Compression failed: %v\n", err)
			os.Exit(1)
		}
	}
}
