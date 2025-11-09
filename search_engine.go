package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// SearchWithProgress performs a search and emits progress updates to the frontend
func (a *App) SearchWithProgress(req SearchRequest) ([]SearchResult, error) {
	// Log the start of the search operation
	searchStart := time.Now()
	a.logInfo("Starting search operation", logrus.Fields{
		"directory":     req.Directory,
		"query":         req.Query,
		"extension":     req.Extension,
		"caseSensitive": req.CaseSensitive,
		"useRegex":      req.UseRegex,
		"maxFileSize":   req.MaxFileSize,
		"maxResults":    req.MaxResults,
		"includeBinary": req.IncludeBinary,
		"searchSubdirs": req.SearchSubdirs,
		"excludeCount":  len(req.ExcludePatterns),
		"allowedTypes":  req.AllowedFileTypes,
	})

	// Validate and set defaults for parameters
	validatedReq, err := a.validateAndSetDefaults(req)
	if err != nil {
		a.logError("Search request validation failed", err, logrus.Fields{
			"directory": req.Directory,
			"query":     req.Query,
		})
		return nil, err
	}
	req = validatedReq

	// If query is empty, return empty results instead of error to maintain compatibility
	if req.Query == "" {
		a.logWarn("Empty query provided, returning empty results", logrus.Fields{
			"directory": req.Directory,
		})
		return []SearchResult{}, nil
	}

	// Prepare search pattern based on case sensitivity and regex requirements
	pattern, err := a.compileSearchPattern(req)
	if err != nil {
		a.logError("Failed to compile search pattern", err, logrus.Fields{
			"query":         req.Query,
			"useRegex":      req.UseRegex,
			"caseSensitive": req.CaseSensitive,
		})
		return nil, err
	}

	// Get the base directory for path traversal check
	absDir, err := filepath.Abs(req.Directory)
	if err != nil {
		a.logError("Failed to get absolute path for directory", err, logrus.Fields{
			"directory": req.Directory,
		})
		return nil, fmt.Errorf("failed to get absolute path for directory: %v", err)
	}
	baseDir := filepath.Clean(absDir) + string(filepath.Separator)

	// Collect all files to process based on search criteria
	a.logDebug("Collecting files to process", logrus.Fields{
		"directory": req.Directory,
	})
	filesToProcess, err := a.collectFilesToProcess(req, pattern, baseDir)
	if err != nil {
		a.logError("Failed to collect files to process", err, logrus.Fields{
			"directory": req.Directory,
			"query":     req.Query,
		})
		return nil, err
	}

	totalFiles := len(filesToProcess)
	a.logInfo("File collection completed", logrus.Fields{
		"totalFiles": totalFiles,
		"directory":  req.Directory,
	})

	// Emit initial progress via both Wails events and WebSocket
	progressData := map[string]interface{}{
		"processedFiles": 0,
		"totalFiles":     totalFiles,
		"currentFile":    "",
		"resultsCount":   0,
		"status":         "started",
	}

	a.logInfo("Sending initial search progress", logrus.Fields{
		"status":       "started",
		"totalFiles":   totalFiles,
		"currentFile":  "",
		"resultsCount": 0,
	})

	a.safeEmitEvent("search-progress", progressData)

	// Create search context with cancellation
	ctx, cancel := a.createSearchContext()
	defer func() {
		// Clear the cancel function when the search completes
		a.searchCancel = nil
		cancel()
	}()

	// Log search start
	a.logInfo("Starting file processing with worker pool", logrus.Fields{
		"totalFiles": totalFiles,
		"workers":    numCPU(),
		"maxResults": req.MaxResults,
	})

	// Process files using worker pool
	resultsChan, searchState := a.processFilesWithWorkers(ctx, filesToProcess, req, pattern, baseDir, totalFiles)

	// Collect results
	var results []SearchResult
	for result := range resultsChan {
		results = append(results, result)

		// Check if we've reached the result limit
		if len(results) >= req.MaxResults {
			a.logInfo("Reached maximum results limit, stopping search", logrus.Fields{
				"resultsCount": len(results),
				"maxResults":   req.MaxResults,
			})
			// The context is already cancelled by the workers, but we'll do it again just in case
			if a.searchCancel != nil {
				a.searchCancel()
			}
			// Trim results to max results if somehow we got more
			if len(results) > req.MaxResults {
				results = results[:req.MaxResults]
			}
			break
		}
	}

	// Emit final progress
	finalProgressData := map[string]interface{}{
		"processedFiles": int(atomic.LoadInt32(&searchState.processedFiles)),
		"totalFiles":     totalFiles,
		"currentFile":    "",
		"resultsCount":   len(results),
		"status":         "completed",
	}

	a.logInfo("Sending final search progress", logrus.Fields{
		"status":         "completed",
		"processedFiles": int(atomic.LoadInt32(&searchState.processedFiles)),
		"totalFiles":     totalFiles,
		"resultsCount":   len(results),
	})

	a.safeEmitEvent("search-progress", finalProgressData)

	// Log search completion
	duration := time.Since(searchStart)
	a.logInfo("Search operation completed", logrus.Fields{
		"resultsCount":    len(results),
		"processedFiles":  int(atomic.LoadInt32(&searchState.processedFiles)),
		"totalFiles":      totalFiles,
		"durationSeconds": duration.Seconds(),
		"directory":       req.Directory,
		"query":           req.Query,
	})

	return results, nil
}

// collectFilesToProcess walks the directory tree and collects all files to process based on search criteria
func (a *App) collectFilesToProcess(req SearchRequest, pattern *regexp.Regexp, baseDir string) ([]string, error) {
	var filesToProcess []string
	filesSkipped := 0
	dirsSkipped := 0

	err := filepath.WalkDir(req.Directory, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// If there's an error accessing a file/directory, skip it and continue
			a.logDebug("Skipping file/directory due to access error", logrus.Fields{
				"path":  path,
				"error": walkErr.Error(),
			})
			return nil
		}

		// Check for path traversal during walk
		absPath, err := filepath.Abs(path)
		if err != nil {
			a.logDebug("Skipping file due to absolute path error", logrus.Fields{
				"path":  path,
				"error": err.Error(),
			})
			return nil // Skip if we can't get absolute path
		}
		relPath, err := filepath.Rel(baseDir, absPath)
		if err != nil || strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
			// This path is outside the base directory - skip it
			a.logDebug("Skipping file due to path traversal detection", logrus.Fields{
				"path":    path,
				"relPath": relPath,
				"baseDir": baseDir,
			})
			if d.IsDir() {
				dirsSkipped++
				return filepath.SkipDir // Skip the entire subdirectory
			}
			return nil
		}

		if d.IsDir() {
			// Skip hidden directories that start with a dot (e.g., .git, .vscode)
			if strings.HasPrefix(d.Name(), ".") {
				a.logDebug("Skipping hidden directory", logrus.Fields{
					"directory": path,
				})
				dirsSkipped++
				return filepath.SkipDir
			}
			return nil
		}

		// Apply file extension filter if specified
		if req.Extension != "" {
			if !matchExtension(path, req.Extension) {
				a.logDebug("Skipping file due to extension filter", logrus.Fields{
					"path":      path,
					"extension": req.Extension,
				})
				filesSkipped++
				return nil
			}
		}

		// If allow list is specified, check if the file type is allowed
		if len(req.AllowedFileTypes) > 0 {
			isAllowed := false
			for _, allowedExt := range req.AllowedFileTypes {
				if matchExtension(path, allowedExt) {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				a.logDebug("Skipping file due to allowed types filter", logrus.Fields{
					"path":         path,
					"allowedTypes": req.AllowedFileTypes,
				})
				filesSkipped++
				return nil
			}
		}

		// Get file information to check size before reading
		fileInfo, err := d.Info()
		if err != nil {
			a.logDebug("Skipping file due to info error", logrus.Fields{
				"path":  path,
				"error": err.Error(),
			})
			return nil // Skip if we can't get file info
		}

		// Skip very large files to prevent memory issues
		if fileInfo.Size() > req.MaxFileSize {
			a.logDebug("Skipping large file due to size limit", logrus.Fields{
				"path":     path,
				"fileSize": fileInfo.Size(),
				"maxSize":  req.MaxFileSize,
			})
			filesSkipped++
			return nil
		}

		// Skip very small files based on min file size
		if fileInfo.Size() < req.MinFileSize {
			a.logDebug("Skipping small file due to size filter", logrus.Fields{
				"path":     path,
				"fileSize": fileInfo.Size(),
				"minSize":  req.MinFileSize,
			})
			filesSkipped++
			return nil
		}

		// Check exclude patterns
		for _, patternStr := range req.ExcludePatterns {
			if patternStr != "" && a.matchesPattern(path, patternStr) {
				a.logDebug("Skipping file due to exclude pattern", logrus.Fields{
					"path":        path,
					"excludePath": patternStr,
				})
				filesSkipped++
				return nil
			}
		}

		// If not including binary files, check if this file is binary and skip if it is
		// Read only the first portion of the file for binary detection to avoid memory issues
		if !req.IncludeBinary {
			file, err := os.Open(path)

			if err == nil {
				defer file.Close()
				// Read only the first 512 bytes to check for binary content
				buffer := make([]byte, 512)
				n, _ := file.Read(buffer)
				if n > 0 && a.isBinary(buffer[:n]) {
					a.logDebug("Skipping binary file", logrus.Fields{
						"path": path,
					})
					filesSkipped++
					return nil // Skip binary files
				}
			} else {
				a.logDebug("Skipping file due to read error for binary check", logrus.Fields{
					"path":  path,
					"error": err.Error(),
				})
				filesSkipped++
				return nil
			}
		}

		filesToProcess = append(filesToProcess, path)
		return nil
	})
	if err != nil {
		a.logError("Error during file walk", err, logrus.Fields{
			"directory": req.Directory,
		})
		return nil, err
	}

	a.logInfo("File collection completed", logrus.Fields{
		"filesProcessed": len(filesToProcess),
		"filesSkipped":   filesSkipped,
		"dirsSkipped":    dirsSkipped,
		"directory":      req.Directory,
	})

	return filesToProcess, nil
}

// processFileLineByLine processes a file line by line to avoid loading large files into memory
func (a *App) processFileLineByLine(ctx context.Context, filePath string, pattern *regexp.Regexp, maxResults int, includeBinary bool) ([]SearchResult, error) {
	a.logDebug("Starting line-by-line file processing", logrus.Fields{
		"filePath":      filePath,
		"maxResults":    maxResults,
		"includeBinary": includeBinary,
	})

	file, err := os.Open(filePath)
	if err != nil {
		a.logError("Failed to open file for line-by-line processing", err, logrus.Fields{
			"filePath": filePath,
		})
		return nil, err
	}
	defer file.Close()

	// If not including binary files, check if this file is binary and skip if it is
	// Read only the first portion of the file for binary detection
	if !includeBinary {
		buffer := make([]byte, 512)
		n, err := file.Read(buffer)
		if err == nil && n > 0 && a.isBinary(buffer[:n]) {
			a.logDebug("Skipping binary file during line-by-line processing", logrus.Fields{
				"filePath": filePath,
			})
			return []SearchResult{}, nil // Return empty results for binary files
		}
		// Reset file pointer back to beginning for processing
		file.Seek(0, 0)
	}

	var results []SearchResult
	scanner := bufio.NewScanner(file)

	// Set a larger buffer for very long lines (1MB)
	buf := make([]byte, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, 1024*1024)

	lineNum := 1
	linesProcessed := 0
	for scanner.Scan() && len(results) < maxResults {
		line := scanner.Text()
		if pattern.MatchString(line) {
			result := SearchResult{
				FilePath:      filePath,
				LineNum:       lineNum,
				Content:       strings.TrimSpace(line),
				MatchedText:   "",         // Will be set later with actual matched text
				ContextBefore: []string{}, // Context lines are not collected in streaming mode
				ContextAfter:  []string{},
			}
			// Set the matched text from the actual match
			matches := pattern.FindString(line)
			if matches != "" {
				result.MatchedText = matches
			}
			results = append(results, result)
		}

		lineNum++
		linesProcessed++

		// Check for context cancellation every 100 lines to avoid performance impact
		if linesProcessed%100 == 0 {
			select {
			case <-ctx.Done(): // Use the specific search context to check for cancellation
				// Context was cancelled externally
				a.logDebug("Line-by-line processing cancelled due to context", logrus.Fields{
					"filePath":       filePath,
					"linesProcessed": linesProcessed,
					"resultsFound":   len(results),
				})
				return results, nil
			default:
				// Continue processing
			}
		}
	}

	if err := scanner.Err(); err != nil {
		a.logError("Error during line-by-line scanning", err, logrus.Fields{
			"filePath": filePath,
		})
		return nil, err
	}

	a.logDebug("Completed line-by-line file processing", logrus.Fields{
		"filePath":       filePath,
		"resultsFound":   len(results),
		"linesProcessed": linesProcessed,
	})
	return results, nil
}

// Helper function to get number of CPUs
func numCPU() int {
	n := runtime.NumCPU()
	if n < 2 {
		return 2 // Use at least 2 workers for parallelism
	}
	return n
}

// createSearchContext creates a context for the search operation with associated cancellation
func (a *App) createSearchContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	// Store the cancel function so it can be called externally to cancel the search
	a.searchCancel = cancel
	return ctx, cancel
}

// processFilesWithWorkers processes files using a worker pool and returns a channel of results
func (a *App) processFilesWithWorkers(ctx context.Context, filesToProcess []string, req SearchRequest, pattern *regexp.Regexp, baseDir string, totalFiles int) (chan SearchResult, *SearchState) {
	// Use a worker pool to process files in parallel
	numWorkers := numCPU()
	if len(filesToProcess) < numWorkers {
		numWorkers = len(filesToProcess)
	}

	// Log worker pool details
	a.logDebug("Initializing worker pool", logrus.Fields{
		"numWorkers":         numWorkers,
		"totalFiles":         totalFiles,
		"maxResults":         req.MaxResults,
		"streamingThreshold": 1024 * 1024, // 1MB
	})

	// Create channels
	filesChan := make(chan string, len(filesToProcess))
	resultsChan := make(chan SearchResult, 100)

	// Track progress
	searchState := &SearchState{
		processedFiles: 0,
		resultsCount:   0,
	}

	// Create atomic flag to track if cancellation has been triggered to prevent multiple cancellations
	var searchCancelled int32 = 0

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					// Context cancelled, stop processing and exit
					a.logDebug("Worker stopped due to context cancellation", logrus.Fields{
						"workerID": workerID,
					})
					return
				case filePath, ok := <-filesChan:
					if !ok {
						// Channel closed, exit worker
						a.logDebug("Worker exiting, file channel closed", logrus.Fields{
							"workerID": workerID,
						})
						return
					}
					// Check if we've already reached the max results
					currentResults := int(atomic.LoadInt32(&searchState.resultsCount))
					if currentResults >= req.MaxResults {
						// Only cancel if not already cancelled to prevent race conditions
						if atomic.CompareAndSwapInt32(&searchCancelled, 0, 1) {
							// The context is already stored in a.searchCancel, so we use that
							if a.searchCancel != nil {
								a.searchCancel()
							}
						}
						return
					}

					// Check if context has been cancelled before processing each file
					select {
					case <-ctx.Done():
						// Context cancelled, stop processing
						a.logDebug("Worker stopping due to context cancellation during file processing", logrus.Fields{
							"workerID": workerID,
							"filePath": filePath,
						})
						return
					default:
						// Context is still active, continue processing
					}

					// Get file info to determine if it's a large file that should be processed in streaming mode
					fileInfo, statErr := os.Stat(filePath)
					if statErr != nil {
						a.logDebug("Skipping file due to stat error", logrus.Fields{
							"workerID": workerID,
							"filePath": filePath,
							"error":    statErr.Error(),
						})
						continue // Skip if we can't get file info
					}

					// For larger files, use streaming line-by-line processing to avoid memory issues
					// Threshold is set to 1MB (can be adjusted as needed)
					const streamingThreshold = 1024 * 1024 // 1MB
					var fileResults []SearchResult

					// Additional path traversal check for the current file path
					absFilePath, absErr := filepath.Abs(filePath)
					if absErr != nil {
						a.logDebug("Skipping file due to absolute path error", logrus.Fields{
							"workerID": workerID,
							"filePath": filePath,
							"error":    absErr.Error(),
						})
						continue // Skip if we can't get absolute path
					}
					relFilePath, relErr := filepath.Rel(baseDir, absFilePath)
					if relErr != nil || strings.HasPrefix(relFilePath, "..") {
						a.logDebug("Skipping file due to path traversal check", logrus.Fields{
							"workerID": workerID,
							"filePath": filePath,
							"relPath":  relFilePath,
							"baseDir":  baseDir,
						})
						continue // Skip if file is outside the base directory
					}

					if fileInfo.Size() > streamingThreshold {
						// Use streaming approach for large files
						a.logDebug("Processing large file with streaming", logrus.Fields{
							"workerID":  workerID,
							"filePath":  absFilePath,
							"fileSize":  fileInfo.Size(),
							"threshold": streamingThreshold,
						})
						streamResults, procErr := a.processFileLineByLine(ctx, absFilePath, pattern, req.MaxResults-int(atomic.LoadInt32(&searchState.resultsCount)), req.IncludeBinary)
						if procErr != nil {
							a.logDebug("Error processing file with streaming", logrus.Fields{
								"workerID": workerID,
								"filePath": absFilePath,
								"error":    procErr.Error(),
							})
							continue // Skip problematic files
						}
						fileResults = streamResults
					} else {
						// Use original approach for smaller files (which is generally faster for small files)
						content, readErr := os.ReadFile(absFilePath)
						if readErr != nil {
							// Skip unreadable files (permissions, etc.)
							a.logDebug("Skipping file due to read error", logrus.Fields{
								"workerID": workerID,
								"filePath": absFilePath,
								"error":    readErr.Error(),
							})
							continue
						}

						// Check if file is binary if we're not including binary files
						if !req.IncludeBinary && a.isBinary(content) {
							a.logDebug("Skipping binary file (small)", logrus.Fields{
								"workerID": workerID,
								"filePath": absFilePath,
							})
							continue
						}

						// Split content into lines for line-by-line searching
						lines := strings.Split(string(content), "\n")
						for i, line := range lines {
							// Check again if we've reached max results before processing more
							if int(atomic.LoadInt32(&searchState.resultsCount)) >= req.MaxResults {
								// Only cancel if not already cancelled to prevent race conditions
								if atomic.CompareAndSwapInt32(&searchCancelled, 0, 1) {
									if a.searchCancel != nil {
										a.searchCancel()
									}
								}
								return
							}

							// Check if context has been cancelled during line processing
							if i%100 == 0 { // Check every 100 lines to avoid performance impact
								select {
								case <-ctx.Done():
									// Context cancelled, stop processing
									a.logDebug("Worker stopping due to context cancellation during line processing", logrus.Fields{
										"workerID": workerID,
										"filePath": absFilePath,
									})
									return
								default:
									// Context is still active, continue processing
								}
							}

							if pattern.MatchString(line) {
								// Calculate context lines (2 before, 2 after)
								contextBefore := []string{}
								contextAfter := []string{}

								// Get up to 2 lines before the match
								for j := i - 2; j < i; j++ {
									if j >= 0 {
										contextBefore = append(contextBefore, lines[j])
									}
								}

								// Get up to 2 lines after the match
								for j := i + 1; j <= i+2 && j < len(lines); j++ {
									contextAfter = append(contextAfter, lines[j])
								}

								// Found a match, send to results channel
								result := SearchResult{
									FilePath:      absFilePath,             // Use absolute cleaned path
									LineNum:       i + 1,                   // Convert to 1-indexed line numbers
									Content:       strings.TrimSpace(line), // Remove leading/trailing whitespace
									MatchedText:   req.Query,               // Store the original query as matched text
									ContextBefore: contextBefore,
									ContextAfter:  contextAfter,
								}

								fileResults = append(fileResults, result)
							}
						}
					}

					// Send all results from this file to the results channel
					for _, result := range fileResults {
						// Check again if max results reached before sending
						if int(atomic.LoadInt32(&searchState.resultsCount)) >= req.MaxResults {
							// Only cancel if not already cancelled to prevent race conditions
							if atomic.CompareAndSwapInt32(&searchCancelled, 0, 1) {
								if a.searchCancel != nil {
									a.searchCancel()
								}
							}
							return
						}

						// Use a non-blocking send with context check
						select {
						case resultsChan <- result:
							// Increment results count atomically
							newResultsCount := atomic.AddInt32(&searchState.resultsCount, 1)

							// Check if we've reached the result limit after incrementing
							if int(newResultsCount) >= req.MaxResults {
								// Only cancel if not already cancelled to prevent race conditions
								if atomic.CompareAndSwapInt32(&searchCancelled, 0, 1) {
									if a.searchCancel != nil {
										a.searchCancel()
									}
								}
							}
						case <-ctx.Done():
							// Context cancelled, stop processing
							return
						}
					}

					// Increment processed files count atomically
					newCount := atomic.AddInt32(&searchState.processedFiles, 1)

					// Emit progress update for each file to improve synchronization
					progressData := map[string]interface{}{
						"processedFiles": int(newCount),
						"totalFiles":     totalFiles,
						"currentFile":    absFilePath,
						"resultsCount":   int(atomic.LoadInt32(&searchState.resultsCount)),
						"status":         "in-progress",
					}

					a.logInfo("Sending file processing progress", logrus.Fields{
						"status":         "in-progress",
						"processedFiles": int(newCount),
						"totalFiles":     totalFiles,
						"currentFile":    absFilePath,
						"resultsCount":   int(atomic.LoadInt32(&searchState.resultsCount)),
					})

					a.safeEmitEvent("search-progress", progressData)
				}
			}
		}(i) // Pass the worker ID
	}

	// Send all files to the channel
	go func() {
		a.logDebug("Starting to send files to workers", logrus.Fields{
			"totalFiles": len(filesToProcess),
		})
		defer close(filesChan)
		for _, file := range filesToProcess {
			select {
			case <-ctx.Done():
				// Context cancelled, stop sending files
				a.logDebug("Stopping file sending due to context cancellation", logrus.Fields{
					"remainingFiles": len(filesToProcess),
				})
				return
			case filesChan <- file:
				// Continue sending files
			}
		}
	}()

	// Close resultsChan when all workers are done
	go func() {
		wg.Wait()
		a.logDebug("All workers completed, closing results channel", logrus.Fields{})
		close(resultsChan)
	}()

	return resultsChan, searchState
}

// CancelSearch cancels any active search operation by calling the cancel function
func (a *App) CancelSearch() error {
	if a.searchCancel != nil {
		a.logInfo("Cancelling active search", logrus.Fields{})
		a.searchCancel()
		// Emit cancellation progress event
		cancelData := map[string]interface{}{
			"processedFiles": 0,
			"totalFiles":     0,
			"currentFile":    "",
			"resultsCount":   0,
			"status":         "cancelled",
		}

		a.logInfo("Sending cancellation progress event", logrus.Fields{
			"status":         "cancelled",
			"processedFiles": 0,
			"totalFiles":     0,
			"resultsCount":   0,
		})
		a.safeEmitEvent("search-progress", cancelData)

		return nil
	}
	// If there's no active search to cancel, return an appropriate message
	a.logDebug("No active search to cancel", logrus.Fields{})
	return fmt.Errorf("no active search to cancel")
}
