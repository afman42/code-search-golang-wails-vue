# Code-Search-Golang Application Architecture

## Overview

The code-search-golang application is a powerful, feature-rich desktop code search tool built with Wails (Go backend + Vue.js frontend). It allows users to search for text patterns within code files across specified directories, with advanced features like regex search, configurable file filtering, security-hardened file operations, exclude patterns, and pagination for better user experience. The application includes extensive performance optimizations, security hardening, and a modern, responsive UI with syntax highlighting.

## Architecture

### Backend (Go)

The backend is a high-performance Go application that handles file system operations and search logic with extensive security and performance optimizations:

**Key Components:**
- `App struct`: Main application with methods for search and system integration
- `SearchRequest`: Contains search parameters with added `AllowedFileTypes` for security (directory, query, extensions, etc.)
- `SearchResult`: Represents individual matches with file path, line number, content
- `SearchWithProgress`: Enhanced search function with real-time progress updates
- `processFileLineByLine`: Memory-efficient streaming function for large files

**Core Features:**
- File system traversal and search with parallel processing
- Cross-platform directory selection (Windows, macOS, Linux with PowerShell support)
- File manager integration with path traversal protection
- Performance optimizations (file size limits, result limits, early termination)
- Progress tracking and reporting with real-time events
- Binary file detection and filtering
- Memory-efficient streaming for large files
- File type allow-lists for enhanced security

**Security Features:**
- Path traversal protection in `ShowInFolder` and file operations
- Input validation and sanitization
- Binary file detection to prevent processing of non-text files
- File type allow-lists to restrict searches to specific extensions

### Frontend (Vue.js)

The frontend is a modern Vue.js 3 application with TypeScript and comprehensive code splitting:

**Key Components:**
- `CodeSearch.vue`: Main application component
- `SearchForm.vue`: Form for search parameters (directory, query, filters, etc.)
- `SearchResults.vue`: Displays search results with pagination
- `ProgressIndicator.vue`: Shows real-time search progress
- `CodeModal.vue`: File preview with syntax highlighting and match navigation
- `useSearch.ts`: Composition composable with all search logic

**Key Features:**
- Real-time search progress updates
- Syntax highlighting for code preview with dynamic imports
- Pagination for large result sets
- Recent searches history with local storage
- Responsive design
- Code splitting and dynamic imports for optimized loading
- Async operations with proper loading states

### Communication Layer (Wails)

Uses Wails framework to connect Go backend with Vue.js frontend:
- Generated TypeScript bindings for Go functions
- Real-time event system for progress updates
- Type-safe communication between backend and frontend
- Cross-platform compatibility

## Performance Optimizations

### Backend:
- File size limits (default 10MB) to prevent memory issues
- Result limits (default 1000) to prevent overwhelming response
- Parallel file processing using Go routines with worker pools
- Binary file detection and skipping
- Memory-efficient streaming for large files (threshold 1MB) to prevent memory issues
- Early termination when max results are reached using context cancellation
- Context-aware file processing to stop operations efficiently

### Frontend:
- Code splitting with dynamic imports to reduce initial bundle size
- Pagination (10 results per page) to limit DOM elements
- Efficient rendering with Vue's reactivity system
- Large file handling with truncation limits (10,000 lines max)
- Asynchronous syntax highlighting loaded on-demand
- Proper loading states with `isReady` flags

## Security Considerations

- Input validation for all search parameters
- File path validation and sanitization to prevent directory traversal
- HTML sanitization to prevent XSS in UI rendering
- Proper handling of special characters and Unicode
- Path traversal protection in `ShowInFolder` and file operations
- File type allow-lists to restrict searches to specific extensions
- Binary file detection to prevent processing of non-text files
- Sanitized file path handling in all system operations

## Testing Strategy

### Backend Tests
- Unit tests for individual functions (directory validation, file operations)
- Integration tests for search functionality with all new features
- Edge case testing (large files, Unicode, special characters)
- Security testing (path traversal, injection attacks)
- Binary file filtering tests to ensure include/exclude works properly
- File type allow-list tests to ensure security filtering works
- Performance tests for streaming large files
- Race condition tests for early termination at max results

### Frontend Tests
- Component tests for UI elements
- Integration tests for backend communication
- Composable tests for business logic
- Mocked Wails function calls for isolated testing
- Async component tests for CodeModal with loading states
- Syntax highlighting tests with dynamic imports
- Code splitting verification tests

## Key Features

### Search Capabilities:
- Text and regex search with advanced pattern matching
- Case-sensitive/insensitive options
- File extension filtering
- File type allow-lists for security
- Exclude patterns (node_modules, .git, etc.)
- Context lines (before/after matches)
- Binary file inclusion/exclusion options
- Min/Max file size filtering

### User Experience:
- Native directory selection dialogs across all platforms
- Real-time progress visualization with detailed metrics
- Code syntax highlighting with dynamic language loading
- Recent searches with local storage persistence
- Responsive design for different screen sizes
- Pagination with navigation controls
- File preview modal with match navigation
- Copy to clipboard functionality
- Progress tracking with file counts and percentages

### Performance & Security:
- Parallel processing with Go routines
- Memory-efficient streaming for large files
- Early termination when max results reached
- File type allow-lists to restrict search scope
- Path traversal protection
- Input sanitization and validation
- Code splitting for optimized loading
- Dynamic syntax highlighting imports

## Development Workflow

### Building:
1. Run `wails dev` for development with hot reload
2. Run `wails build` for production builds with automatic code splitting
3. Execute `go test -v` for backend tests
4. Run `npm test` in frontend directory for frontend tests
5. Build generates optimized chunks for faster loading

### Architecture Benefits:
- Clear separation between backend (Go) and frontend (Vue.js)
- Type safety with TypeScript and Go typing
- Cross-platform compatibility with native system integration
- Scalable architecture with parallel processing and streaming for large files
- Comprehensive test coverage with unit, integration, and security tests
- Performance optimized with code splitting and memory-efficient operations
- Security-hardened with path traversal protection and file type allow-lists
- Modern architecture with Vue 3 Composition API and async operations