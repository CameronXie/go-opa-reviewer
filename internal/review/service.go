package review

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/CameronXie/go-opa-reviewer/pkg/reviewer"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog"
)

type File struct {
	Name    string
	Content []byte
}

type Result struct {
	File   string
	Output []byte
	Error  error
}

type ReadFileFunc func(ctx context.Context, file string) ([]byte, error)

type Service interface {
	Review(context.Context, ReadFileFunc, []string) ([]Result, error)
}

type service struct {
	readerPoolSize   int
	reviewerPoolSize int
	fileReviewer     reviewer.Reviewer
}

// Review is a method that orchestrates the file reading and file reviewing processes.
// It creates goroutine pools for reading files and reviewing files, and manages the communication between them through channels.
// The method takes a ReadFileFunc, which is a function for reading files,
// and a slice of paths representing the files to be read and reviewed.
// It returns a slice of Result containing the output of the review process for each file, and an error if any occurred.
func (s *service) Review(ctx context.Context, read ReadFileFunc, paths []string) ([]Result, error) {
	fileChan := make(chan File)
	resultChan := make(chan Result)
	errorChan := make(chan error)

	var readerWG sync.WaitGroup
	var reviewerWG sync.WaitGroup

	readerPool, readerPoolErr := s.setupReaderPoolWithFunc(ctx, &readerWG, read, fileChan, resultChan)
	if readerPoolErr != nil {
		return nil, readerPoolErr
	}
	defer readerPool.Release()

	reviewerPool, reviewerPoolErr := s.setupReviewerPoolWithFunc(ctx, &reviewerWG, resultChan)
	if reviewerPoolErr != nil {
		return nil, reviewerPoolErr
	}
	defer reviewerPool.Release()

	go s.readFile(&readerWG, readerPool, paths, fileChan, errorChan)
	go s.reviewFile(&reviewerWG, reviewerPool, fileChan, resultChan, errorChan)

	results := make([]Result, 0)
	var errs error

	// Collecting review results and errors.
	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				return results, errs
			}

			results = append(results, result)
		case err, ok := <-errorChan:
			if ok {
				errs = errors.Join(errs, err)
			}
		}
	}
}

// setupReaderPoolWithFunc is a method that creates and configures a pool of goroutines to execute the read function.
func (s *service) setupReaderPoolWithFunc(
	ctx context.Context,
	wg *sync.WaitGroup,
	read ReadFileFunc,
	fileChan chan<- File,
	resultChan chan<- Result,
) (*ants.PoolWithFunc, error) {
	logger := zerolog.Ctx(ctx)

	return ants.NewPoolWithFunc(s.readerPoolSize, func(input any) {
		defer wg.Done()

		path := input.(string)
		logger.Debug().Msgf("reading file from %s", path)

		content, err := read(ctx, path)
		if err != nil {
			resultChan <- Result{
				File:  path,
				Error: processFileErr(err, "read"),
			}
			return
		}

		fileChan <- File{
			Name:    path,
			Content: content,
		}
	}, ants.WithLogger(logger))
}

// setupReviewerPoolWithFunc is a method that creates a goroutine pool with a function to review files.
func (s *service) setupReviewerPoolWithFunc(
	ctx context.Context,
	wg *sync.WaitGroup,
	resultChan chan<- Result,
) (*ants.PoolWithFunc, error) {
	logger := zerolog.Ctx(ctx)
	return ants.NewPoolWithFunc(s.reviewerPoolSize, func(input any) {
		defer wg.Done()

		file := input.(File)
		logger.Debug().Msgf("reviewing file %s", file.Name)

		result, err := s.fileReviewer.Review(context.TODO(), file.Content)
		if err != nil {
			resultChan <- Result{
				File:  file.Name,
				Error: processFileErr(err, "review"),
			}
			return
		}

		resultChan <- Result{
			File:   file.Name,
			Output: result,
		}
	}, ants.WithLogger(logger))
}

// readFile is a method that loop through paths, reads files and sends them to the fileChan channel for processing.
func (s *service) readFile(
	wg *sync.WaitGroup,
	pool *ants.PoolWithFunc,
	paths []string,
	fileChan chan<- File,
	errorChan chan<- error,
) {
	for idx := range paths {
		wg.Add(1)
		if err := pool.Invoke(paths[idx]); err != nil {
			wg.Done()
			errorChan <- taskSubmitErr(err, "reader")
			continue
		}
	}

	wg.Wait()
	close(fileChan)
}

// reviewFile is a goroutine that processes the files received from the fileChan channel.
func (s *service) reviewFile(
	wg *sync.WaitGroup,
	pool *ants.PoolWithFunc,
	fileChan <-chan File,
	resultChan chan<- Result,
	errorChan chan<- error,
) {
	for file := range fileChan {
		wg.Add(1)
		if err := pool.Invoke(file); err != nil {
			wg.Done()
			errorChan <- taskSubmitErr(err, "reviewer")
			continue
		}
	}

	wg.Wait()
	close(resultChan)
	close(errorChan)
}

func processFileErr(err error, process string) error {
	return fmt.Errorf("failed to %s file: %w", process, err)
}

func taskSubmitErr(err error, poolType string) error {
	return fmt.Errorf("failed to submit a task to %s pool: %w", poolType, err)
}

func New(
	fileReviewer reviewer.Reviewer,
	readerPoolSize int,
	reviewerPoolSize int,
) (Service, error) {
	return &service{
		readerPoolSize:   readerPoolSize,
		reviewerPoolSize: reviewerPoolSize,
		fileReviewer:     fileReviewer,
	}, nil
}
