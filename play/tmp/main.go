package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// compressSegment compresses a segment of data and writes the result to the output.
func compressSegment(input io.Reader, output io.Writer, compressionLevel int) error {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressionLevel)))
	if err != nil {
		return fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	defer encoder.Close()

	buf := make([]byte, 1<<20) // 1 MB buffer
	for {
		n, err := input.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read input: %w", err)
		}
		if n == 0 {
			break
		}

		compressed := encoder.EncodeAll(buf[:n], nil)
		_, err = output.Write(compressed)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}

// compressFileBlock compresses the input file using block mode and writes the result to the output.
func compressFileBlock(input *os.File, output io.Writer, compressionLevel, numThreads int) error {
	info, err := input.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat input file: %w", err)
	}

	fileSize := info.Size()
	partSize := fileSize / int64(numThreads)
	var wg sync.WaitGroup
	errChan := make(chan error, numThreads)
	compressedData := make([][]byte, numThreads)

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := int64(i) * partSize
			end := start + partSize
			if i == numThreads-1 {
				end = fileSize // last chunk gets any remaining data
			}

			partFile, err := os.Open(input.Name())
			if err != nil {
				errChan <- fmt.Errorf("failed to open input file: %w", err)
				return
			}
			defer partFile.Close()

			_, err = partFile.Seek(start, 0)
			if err != nil {
				errChan <- fmt.Errorf("failed to seek input file: %w", err)
				return
			}

			pr, pw := io.Pipe()
			defer pw.Close()

			go func() {
				defer pw.Close()
				limitedReader := io.LimitReader(partFile, end-start)
				err := compressSegment(limitedReader, pw, compressionLevel)
				if err != nil {
					errChan <- err
				}
			}()

			chunkData := make([]byte, 0)
			for {
				data := make([]byte, 1<<20)
				n, err := pr.Read(data)
				if err != nil && err != io.EOF {
					errChan <- fmt.Errorf("failed to read compressed data: %w", err)
					return
				}
				if n == 0 {
					break
				}
				chunkData = append(chunkData, data[:n]...)
			}
			compressedData[i] = chunkData
		}(i)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	for _, chunk := range compressedData {
		_, err = output.Write(chunk)
		if err != nil {
			return fmt.Errorf("failed to write compressed data: %w", err)
		}
	}

	return nil
}

// compressStream compresses data using stream mode and writes it to the output.
func compressStream(input io.Reader, output io.Writer, compressionLevel int) error {
	encoder, err := zstd.NewWriter(output, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressionLevel)))
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

// decompressFile decompresses data and writes it to the output.
func decompressFile(input io.Reader, output io.Writer) error {
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
	compressionLevel := flag.Int("1", 3, "Set compression level (1-19, default: 3)")
	numThreads := flag.Int("T", 4, "Number of threads for compression (default: 4)")
	blockMode := flag.Bool("block", false, "Use block mode for compression")

	// Parse flags
	flag.Parse()

	// Determine input source
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
				fmt.Printf("Failed to create output file: %v\n", err)
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
			fmt.Printf("Decompression failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		if *blockMode {
			inputFile, ok := input.(*os.File)
			if !ok {
				fmt.Println("Block mode requires a file input.")
				os.Exit(1)
			}

			err := compressFileBlock(inputFile, output, *compressionLevel, *numThreads)
			if err != nil {
				fmt.Printf("Block mode compression failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			err := compressStream(input, output, *compressionLevel)
			if err != nil {
				fmt.Printf("Stream mode compression failed: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
