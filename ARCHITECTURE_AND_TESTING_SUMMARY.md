# Code-Search-Golang Application Architecture & Testing Strategy

## Executive Summary

The code-search-golang application is a powerful, feature-rich desktop code search tool built with Wails (Go backend + Vue.js frontend). It enables users to search for text patterns, keywords, and regular expressions across code files in specified directories with advanced features like regex search, configurable file filtering, security-hardened file operations, exclude patterns, and pagination for better user experience. The application includes extensive performance optimizations, security hardening, and a modern, responsive UI with syntax highlighting.

This document provides a comprehensive overview of the application architecture, implementation details, testing strategy, and development best practices.

## Table of Contents
- [Architecture Overview](#architecture-overview)
- [Backend Architecture (Go)](#backend-architecture-go)
- [Frontend Architecture (Vue.js)](#frontend-architecture-vuejs)
- [Communication Layer (Wails)](#communication-layer-wails)
- [Performance Optimizations](#performance-optimizations)
- [Security Considerations](#security-considerations)
- [Testing Strategy](#testing-strategy)
- [Key Features Deep Dive](#key-features-deep-dive)
- [Development Workflow](#development-workflow)
- [Troubleshooting and Best Practices](#troubleshooting-and-best-practices)

## Architecture Overview

### System Design Principles
The application follows these core architectural principles:

- **Separation of Concerns**: Clear boundaries between frontend presentation logic, backend business logic, and communication layers
- **Performance First**: Optimized for handling large codebases with parallel processing and streaming
- **Security by Design**: Built-in protections against file system traversal and malicious inputs
- **Cross-Platform Compatibility**: Native experience across Windows, Linux, and macOS
- **Scalability**: Designed to handle large file trees and complex search operations efficiently

### High-Level Architecture
```
┌─────────────────┐    Wails Bindings    ┌─────────────────┐
│   Vue.js 3      │ ←──────────────────→ │      Go         │
│ Frontend Layer  │                      │  Backend Layer  │
│ (UI/UX Logic)   │                      │ (Search Engine) │
└─────────────────┘                      └─────────────────┘
                              │
                    File System Operations
                    Progress Events
                    System Integration
```

## Backend Architecture (Go)

### Core Components

#### App Structure
```go
type App struct {
    ctx          context.Context
    searchCancel context.CancelFunc // Cancel function for active searches
}
```

The `App` struct serves as the main entry point for all backend functionality, managing context for communication with the frontend and providing control over search operations.

#### SearchRequest Structure
```go
type SearchRequest struct {
    Directory        string   `json:"directory"`        // Path to the directory to search in
    Query            string   `json:"query"`            // Text to search for
    Extension        string   `json:"extension"`        // File extension to filter by (empty means all extensions)
    CaseSensitive    bool     `json:"caseSensitive"`    // Whether the search should be case sensitive
    IncludeBinary    bool     `json:"includeBinary"`    // Whether to include binary files in search
    MaxFileSize      int64    `json:"maxFileSize"`      // Maximum file size in bytes (default 10MB if 0)
    MinFileSize      int64    `json:"minFileSize"`      // Minimum file size in bytes (default 0 if not specified)
    MaxResults       int      `json:"maxResults"`       // Maximum number of results to return (default 1000 if 0)
    SearchSubdirs    bool     `json:"searchSubdirs"`    // Whether to search subdirectories (default true)
    UseRegex         *bool    `json:"useRegex"`         // Whether to treat query as regex (default true for backward compatibility)
    ExcludePatterns  []string `json:"excludePatterns"`  // Patterns to exclude from search (e.g., node_modules, *.log)
    AllowedFileTypes []string `json:"allowedFileTypes"` // List of file extensions that are allowed to be searched (if empty, all types allowed)
}
```

The `SearchRequest` structure encapsulates all parameters needed for search operations, providing flexibility and extensibility for various search scenarios.

#### SearchResult Structure
```go
type SearchResult struct {
    FilePath      string   `json:"filePath"`      // Full path to the file containing the match
    LineNum       int      `json:"lineNum"`       // Line number where the match was found (1-indexed)
    Content       string   `json:"content"`       // Content of the line containing the match
    MatchedText   string   `json:"matchedText"`   // The specific text that matched the query
    ContextBefore []string `json:"contextBefore"` // Lines before the match for context
    ContextAfter  []string `json:"contextAfter"`  // Lines after the match for context
}
```

The `SearchResult` structure provides comprehensive information about each match, including context lines for better understanding of the search results.

### Key Backend Functions

#### SearchWithProgress
The core search functionality with real-time progress updates:

- **Parallel Processing**: Uses Go goroutines with worker pools to process multiple files simultaneously
- **Memory Efficiency**: Implements streaming for large files (>1MB) to prevent memory issues
- **Early Termination**: Uses context cancellation to stop search when max results are reached
- **Progress Tracking**: Emits real-time updates to the frontend during the search process
- **Error Handling**: Comprehensive error management with graceful degradation

#### processFileLineByLine
Memory-efficient file processing for large files:

- **Streaming Approach**: Reads and processes files line-by-line to avoid memory issues
- **Binary Detection**: Checks for binary content without loading entire files into memory
- **Context Cancellation**: Respects search cancellation during line-by-line processing
- **Performance Optimization**: Efficient processing with configurable buffer sizes

#### isBinary
Binary file detection with multiple validation layers:

- **Null Byte Detection**: Identifies binary files containing null bytes
- **Printable Character Analysis**: Evaluates the ratio of printable vs. non-printable characters
- **Size-Based Detection**: Analyzes only the first 512 bytes for performance
- **Configurable Thresholds**: Allows adjustment of binary detection sensitivity

### Cross-Platform System Integration

#### SelectDirectory
Handles native directory selection across platforms:

- **Windows**: Uses PowerShell with System.Windows.Forms for native experience
- **Linux**: Tries multiple options in order of preference (zenity, kdialog, yad)
- **macOS**: Uses AppleScript (implementation in separate file)
- **Error Handling**: Comprehensive error management for unavailable system tools

#### ShowInFolder
Secure file manager integration with path traversal protection:

- **Path Sanitization**: Uses filepath.Clean to prevent directory traversal attacks
- **Validation Checks**: Ensures directory exists and is accessible before opening
- **Cross-Platform Commands**: Uses appropriate OS commands (xdg-open, cmd /c start)
- **Security Validation**: Prevents access to parent directories via traversal attempts

## Frontend Architecture (Vue.js)

### Component Architecture

#### CodeSearch.vue (Main Component)
- **Single Responsibility**: Orchestrates the entire search workflow
- **Composable Integration**: Uses `useSearch` composable for all business logic
- **UI Composition**: Aggregates child components (SearchForm, ProgressIndicator, SearchResults)

#### SearchForm.vue
- **User Input Management**: Handles all search parameters and options
- **Validation**: Client-side validation before backend calls
- **Recent Searches**: Integrates with localStorage for search history

#### SearchResults.vue
- **Result Presentation**: Displays search results with pagination
- **Interactive Features**: Provides copy, open folder, and preview functionality
- **Performance**: Implements pagination to handle large result sets

#### ProgressIndicator.vue
- **Real-time Updates**: Displays search progress with detailed metrics
- **Visual Feedback**: Provides clear indication of search status

#### CodeModal.vue
- **Code Preview**: Displays file content with syntax highlighting
- **Navigation**: Allows navigation between search matches
- **Performance**: Implements truncation limits to prevent performance issues

### Composable Pattern

#### useSearch (Composition Composable)
The `useSearch` composable centralizes all search business logic:

- **State Management**: Manages all reactive state for the search functionality
- **Backend Integration**: Handles all communication with Go backend
- **Event Handling**: Manages progress updates and search lifecycle
- **Persistence**: Handles localStorage operations for recent searches
- **Error Handling**: Provides comprehensive error management and user feedback

#### Reactive State Structure
The composable manages the following reactive state:

```typescript
interface SearchState {
  directory: string;                    // Directory path to search in
  query: string;                        // Search query string
  extension: string;                    // File extension filter (optional)
  caseSensitive: boolean;               // Whether search should be case sensitive
  useRegex: boolean;                    // Whether to treat query as regex
  includeBinary: boolean;               // Whether to include binary files in search
  maxFileSize: number;                  // Max file size in bytes (10MB default)
  maxResults: number;                   // Max number of results (1000 default)
  searchSubdirs: boolean;               // Whether to search subdirectories
  resultText: string;                   // Status text
  searchResults: SearchResult[];        // Search results array
  truncatedResults: boolean;            // Whether results were truncated (due to limit)
  isSearching: boolean;                 // Whether a search is currently in progress
  searchProgress: SearchProgress;       // Progress information
  showProgress: boolean;                // Whether to show progress bar
  minFileSize: number;                  // Minimum file size filter (bytes)
  excludePatterns: string[];            // Array of patterns to exclude (e.g., ["node_modules","*.log"])
  allowedFileTypes: string[];           // Array of file extensions that are allowed (empty means all allowed)
  recentSearches: Array<{              // Recent searches history
    query: string;
    extension: string;
  }>;
  error: string | null;                 // Error message if any
}
```

### Frontend Performance Optimizations

#### Code Splitting and Dynamic Imports
- **Syntax Highlighting**: Dynamically imports highlight.js only when needed
- **Component Loading**: Splits large components for faster initial loading
- **Bundle Optimization**: Reduces initial bundle size by lazy-loading features

#### Efficient Rendering
- **Pagination**: Limits DOM elements by showing results in pages (10 per page)
- **Virtualization**: Optimizes rendering for large result sets
- **Memory Management**: Truncates large files to prevent browser crashes (10,000 lines max)

#### Async Operations
- **Non-blocking UI**: Maintains responsive UI during search operations
- **Loading States**: Provides clear loading indicators
- **Progress Updates**: Real-time progress visualization

### Security Considerations in Frontend

#### Input Sanitization
- **Path Validation**: Validates file paths to prevent directory traversal
- **HTML Sanitization**: Sanitizes content before rendering to prevent XSS
- **Regex Validation**: Validates regex patterns before highlighting

#### Content Security
- **Trusted Types**: Ensures only safe content is rendered
- **CSP Compliance**: Follows Content Security Policy best practices

## Communication Layer (Wails)

### Architecture Overview
The Wails framework provides a robust communication layer between Go and Vue.js:

- **Type Safety**: Generated TypeScript bindings ensure type-safe communication
- **Real-time Events**: Efficient progress updates without blocking operations
- **Cross-Platform Compatibility**: Native system integration across all platforms
- **Performance**: Optimized for low-latency communication

### Generated Bindings
Wails automatically generates TypeScript bindings for all exported Go functions:

- **Go Functions**: All `App` methods with proper Wails tags become available in frontend
- **TypeScript Interfaces**: Generated based on Go struct definitions
- **Error Handling**: Proper error propagation from Go to TypeScript
- **Async Operations**: All backend calls are asynchronous

### Event System
The real-time event system enables efficient progress reporting:

- **Progress Updates**: Search progress communicated via "search-progress" events
- **Event Cleanup**: Proper cleanup to prevent memory leaks
- **Error Events**: Specialized events for error conditions and cancellation

## Performance Optimizations

### Backend Optimizations

#### Parallel Processing Architecture
- **Worker Pool Pattern**: Dynamically sized based on CPU cores
- **Efficient Scheduling**: Load balancing across available goroutines
- **Resource Management**: Prevents resource exhaustion during large searches

#### Memory Management
- **Streaming for Large Files**: Processes files line-by-line to prevent memory issues
- **Buffer Management**: Configurable buffer sizes (default 1MB) for optimal performance
- **Early Termination**: Cancels operations when max results reached

#### File System Optimization
- **Size-Based Filtering**: Excludes large files before processing
- **Binary Detection**: Skips binary files when not required
- **File Type Filtering**: Allows/disallows specific extensions efficiently

#### Context-Aware Operations
- **Cancellation Support**: Uses Go contexts for clean operation termination
- **Timeout Handling**: Prevents operations from running indefinitely
- **Resource Cleanup**: Proper cleanup of file handles and memory

### Frontend Optimizations

#### Rendering Performance
- **Virtual Scrolling**: Efficient rendering of large result sets
- **Progressive Loading**: Results loaded in batches for better UX
- **Memory Management**: Limits DOM elements to prevent performance issues

#### Bundle Optimization
- **Code Splitting**: Critical and non-critical code separated
- **Tree Shaking**: Unused code eliminated from bundles
- **Dynamic Imports**: Features loaded on-demand

#### User Experience Optimizations
- **Progress Visualization**: Real-time feedback keeps users informed
- **Responsive Design**: Adapts to different screen sizes
- **Accessibility**: Proper ARIA attributes and keyboard navigation

## Security Considerations

### Input Validation and Sanitization

#### Directory and File Path Security
- **Path Traversal Prevention**: Uses filepath.Clean to prevent `../` attacks
- **Validation Checks**: Ensures paths are within expected scope before operations
- **Sanitization**: Cleans all file paths before system operations

#### Search Query Security
- **Pattern Validation**: Validates regex patterns before execution
- **Injection Prevention**: Sanitizes search queries to prevent injection attacks
- **Size Limits**: Prevents denial-of-service through overly large queries

### File System Security

#### Access Control
- **File Type Allow-Lists**: Restricts searches to specific file extensions
- **Binary File Handling**: Prevents processing of binary files when inappropriate
- **Permission Checks**: Verifies file access permissions before operations

#### Security Measures
- **Read-Only Operations**: No write operations performed during searches
- **Isolation**: Search operations are isolated to specified directories
- **Validation**: All file operations validated before execution

### Frontend Security

#### Content Security
- **XSS Prevention**: Sanitizes all content before rendering
- **CSP Implementation**: Content Security Policy to prevent injection attacks
- **Trusted Input**: Only renders trusted content from secure sources

#### Data Security
- **Local Storage**: Secure storage of recent searches with validation
- **Session Management**: Proper cleanup of temporary data
- **Privacy**: No data transmitted to external services

## Testing Strategy

### Backend Testing

#### Unit Tests
- **Function-Level Testing**: Each function tested in isolation
- **Edge Cases**: Comprehensive testing of boundary conditions
- **Error Handling**: Verification of error paths and recovery
- **Security Tests**: Validation of security measures (path traversal, etc.)

#### Integration Tests
- **Full Search Workflows**: End-to-end testing of search functionality
- **Cross-Platform Integration**: Verification of platform-specific behavior
- **Performance Testing**: Validation of performance optimizations
- **Large File Handling**: Testing of streaming and memory management

#### Security Testing
- **Path Traversal Tests**: Verification of path validation measures
- **Injection Attacks**: Testing for potential injection vulnerabilities
- **Binary File Handling**: Verification of binary detection and exclusion
- **Access Control**: Testing of file type allow-lists

### Frontend Testing

#### Component Tests
- **Unit Testing**: Individual components tested in isolation
- **Integration Testing**: Component interactions and communications
- **State Management**: Verification of reactive state behavior
- **Event Handling**: Testing of UI events and responses

#### End-to-End Testing
- **User Workflows**: Full user interaction paths
- **Search Scenarios**: Various search configurations and parameters
- **Error Handling**: Frontend response to backend errors
- **Performance**: Verification of UI performance under load

#### Mock Testing
- **Wails Bindings**: Mocked backend functions for isolated testing
- **Event Simulation**: Simulated progress and error events
- **State Transitions**: Testing of different UI states

### Performance Testing

#### Load Testing
- **Large Codebases**: Testing with large file trees and many files
- **Concurrent Searches**: Multiple simultaneous search operations
- **Memory Usage**: Monitoring memory consumption during operations
- **CPU Utilization**: Verification of efficient resource usage

#### Stress Testing
- **Maximum Limits**: Testing at maximum file sizes and result counts
- **Invalid Inputs**: Handling of malformed or malicious inputs
- **Resource Exhaustion**: Testing under resource-constrained conditions

### Automated Testing Pipeline

#### Continuous Integration
- **Unit Test Execution**: All tests run on every commit
- **Integration Verification**: Cross-platform testing in CI environment
- **Performance Baselines**: Verification of performance metrics
- **Security Scanning**: Automated security vulnerability detection

## Key Features Deep Dive

### Advanced Search Capabilities

#### Regular Expression Support
- **Pattern Validation**: Real-time validation of regex patterns
- **Performance Optimization**: Efficient regex execution with timeouts
- **Case Sensitivity**: Configurable case sensitivity options
- **Feature Completeness**: Full regex feature support with Go's regex engine

#### Multi-Part Extension Support
- **Complex Extensions**: Support for extensions like `.tar.gz`, `.min.js`, etc.
- **Flexible Matching**: Both final and full extension matching available
- **Performance**: Optimized extension matching algorithms
- **User Experience**: Intuitive extension selection interface

#### Exclude Pattern Feature
- **Pattern Matching**: Glob pattern support (e.g., `node_modules/*`)
- **Multiple Exclusions**: Support for multiple exclude patterns simultaneously
- **Performance**: Efficient pattern matching during file traversal
- **User Interface**: Multi-select dropdown for common patterns

### User Experience Features

#### Real-Time Progress Tracking
- **Detailed Metrics**: File counts, processing speed, and time estimates
- **Visual Indicators**: Progress bars and status messages
- **Performance**: Efficient progress updates without blocking
- **User Control**: Clear feedback throughout the process

#### Code Preview Modal
- **Syntax Highlighting**: Language-aware highlighting with highlight.js
- **Navigation**: Ability to navigate between search matches in file
- **Performance**: Truncation limits to prevent performance issues
- **Readability**: Optimized display with line numbers and context

#### Recent Searches
- **Persistence**: Local storage with expiration and cleanup
- **Performance**: Efficient storage and retrieval mechanisms
- **User Experience**: Quick access to previous search parameters
- **Privacy**: No external data transmission

### Performance Optimization Features

#### Large File Handling
- **Streaming Architecture**: Line-by-line processing for files >1MB
- **Memory Management**: Configurable buffer sizes and memory limits
- **Performance**: Optimized I/O operations for large files
- **User Experience**: Continues operation without freezing UI

#### Result Limiting
- **Configurable Limits**: User-adjustable result count limits
- **Early Termination**: Efficient search cancellation when limit reached
- **Performance**: Prevents overwhelming result sets
- **User Control**: Clear indication when results are limited

### Security Features

#### File Type Allow-Lists
- **Granular Control**: Selective file type restriction
- **Security**: Prevents searching of potentially dangerous file types
- **Performance**: Improves search performance by reducing file types
- **User Experience**: Intuitive interface for selection

#### Binary File Detection
- **Multiple Checks**: Multiple validation methods for binary detection
- **Performance**: Efficient detection without full file loading
- **Security**: Prevents processing of non-text files
- **Accuracy**: High accuracy binary detection algorithms

## Development Workflow

### Setup and Configuration

#### Prerequisites
- **Go Environment**: Go 1.23+ with proper GOPATH configuration
- **Node.js Environment**: Node.js 16.x+ with npm/yarn
- **Wails CLI**: Properly installed and configured Wails
- **System Dependencies**: Platform-specific tools for directory selection

#### Project Initialization
1. **Repository Setup**: Clone and initialize the repository
2. **Dependency Management**: Install Go modules and NPM packages
3. **Development Server**: Start Wails development server with `wails dev`
4. **Testing Environment**: Set up testing with proper mocks

### Development Process

#### Backend Development
- **Go Modules**: Proper dependency management with go.mod
- **Testing**: Run tests with `go test -v ./...`
- **Formatting**: Use `go fmt` for consistent code style
- **Documentation**: Add godoc comments for exported functions

#### Frontend Development
- **TypeScript**: Strong typing throughout the codebase
- **Vue 3 Composition API**: Modern Vue.js patterns
- **Component Testing**: Unit tests for all components
- **Performance**: Monitor bundle sizes and load times

#### Testing Strategy
1. **Unit Tests**: All functions and components tested in isolation
2. **Integration Tests**: End-to-end workflows validated
3. **Performance Tests**: Speed and memory usage verified
4. **Security Tests**: Vulnerability and penetration testing

### Build and Deployment

#### Development Builds
- **Hot Reloading**: Use `wails dev` for development
- **Real-time Updates**: Changes reflected immediately in UI
- **Debugging**: Full debugging capabilities with source maps

#### Production Builds
- **Optimization**: Code splitting and minification applied
- **Cross-Platform**: Builds for Windows, Linux, and macOS
- **Performance**: Optimized for production performance
- **Security**: Production security configurations applied

### Best Practices

#### Code Quality
- **Consistency**: Follow project coding standards
- **Documentation**: Comprehensive documentation for all features
- **Testing**: High test coverage with meaningful tests
- **Review**: Code reviews for all changes

#### Performance
- **Efficiency**: Optimize for both speed and resource usage
- **Scalability**: Design for large codebases and complex searches
- **Memory**: Monitor and optimize memory usage
- **User Experience**: Prioritize responsive UI and clear feedback

#### Security
- **Validation**: Validate all inputs and sanitize all outputs
- **Principle of Least Privilege**: Minimal required permissions
- **Protection**: Implement multiple layers of security
- **Updates**: Regular security updates and patches

## Troubleshooting and Best Practices

### Common Issues and Solutions

#### Search Performance Issues
- **Large Files**: Check `Max File Size` settings to exclude very large files
- **Many Results**: Use `Max Results` limit to prevent overwhelming results
- **Complex Regex**: Simplify regex patterns that may be inefficient
- **System Resources**: Monitor CPU and memory usage during searches

#### Directory Selection Problems
- **Linux Systems**: Ensure zenity, kdialog, or yad are installed
- **Windows Systems**: PowerShell must be available (included with Windows 7+)
- **macOS Systems**: AppleScript integration is planned for future release
- **Permissions**: Verify directory has proper read permissions

#### UI/UX Issues
- **Large Results**: Use pagination to handle many results efficiently
- **Slow Rendering**: Implement virtual scrolling for large result sets
- **Memory Issues**: Monitor and optimize memory usage for large files
- **Responsive Design**: Test across different screen sizes and devices

### Development Best Practices

#### Go Backend Best Practices
- **Context Usage**: Properly use context for cancellation and timeouts
- **Error Handling**: Comprehensive error handling with graceful degradation
- **Memory Management**: Efficient use of memory with streaming for large files
- **Concurrency**: Proper use of goroutines and channels for parallel processing

#### Vue.js Frontend Best Practices
- **Composition API**: Use composables for shared logic
- **Reactivity**: Efficient use of Vue's reactivity system
- **Performance**: Optimize rendering and component updates
- **Type Safety**: Use TypeScript for comprehensive type checking

#### Wails Framework Best Practices
- **Bindings**: Properly structure Go functions for Wails integration
- **Events**: Use events for real-time communication efficiently
- **Security**: Follow Wails security guidelines
- **Cross-Platform**: Test functionality across all supported platforms

### Performance Tuning

#### Backend Tuning
- **Worker Count**: Adjust worker pool size based on system capabilities
- **Buffer Sizes**: Optimize buffer sizes for different file types
- **Caching**: Implement caching for frequently accessed data
- **Resource Limits**: Configure appropriate limits for file sizes and counts

#### Frontend Tuning
- **Bundle Size**: Optimize bundle size through code splitting
- **Rendering**: Optimize rendering through virtualization and pagination
- **Memory**: Monitor memory usage during large operations
- **API Calls**: Minimize and optimize backend API calls

### Security Guidelines

#### Input Validation
- **All Inputs**: Validate all user inputs on both frontend and backend
- **File Paths**: Sanitize all file paths to prevent traversal attacks
- **Search Queries**: Validate and sanitize search patterns
- **Configuration**: Validate all configuration parameters

#### Secure Communication
- **Data Integrity**: Ensure data integrity between frontend and backend
- **Privacy**: Protect user data and privacy in all operations
- **System Access**: Limit system access to necessary operations only
- **Logging**: Secure logging practices without sensitive information