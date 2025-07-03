package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"golang.org/x/sync/errgroup"

	"github.com/kanwren/norml/pkg/normalizer"
)

type normalizeCmd struct {
	InPlace bool
	Files   []string
	Workers int
	Verbose bool
	Version bool
}

func normalizeInPlace(ctx context.Context, logger *log.Logger, files []string, numWorkers int) error {
	g, egCtx := errgroup.WithContext(ctx)

	filesChan := make(chan string, len(files))

	for range numWorkers {
		g.Go(func() error {
			for filename := range filesChan {
				if egCtx.Err() != nil {
					return egCtx.Err()
				}

				logger.Printf("normalizing file: %s", filename)
				if err := normalizer.NormalizeFile(filename); err != nil {
					return fmt.Errorf("failed to normalize file %s: %w", filename, err)
				}
			}
			return nil
		})
	}

	for _, file := range files {
		filesChan <- file
	}
	close(filesChan)

	return g.Wait()
}

type fileInfo struct {
	filename string
	index    int
}

type fileResult struct {
	filename string
	content  []byte
	index    int
}

func normalizeTo(ctx context.Context, logger *log.Logger, w io.Writer, files []string, numWorkers int) error {
	filesChan := make(chan fileInfo, len(files))
	resultsChan := make(chan fileResult, len(files))

	workers, workersCtx := errgroup.WithContext(ctx)
	for range numWorkers {
		workers.Go(func() error {
			for info := range filesChan {
				if workersCtx.Err() != nil {
					return workersCtx.Err()
				}

				filename := info.filename
				index := info.index

				logger.Printf("normalizing file: %s", filename)

				file, err := os.Open(filename)
				if err != nil {
					return fmt.Errorf("failed to open file %s: %w", filename, err)
				}

				buf := new(bytes.Buffer)
				err = normalizer.Normalize(file, buf)
				file.Close()
				if err != nil {
					return fmt.Errorf("failed to normalize file %s: %w", filename, err)
				}

				resultsChan <- fileResult{
					filename: filename,
					index:    index,
					content:  buf.Bytes(),
				}
			}
			return nil
		})
	}

	reader, readerCtx := errgroup.WithContext(ctx)
	reader.Go(func() error {
		nextIndex := 0
		results := make(map[int][]byte)

		for result := range resultsChan {
			if readerCtx.Err() != nil {
				return readerCtx.Err()
			}

			results[result.index] = result.content

			if result.index == nextIndex {
				for doc, exists := results[nextIndex]; exists; doc, exists = results[nextIndex] {
					if nextIndex > 0 {
						if _, err := w.Write([]byte("---\n")); err != nil {
							return fmt.Errorf("failed to write document delimiter: %w", err)
						}
					}

					if _, err := w.Write(doc); err != nil {
						return fmt.Errorf("failed to write to stdout: %w", err)
					}

					delete(results, nextIndex)
					nextIndex++
				}
			}
		}

		return nil
	})

	for i, filename := range files {
		filesChan <- fileInfo{filename: filename, index: i}
	}
	close(filesChan)

	if err := workers.Wait(); err != nil {
		return err
	}
	close(resultsChan)

	return reader.Wait()
}

func run(ctx context.Context, logger *log.Logger, stdin io.Reader, stdout io.Writer, args []string) error {
	cmd := &normalizeCmd{}

	flags := flag.NewFlagSet("norml", flag.ExitOnError)

	numCPU := runtime.NumCPU()

	flags.BoolVar(&cmd.InPlace, "i", false, "Edit files in-place")
	flags.IntVar(&cmd.Workers, "j", numCPU, "Number of parallel workers (default: number of CPUs)")
	flags.BoolVar(&cmd.Verbose, "v", false, "Verbose output")
	flags.BoolVar(&cmd.Version, "version", false, "Print version and exit")

	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	cmd.Files = flags.Args()

	if cmd.Workers <= 0 {
		cmd.Workers = runtime.NumCPU()
	}
	if !cmd.Verbose {
		logger.SetOutput(io.Discard)
	}
	if len(cmd.Files) < cmd.Workers {
		cmd.Workers = len(cmd.Files)
	}

	if cmd.Version {
		fmt.Fprintln(stdout, Version())
		return nil
	}

	if len(cmd.Files) == 0 {
		logger.Println("No files specified, reading from stdin")
		return normalizer.Normalize(stdin, stdout)
	}
	if cmd.InPlace {
		return normalizeInPlace(ctx, logger, cmd.Files, cmd.Workers)
	} else {
		return normalizeTo(ctx, logger, stdout, cmd.Files, cmd.Workers)
	}
}

func main() {
	ctx := context.Background()

	logger := log.New(os.Stderr, "", log.LstdFlags)

	if err := run(ctx, logger, os.Stdin, os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
