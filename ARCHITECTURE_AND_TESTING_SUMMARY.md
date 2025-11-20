# Code-Search-Golang Application Architecture & Testing Strategy

## Executive Summary

The code-search-golang application is a powerful, feature-rich desktop code search tool built with Wails (Go backend + Vue.js frontend). It enables users to search for text patterns, keywords, and regular expressions across code files in specified directories with advanced features like regex search, configurable file filtering, security-hardened file operations, exclude patterns, and pagination for better user experience. The application includes extensive performance optimizations, security hardening, and a modern, responsive UI with syntax highlighting.

This document provides a comprehensive overview of the application architecture, implementation details, testing strategy, and development best practices, serving as a reference for current and future development efforts.

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
The application follows these core architectural principles to ensure maintainability, performance, and security:

- **Separation of Concerns**: Clear boundaries between frontend presentation logic, backend business logic, and communication layers to promote modularity and maintainability
- **Performance First**: Optimized for handling large codebases with parallel processing and streaming to deliver fast search results even with extensive file trees
- **Security by Design**: Built-in protections against file system traversal and malicious inputs to ensure safe file system operations
- **Cross-Platform Compatibility**: Native experience across Windows, Linux, and macOS through platform-specific system integration
- **Scalability**: Designed to handle large file trees and complex search operations efficiently through intelligent resource management and parallel processing
- **Real-time Communication**: Dual communication channels (Wails events and HTTP polling) for different types of data flow and user interaction
- **Resource Management**: Proper context cancellation, memory management, and resource cleanup to prevent system resource exhaustion
- **Type Safety**: End-to-end type safety from Go backend structures to TypeScript frontend components through Wails bindings

### High-Level Architecture
The application follows a client-server architecture with the Vue.js frontend as the client and Go backend as the server, connected through Wails' binding system:

```
┌─────────────────┐    Wails Bindings    ┌─────────────────┐
│   Vue.js 3      │ ←──────────────────→ │      Go         │
│ Frontend Layer  │                      │  Backend Layer  │
│ (UI/UX Logic)   │                      │ (Search Engine) │
└─────────────────┘                      └─────────────────┘
         │                                          │
         │ HTTP Polling Communications              │ File System
         │ (Log Streaming, Real-time Updates)       │ Operations
         ↓                                          ↓
┌─────────────────┐                    ┌─────────────────────┐
│ HTTP Polling    │                    │ File System         │
│ Server (Port    │                    │ & Process Management│
│ 34116)          │                    │                     │
└─────────────────┘                    └─────────────────────┘

         │
         │ Event System
         │ (Progress, Status, Errors)
         ↓
┌─────────────────┐
│ Real-time UI    │
│ Updates         │
└─────────────────┘
```

The architecture enables efficient communication between the user interface and file system operations while maintaining platform-specific system integration capabilities. The dual communication approach (Wails bindings for direct function calls and HTTP polling for real-time streaming) provides both immediate responses and continuous updates.

### Communication Architecture
The application implements a sophisticated dual-channel communication system:

1. **Wails Binding Channel** (Primary):
   - Direct function calls from frontend to Go backend
   - Type-safe TypeScript bindings generated from Go functions
   - Real-time progress events for search operations
   - Synchronous and asynchronous operations as needed

2. **HTTP Polling Channel** (Secondary):
   - Real-time log streaming via file tailing
   - Live search progress and result streaming
   - Separate HTTP server on port 34116
   - Polling endpoints for clients to fetch updates

This dual-channel approach ensures optimal performance and user experience by using the appropriate communication method for each type of data flow.

## Backend Architecture (Go)

### Core Components

#### App Structure
```go
type App struct {
    ctx          context.Context
    searchCancel context.CancelFunc // Cancel function for active searches
}
```

The `App` struct serves as the main entry point for all backend functionality, managing context for communication with the frontend and providing control over search operations. It maintains application state and lifecycle management, ensuring proper resource cleanup and operation cancellation when needed.

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

The `SearchRequest` structure encapsulates all parameters needed for search operations, providing flexibility and extensibility for various search scenarios. It includes security features like `AllowedFileTypes` to restrict file types that can be searched, and performance controls like file size limits to prevent resource exhaustion.

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

The `SearchResult` structure provides comprehensive information about each match, including context lines for better understanding of the search results. The inclusion of context lines (`ContextBefore` and `ContextAfter`) helps users understand the matched text within its surrounding code, making it easier to evaluate the relevance of each result.

### Key Backend Functions

#### SearchWithProgress
The core search functionality with real-time progress updates, representing the heart of the application's search engine:

- **Parallel Processing**: Uses Go goroutines with worker pools to process multiple files simultaneously, dynamically sized based on available CPU cores to maximize performance
- **Memory Efficiency**: Implements streaming for large files (>1MB) to prevent memory issues by processing files line-by-line instead of loading entire files into memory
- **Early Termination**: Uses context cancellation to stop search when max results are reached, saving computational resources and providing faster response times
- **Progress Tracking**: Emits real-time updates to the frontend during the search process, including file counts, progress percentage, and estimated time remaining
- **Error Handling**: Comprehensive error management with graceful degradation, ensuring searches continue despite individual file access errors
- **Worker Pool Management**: Dynamic worker allocation based on file count and system resources for optimal performance
- **Atomic Counter Management**: Thread-safe progress tracking using atomic operations to prevent race conditions

#### processFileLineByLine
Memory-efficient file processing for large files, implementing streaming techniques to handle files that exceed available memory:

- **Streaming Approach**: Reads and processes files line-by-line to avoid memory issues, using buffered readers that maintain optimal memory usage
- **Binary Detection**: Checks for binary content without loading entire files into memory, identifying non-text files early in the process
- **Context Cancellation**: Respects search cancellation during line-by-line processing, allowing searches to be stopped immediately when needed
- **Performance Optimization**: Efficient processing with configurable buffer sizes to balance memory usage with I/O performance
- **Progress Updates**: Periodic context checks during processing to maintain responsiveness during long file reads
- **Resource Cleanup**: Proper file handle management with deferred closures to prevent resource leaks

#### isBinary
Binary file detection with multiple validation layers to prevent processing of non-text files:

- **Null Byte Detection**: Identifies binary files containing null bytes, which are uncommon in text files
- **Printable Character Analysis**: Evaluates the ratio of printable vs. non-printable characters to distinguish between text and binary content
- **Size-Based Detection**: Analyzes only the first 512 bytes for performance, as binary content is often present in file headers
- **Configurable Thresholds**: Allows adjustment of binary detection sensitivity to handle edge cases where files contain mixed content
- **UTF-8 Tolerance**: Properly handles UTF-8 encoded text files with high-byte values to avoid false positives

#### PollingLogManager
HTTP-based log polling system for log updates:

- **Separate HTTP Server**: Runs on port 34116 (next to Wails default port 34115) to avoid conflicts and provide dedicated communication channel
- **File Tailing**: Uses nxadm/tail library to track log file updates and store them in memory for polling
- **Polling Endpoints**: Provides `/poll` endpoint for new log entries and `/initial` endpoint for initial log set
- **Thread-Safe Storage**: Concurrent access to log entries using RWMutex to ensure data consistency
- **Message Serialization**: JSON-based message formatting with LogMessage structure for consistent data transmission
- **Memory Management**: Limits log storage size to prevent memory bloat with sliding window approach
- **Error Handling**: Graceful degradation with proper HTTP error codes when endpoints fail
- **Resource Management**: Proper cleanup of HTTP server and tailing resources to prevent memory leaks

### Cross-Platform System Integration

#### SelectDirectory
Handles native directory selection across platforms, ensuring users have a familiar experience regardless of their operating system:

- **Windows**: Uses PowerShell with System.Windows.Forms for native experience, leveraging Windows' built-in folder selection dialog
- **Linux**: Tries multiple options in order of preference (zenity, kdialog, yad) to accommodate different desktop environments
- **macOS**: Uses AppleScript (implementation in separate file) to access macOS Finder's built-in directory selection
- **Error Handling**: Comprehensive error management for unavailable system tools, gracefully falling back or providing clear error messages

#### ShowInFolder
Secure file manager integration with path traversal protection, allowing users to quickly navigate to files in their system's file manager:

- **Path Sanitization**: Uses filepath.Clean to prevent directory traversal attacks, ensuring file paths remain within intended directories
- **Validation Checks**: Ensures directory exists and is accessible before opening, preventing errors when files are moved or deleted
- **Cross-Platform Commands**: Uses appropriate OS commands (xdg-open, cmd /c start, open) to open the correct file manager
- **Security Validation**: Prevents access to parent directories via traversal attempts, maintaining system security

## Frontend Architecture (Vue.js)

### Component Architecture

#### CodeSearch.vue (Main Component)
- **Single Responsibility**: Orchestrates the entire search workflow by coordinating between UI components and business logic
- **Composable Integration**: Uses `useSearch` composable for all business logic, following Vue 3's composition API patterns
- **UI Composition**: Aggregates child components (SearchForm, ProgressIndicator, SearchResults) into a cohesive user interface

#### SearchForm.vue
- **User Input Management**: Handles all search parameters and options with proper input validation and user feedback
- **Validation**: Client-side validation before backend calls to prevent invalid searches and provide immediate feedback
- **Recent Searches**: Integrates with localStorage for search history, improving user workflow and productivity

#### SearchResults.vue
- **Result Presentation**: Displays search results with pagination for optimal performance and user experience
- **Interactive Features**: Provides copy, open folder, and preview functionality to enable efficient code navigation
- **Performance**: Implements pagination to handle large result sets, preventing UI slowdowns and browser memory issues

#### ProgressIndicator.vue
- **Real-time Updates**: Displays search progress with detailed metrics including file counts, percentage complete, and time estimates
- **Visual Feedback**: Provides clear indication of search status with visual progress bars and status messages

#### CodeModal.vue
- **Code Preview**: Displays file content with syntax highlighting using highlight.js for enhanced readability
- **Navigation**: Allows navigation between search matches within the same file for efficient code review
- **Performance**: Implements truncation limits to prevent performance issues when viewing extremely large files

### Composable Pattern

#### useSearch (Composition Composable)
The `useSearch` composable centralizes all search business logic, following Vue 3's composition API best practices:

- **State Management**: Manages all reactive state for the search functionality with proper TypeScript typing
- **Backend Integration**: Handles all communication with Go backend through Wails bindings with error handling
- **Event Handling**: Manages progress updates and search lifecycle using Wails' real-time event system
- **Persistence**: Handles localStorage operations for recent searches with proper data validation and cleanup
- **Error Handling**: Provides comprehensive error management and user feedback with appropriate error messages

#### Reactive State Structure
The composable manages the following reactive state with full TypeScript type safety:

```typescript
interface SearchState {
  directory: string;                    // Directory path to search in - validated for security
  query: string;                        // Search query string - sanitized before transmission
  extension: string;                    // File extension filter (optional) - supports multi-part extensions
  caseSensitive: boolean;               // Whether search should be case sensitive for precise matching
  useRegex: boolean;                    // Whether to treat query as regex - with validation for pattern safety
  includeBinary: boolean;               // Whether to include binary files in search - with detection safeguards
  maxFileSize: number;                  // Max file size in bytes (10MB default) to prevent memory issues
  maxResults: number;                   // Max number of results (1000 default) for UI performance
  searchSubdirs: boolean;               // Whether to search subdirectories for comprehensive coverage
  resultText: string;                   // Status text - provides clear feedback to users
  searchResults: SearchResult[];        // Search results array - paginated for optimal performance
  truncatedResults: boolean;            // Whether results were truncated (due to limit) for user awareness
  isSearching: boolean;                 // Whether a search is currently in progress - controls UI state
  searchProgress: SearchProgress;       // Progress information - with detailed metrics and time estimates
  showProgress: boolean;                // Whether to show progress bar - based on search duration
  minFileSize: number;                  // Minimum file size filter (bytes) - to exclude tiny files
  excludePatterns: string[];            // Array of patterns to exclude (e.g., ["node_modules","*.log"]) for intelligent filtering
  allowedFileTypes: string[];           // Array of file extensions that are allowed (empty means all allowed) for security
  recentSearches: Array<{              // Recent searches history - with automatic cleanup and deduplication
    query: string;
    extension: string;
  }>;
  error: string | null;                 // Error message if any - with user-friendly formatting
}
```

### Frontend Performance Optimizations

#### Code Splitting and Dynamic Imports
- **Syntax Highlighting**: Dynamically imports highlight.js only when needed to reduce initial bundle size and memory usage
- **Component Loading**: Splits large components for faster initial loading, allowing critical functionality to load first
- **Bundle Optimization**: Reduces initial bundle size by lazy-loading features that are not immediately required
- **Editor Detection**: Asynchronous editor detection with progress updates to avoid blocking the main UI thread
- **HTTP Polling Integration**: Separate HTTP server handling for real-time log streaming without affecting main search operations

#### Efficient Rendering
- **Pagination**: Limits DOM elements by showing results in pages (10 per page) to maintain smooth UI performance
- **Virtualization**: Optimizes rendering for large result sets by only rendering visible elements and recycling DOM nodes
- **Memory Management**: Truncates large files to prevent browser crashes (10,000 lines max) while maintaining usability
- **Progress Visualization**: Efficient progress bar updates with throttling to prevent excessive DOM updates
- **Search Result Caching**: Caches recent search results to improve responsiveness when navigating between results

#### Async Operations
- **Non-blocking UI**: Maintains responsive UI during search operations by using asynchronous patterns throughout
- **Loading States**: Provides clear loading indicators to keep users informed about ongoing operations
- **Progress Updates**: Real-time progress visualization using Wails' event system for immediate feedback
- **Event Management**: Proper cleanup of Wails event listeners to prevent memory leaks and performance degradation
- **HTTP Polling Handling**: Separate HTTP polling mechanism without blocking main UI thread

### Security Considerations in Frontend

#### Input Sanitization
- **Path Validation**: Validates file paths to prevent directory traversal attacks before sending to backend
- **HTML Sanitization**: Sanitizes content before rendering to prevent XSS vulnerabilities when displaying search results
- **Regex Validation**: Validates regex patterns before highlighting to prevent potentially malicious patterns
- **Content Filtering**: Implements DOMPurify for additional content sanitization when rendering search results
- **File Path Handling**: Proper validation of file paths before showing file locations to prevent path injection

#### Content Security
- **Trusted Types**: Ensures only safe content is rendered by implementing proper content security measures
- **CSP Compliance**: Follows Content Security Policy best practices to prevent injection attacks and unauthorized resource loading
- **Event Handling**: Secure handling of Wails events with proper validation to prevent malicious data injection
- **Local Storage Protection**: Secure storage of recent searches with validation to prevent malicious injection
- **HTTP Polling Security**: Secure HTTP polling communication with origin validation and message sanitization

## Communication Layer (Wails)

### Architecture Overview
The Wails framework provides a robust communication layer between Go and Vue.js, serving as the bridge that enables secure and efficient interaction between the frontend UI and backend file system operations:

- **Type Safety**: Generated TypeScript bindings ensure type-safe communication, preventing runtime errors and improving development experience
- **Real-time Events**: Efficient progress updates without blocking operations, enabling smooth user experience during long-running searches
- **Cross-Platform Compatibility**: Native system integration across all platforms through Wails' platform abstraction layer
- **Performance**: Optimized for low-latency communication with efficient data serialization between Go and JavaScript
- **Dual Channel Architecture**: Wails bindings for direct Go-Vue communication and HTTP polling for real-time streaming
- **Event Management**: Comprehensive event system with proper cleanup and resource management to prevent memory leaks
- **Security**: Built-in protection against injection attacks and unauthorized system access through proper data validation

### Generated Bindings
Wails automatically generates TypeScript bindings for all exported Go functions, creating a seamless interface between the two languages:

- **Go Functions**: All `App` methods with proper Wails tags become available in frontend with full TypeScript type safety
- **TypeScript Interfaces**: Generated based on Go struct definitions, maintaining type consistency across the application
- **Error Handling**: Proper error propagation from Go to TypeScript with detailed error messages for debugging
- **Async Operations**: All backend calls are asynchronous by default, preventing UI blocking during file operations
- **Structure Mapping**: Direct TypeScript interface generation from Go structs ensuring data consistency
- **Method Validation**: Automatic validation of parameter types and return values to prevent runtime errors
- **Debugging Support**: Generated bindings include debugging information and error context for easier troubleshooting

### Event System
The real-time event system enables efficient progress reporting and maintains responsive communication during search operations:

- **Progress Updates**: Search progress communicated via "search-progress" events with detailed metrics including file counts and time estimates
- **Event Cleanup**: Proper cleanup to prevent memory leaks through proper event listener management and resource disposal
- **Error Events**: Specialized events for error conditions and cancellation, ensuring users receive appropriate feedback for all scenarios
- **Editor Detection Events**: Specialized event system for tracking editor availability detection progress and status
- **Event Throttling**: Intelligent event rate limiting to prevent UI flooding during rapid status changes
- **Connection Validation**: Event system validates connections before sending to prevent errors in disconnected states

### HTTP Polling Integration
The application implements an HTTP-based polling system for real-time log streaming and progress updates as a replacement for WebSocket communication. This approach provides a simpler, more reliable communication mechanism that works consistently across different network environments and platforms without the complexity of managing WebSocket connections.

- **Dedicated Server**: Separate HTTP server running on port 34116 to avoid conflicts with main Wails application (next to default Wails port 34115)
- **File Tail Integration**: Real-time log tailing using nxadm/tail library to monitor log file changes and store them in memory for polling
- **Polling Endpoints**: Provides `/poll` endpoint for retrieving new log entries since last poll and `/initial` endpoint for fetching initial log set
- **Thread-Safe Storage**: Concurrent access to log entries using RWMutex to ensure data consistency during simultaneous read/write operations
- **Message Serialization**: JSON-based message formatting with LogMessage structure (Type and Content fields) for consistent data transmission
- **Search Progress Streaming**: Dedicated polling endpoints for real-time search progress and result updates allowing clients to fetch updates at regular intervals
- **Resource Management**: Proper cleanup of HTTP polling resources to prevent memory leaks with sliding window approach that maintains last 750 entries while removing older ones
- **Cross-Platform Compatibility**: HTTP polling communication works consistently across Windows, Linux, and macOS with standard HTTP requests
- **Memory Optimization**: Efficient memory management through index tracking and array rotation to handle continuous log streaming without memory bloat
- **CORS Support**: Proper CORS headers allowing cross-origin requests from the frontend application
- **Request Filtering**: Intelligent filtering of log entries to skip verbose messages like "Skipping" or "Sending file" that don't provide value to users

## Performance Optimizations

### Backend Optimizations

#### Parallel Processing Architecture
- **Worker Pool Pattern**: Dynamically sized based on CPU cores to maximize search throughput while preventing system overload
- **Efficient Scheduling**: Load balancing across available goroutines to optimize resource utilization and search performance
- **Resource Management**: Prevents resource exhaustion during large searches by limiting concurrent operations based on system capabilities

#### Memory Management
- **Streaming for Large Files**: Processes files line-by-line to prevent memory issues, maintaining consistent memory usage regardless of file size
- **Buffer Management**: Configurable buffer sizes (default 1MB) for optimal performance balancing memory usage with I/O efficiency
- **Early Termination**: Cancels operations when max results reached, saving computational resources and providing faster response times

#### File System Optimization
- **Size-Based Filtering**: Excludes large files before processing to avoid unnecessary I/O operations and memory allocation
- **Binary Detection**: Skips binary files when not required, reducing processing time and preventing errors from non-text content
- **File Type Filtering**: Allows/disallows specific extensions efficiently using pattern matching for improved search relevance

#### Context-Aware Operations
- **Cancellation Support**: Uses Go contexts for clean operation termination, allowing users to stop long-running searches immediately
- **Timeout Handling**: Prevents operations from running indefinitely through configurable context timeouts and cancellation signals
- **Resource Cleanup**: Proper cleanup of file handles and memory through deferred functions and context cancellation

### Frontend Optimizations

#### Rendering Performance
- **Virtual Scrolling**: Efficient rendering of large result sets by only rendering visible elements in the viewport
- **Progressive Loading**: Results loaded in batches for better UX, preventing UI freezes during result display
- **Memory Management**: Limits DOM elements to prevent performance issues and browser memory exhaustion

#### Bundle Optimization
- **Code Splitting**: Critical and non-critical code separated to reduce initial load time and improve perceived performance
- **Tree Shaking**: Unused code eliminated from bundles through webpack's dead code elimination during build process
- **Dynamic Imports**: Features loaded on-demand when needed, rather than at application startup, reducing initial bundle size

#### User Experience Optimizations
- **Progress Visualization**: Real-time feedback keeps users informed about search status, file counts, and estimated completion time
- **Responsive Design**: Adapts to different screen sizes ensuring consistent experience across desktop and mobile devices
- **Accessibility**: Proper ARIA attributes and keyboard navigation supporting users with different accessibility requirements

## Security Considerations

### Input Validation and Sanitization

#### Directory and File Path Security
- **Path Traversal Prevention**: Uses filepath.Clean to prevent `../` attacks that could access unauthorized directories
- **Validation Checks**: Ensures paths are within expected scope before operations, verifying directory existence and accessibility
- **Sanitization**: Cleans all file paths before system operations to remove potentially dangerous characters or sequences
- **Multi-Layer Validation**: Implements multiple validation checkpoints during path processing to catch traversal attempts
- **Absolute Path Verification**: Converts to absolute paths and validates against base directory to ensure containment
- **Character Filtering**: Blocks potentially dangerous characters and sequences that could be used for path manipulation

#### Search Query Security
- **Pattern Validation**: Validates regex patterns before execution to prevent catastrophic backtracking and resource exhaustion
- **Injection Prevention**: Sanitizes search queries to prevent injection attacks through special characters or sequences
- **Size Limits**: Prevents denial-of-service through overly large queries by implementing configurable size limits
- **Content Filtering**: Applies multiple layers of input validation to prevent malicious search patterns
- **Regex Complexity Limits**: Implements complexity checks to prevent resource-intensive regex patterns
- **Query Sanitization**: Strips potentially dangerous characters while preserving search functionality

### File System Security

#### Access Control
- **File Type Allow-Lists**: Restricts searches to specific file extensions, preventing access to sensitive system or configuration files
- **Binary File Handling**: Prevents processing of binary files when inappropriate to avoid potential security risks from non-text content
- **Permission Checks**: Verifies file access permissions before operations by respecting system-level file permissions
- **Protected Directory Blocking**: Automatically blocks critical system directories (like /, /usr, Windows C:\) to prevent system access
- **Size-Based Filtering**: Excludes extremely large files to prevent resource exhaustion attacks
- **Extension Validation**: Implements multi-part extension matching to prevent bypass of file type restrictions

#### Security Measures
- **Read-Only Operations**: No write operations performed during searches, preventing accidental or malicious file modification
- **Isolation**: Search operations are isolated to specified directories to prevent unauthorized file system access
- **Validation**: All file operations validated before execution to ensure they meet security requirements
- **Contextual Security**: Uses Go contexts for operation timeout and cancellation to prevent indefinite processing
- **Resource Limits**: Implements configurable limits on file size, result count, and processing time
- **Secure File Handling**: Proper file handle management with deferred closures to prevent resource leaks

#### Editor Integration Security
- **Safe Editor Launching**: Validates editor commands and file paths before launching external applications
- **Command Injection Prevention**: Properly sanitizes file paths for editor integration to prevent command injection
- **Available Editor Validation**: Verifies editor executables exist in system PATH before attempted launch
- **Path Sanitization**: Applies the same security measures to file paths opened in external editors

### Frontend Security

#### Content Security
- **XSS Prevention**: Sanitizes all content before rendering to prevent cross-site scripting attacks through malicious content injection
- **CSP Implementation**: Content Security Policy to prevent injection attacks by controlling which resources can be loaded and executed
- **Trusted Input**: Only renders trusted content from secure sources, validating all data before display to users
- **DOM Purification**: Uses DOMPurify library for additional content sanitization when displaying search results
- **Content Filtering**: Applies multiple validation layers before rendering any potentially unsafe content
- **HTML Escaping**: Properly escapes HTML content that could be misinterpreted as executable code

#### Data Security
- **Local Storage**: Secure storage of recent searches with validation to prevent malicious data injection in browser storage
- **Session Management**: Proper cleanup of temporary data and secure handling of session information between application runs
- **Privacy**: No data transmitted to external services, ensuring all search operations remain local to the user's system
- **State Validation**: Validates all state changes to prevent malicious manipulation of frontend state
- **Event Security**: Proper validation of all Wails events before state updates to prevent injection attacks
- **HTTP Polling Security**: Secure HTTP polling communication with origin validation and message sanitization

#### Communication Security
- **Wails Binding Security**: Secures all backend communication through Wails' type-safe binding system
- **Input Validation**: Client-side validation of all parameters before sending to backend
- **Output Sanitization**: Sanitizes all content received from backend before display or processing
- **Event Cleanup**: Proper cleanup of event listeners to prevent memory leaks and potential security vulnerabilities
- **Message Validation**: Validates all messages received through HTTP polling communication channels
- **Connection Security**: Implements security measures for HTTP polling connections to prevent unauthorized access

### Communication Layer Security

#### Wails Framework Security
- **Type Safety**: Generated TypeScript bindings ensure type-safe communication, preventing runtime errors and data corruption
- **Data Validation**: Automatic validation of parameter types and return values to prevent injection attacks
- **Secure Event System**: Validates all events and data before processing to prevent malicious data injection
- **Error Handling**: Secure error propagation without exposing sensitive system information
- **Method Access Control**: Limits backend method exposure to only necessary operations
- **Parameter Sanitization**: Automatic sanitization of all parameters passed between frontend and backend

#### HTTP Polling Security
- **Connection Validation**: Validates HTTP polling request origins to prevent unauthorized connections
- **Message Sanitization**: Sanitizes all messages before sending to requesting clients
- **Resource Management**: Proper cleanup of HTTP polling resources to prevent resource exhaustion
- **Polling Security**: Secure data retrieval with validation to prevent injection attacks
- **Log Streaming Security**: Secure file tailing with access validation to prevent unauthorized log access
- **Request Limiting**: Implements rate limiting to prevent HTTP polling-based denial-of-service attacks

## Testing Strategy

### Backend Testing

#### Unit Tests
- **Function-Level Testing**: Each function tested in isolation to ensure correct behavior and proper error handling
- **Edge Cases**: Comprehensive testing of boundary conditions and unusual inputs to prevent unexpected behavior
- **Error Handling**: Verification of error paths and recovery mechanisms to ensure graceful degradation
- **Security Tests**: Validation of security measures (path traversal, input sanitization, etc.) to prevent vulnerabilities

#### Integration Tests
- **Full Search Workflows**: End-to-end testing of search functionality including all components working together
- **Cross-Platform Integration**: Verification of platform-specific behavior to ensure consistent functionality across Windows, Linux, and macOS
- **Performance Testing**: Validation of performance optimizations to ensure they function as expected under various conditions
- **Large File Handling**: Testing of streaming and memory management to verify efficient processing of large files

#### Security Testing
- **Path Traversal Tests**: Verification of path validation measures to ensure users cannot access unauthorized directories
- **Injection Attacks**: Testing for potential injection vulnerabilities in search queries and file operations
- **Binary File Handling**: Verification of binary detection and exclusion to prevent processing of non-text files
- **Access Control**: Testing of file type allow-lists to ensure they properly restrict file access

### Frontend Testing

#### Component Tests
- **Unit Testing**: Individual components tested in isolation using Vue Test Utils and Jest to verify functionality
- **Integration Testing**: Component interactions and communications to ensure proper data flow and event handling
- **State Management**: Verification of reactive state behavior in the useSearch composable and dependent components
- **Event Handling**: Testing of UI events and responses to ensure proper user interaction handling

#### End-to-End Testing
- **User Workflows**: Full user interaction paths to verify complete search and navigation workflows function correctly
- **Search Scenarios**: Various search configurations and parameters to ensure all feature combinations work properly
- **Error Handling**: Frontend response to backend errors to verify proper error display and user guidance
- **Performance**: Verification of UI performance under load to ensure responsive behavior with large result sets

#### Mock Testing
- **Wails Bindings**: Mocked backend functions for isolated testing of frontend components without requiring backend
- **Event Simulation**: Simulated progress and error events to test real-time UI updates and state changes
- **State Transitions**: Testing of different UI states to ensure proper rendering and functionality in all scenarios

### Performance Testing

#### Load Testing
- **Large Codebases**: Testing with large file trees and many files to verify performance under realistic usage conditions
- **Concurrent Searches**: Multiple simultaneous search operations to ensure proper resource management and isolation
- **Memory Usage**: Monitoring memory consumption during operations to verify efficient resource usage and prevent leaks
- **CPU Utilization**: Verification of efficient resource usage to ensure searches don't monopolize system resources

#### Stress Testing
- **Maximum Limits**: Testing at maximum file sizes and result counts to verify system stability under extreme conditions
- **Invalid Inputs**: Handling of malformed or malicious inputs to ensure robust error handling and security
- **Resource Exhaustion**: Testing under resource-constrained conditions to verify graceful degradation behavior

### Automated Testing Pipeline

#### Continuous Integration
- **Unit Test Execution**: All tests run on every commit to catch regressions early in the development cycle
- **Integration Verification**: Cross-platform testing in CI environment to ensure consistent behavior across all supported platforms
- **Performance Baselines**: Verification of performance metrics to detect performance regressions before they're merged
- **Security Scanning**: Automated security vulnerability detection to identify potential security issues early

## Key Features Deep Dive

### Advanced Search Capabilities

#### Regular Expression Support
- **Pattern Validation**: Real-time validation of regex patterns to prevent catastrophic backtracking and security issues
- **Performance Optimization**: Efficient regex execution with configurable timeouts to prevent resource exhaustion
- **Case Sensitivity**: Configurable case sensitivity options for flexible matching requirements
- **Feature Completeness**: Full regex feature support with Go's powerful regex engine for complex pattern matching

#### Multi-Part Extension Support
- **Complex Extensions**: Support for extensions like `.tar.gz`, `.min.js`, `.config.bak`, etc. for comprehensive file filtering
- **Flexible Matching**: Both final and full extension matching available to accommodate different user needs
- **Performance**: Optimized extension matching algorithms that minimize processing overhead during file traversal
- **User Experience**: Intuitive extension selection interface that makes complex filtering accessible to all users

#### Exclude Pattern Feature
- **Pattern Matching**: Glob pattern support (e.g., `node_modules/*`, `*.log`) for flexible directory and file exclusion
- **Multiple Exclusions**: Support for multiple exclude patterns simultaneously to handle complex filtering requirements
- **Performance**: Efficient pattern matching during file traversal using optimized matching algorithms
- **User Interface**: Multi-select dropdown for common patterns with option for custom patterns for ease of use

### User Experience Features

#### Real-Time Progress Tracking
- **Detailed Metrics**: File counts, processing speed, and time estimates to keep users informed about search progress
- **Visual Indicators**: Progress bars and status messages providing clear visual feedback about operation status
- **Performance**: Efficient progress updates without blocking search operations to maintain optimal performance
- **User Control**: Clear feedback throughout the process allowing users to monitor and potentially cancel operations

#### Code Preview Modal
- **Syntax Highlighting**: Language-aware highlighting with highlight.js using appropriate syntax rules for each file type
- **Navigation**: Ability to navigate between search matches in file for efficient code review and analysis
- **Performance**: Truncation limits to prevent performance issues when viewing extremely large files
- **Readability**: Optimized display with line numbers and context lines for better code comprehension

#### Recent Searches
- **Persistence**: Local storage with automatic cleanup and deduplication to maintain relevant search history
- **Performance**: Efficient storage and retrieval mechanisms that don't impact application performance
- **User Experience**: Quick access to previous search parameters for improved workflow efficiency
- **Privacy**: No external data transmission ensuring all search history remains local to the user's system

### Performance Optimization Features

#### Large File Handling
- **Streaming Architecture**: Line-by-line processing for files >1MB to maintain consistent memory usage regardless of file size
- **Memory Management**: Configurable buffer sizes and memory limits to optimize performance on different hardware configurations
- **Performance**: Optimized I/O operations for large files using efficient buffering and reading strategies
- **User Experience**: Continues operation without freezing UI, maintaining responsiveness during large file processing

#### Result Limiting
- **Configurable Limits**: User-adjustable result count limits to balance between comprehensive results and performance
- **Early Termination**: Efficient search cancellation when limit reached using Go context cancellation for immediate response
- **Performance**: Prevents overwhelming result sets that could impact UI performance and user experience
- **User Control**: Clear indication when results are limited to maintain user awareness of search scope

### Security Features

#### File Type Allow-Lists
- **Granular Control**: Selective file type restriction providing fine-grained control over search scope
- **Security**: Prevents searching of potentially dangerous file types like executables or system configuration files
- **Performance**: Improves search performance by reducing the number of files that need to be processed and analyzed
- **User Experience**: Intuitive interface for selection that makes security controls accessible without complexity

#### Binary File Detection
- **Multiple Checks**: Multiple validation methods for binary detection to ensure high accuracy and few false positives
- **Performance**: Efficient detection without full file loading by analyzing only file headers and small content samples
- **Security**: Prevents processing of non-text files that could contain malicious content or cause processing errors
- **Accuracy**: High accuracy binary detection algorithms that properly distinguish between text and binary content

#### System Integration Security
- **Directory Traversal Prevention**: Multiple validation layers preventing unauthorized directory access during file operations
- **Editor Integration Safety**: Secure launching of external editors with path validation to prevent command injection
- **File Manager Integration**: Safe integration with system file managers using validated paths only
- **Cross-Platform Security**: Consistent security measures across Windows, Linux, and macOS implementations

## Development Workflow

### Security-First Development

#### Secure Coding Practices
- **Input Validation**: All user inputs validated on both frontend and backend with proper sanitization techniques
- **Output Encoding**: Proper encoding of all content before rendering to prevent injection attacks
- **Error Handling**: Secure error handling without exposing sensitive system information to users
- **Resource Management**: Proper cleanup of resources and connections to prevent resource exhaustion

#### Security Testing Integration
- **Automated Scanning**: Integration of security scanning tools in the development workflow to catch vulnerabilities early
- **Penetration Testing**: Regular security testing of application features to identify potential security gaps
- **Dependency Scanning**: Regular auditing of dependencies for security vulnerabilities in both Go and Node.js ecosystems
- **Code Review Security**: Security-focused code reviews to ensure security best practices are maintained

### Setup and Configuration

#### Prerequisites
- **Go Environment**: Go 1.23+ with proper GOPATH configuration and available in system PATH for module management
- **Node.js Environment**: Node.js 16.x+ with npm/yarn package managers for frontend dependency management
- **Wails CLI**: Properly installed and configured Wails using `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **System Dependencies**: Platform-specific tools for directory selection (zenity/kdialog/yad for Linux, PowerShell for Windows)

#### Project Initialization
1. **Repository Setup**: Clone and initialize the repository using git with proper remote configuration
2. **Dependency Management**: Install Go modules with `go mod tidy` and NPM packages with `npm install`
3. **Development Server**: Start Wails development server with `wails dev` for hot-reloading development
4. **Testing Environment**: Set up testing with proper mocks and test data for comprehensive validation

### Development Process

#### Backend Development
- **Go Modules**: Proper dependency management with go.mod ensuring reproducible builds and clean dependency tracking
- **Testing**: Run comprehensive tests with `go test -v ./...` including unit, integration, and security tests
- **Formatting**: Use `go fmt` for consistent code style following Go community standards and best practices
- **Documentation**: Add comprehensive godoc comments for exported functions, types, and packages for maintainability

#### Frontend Development
- **TypeScript**: Strong typing throughout the codebase with proper interfaces and type definitions for all components
- **Vue 3 Composition API**: Modern Vue.js patterns using composables for logic sharing and state management
- **Component Testing**: Unit tests for all components using Vue Test Utils and Jest with proper mock implementations
- **Performance**: Monitor bundle sizes and load times using webpack bundle analyzer and performance profiling tools

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