# Code-Search-Golang Application Architecture

## Overview

The code-search-golang application is a desktop code search tool built with Wails (Go backend + Vue.js frontend). It allows users to search for text patterns within code files across specified directories, with advanced features like regex search, file filtering, exclude patterns, and pagination for better user experience.

## Architecture

### Backend (Go)

The backend is a Go application that handles file system operations and search logic:

**Key Components:**
- `App struct`: Main application with methods for search and system integration
- `SearchRequest`: Contains search parameters (directory, query, extensions, etc.)
- `SearchResult`: Represents individual matches with file path, line number, content
- `SearchWithProgress`: Enhanced search function with real-time progress updates

**Core Features:**
- File system traversal and search
- Cross-platform directory selection
- File manager integration
- Performance optimizations (file size limits, result limits)
- Progress tracking and reporting

### Frontend (Vue.js)

The frontend is a Vue.js 3 application with TypeScript:

**Key Components:**
- `CodeSearch.vue`: Main application component
- `SearchForm.vue`: Form for search parameters (directory, query, filters, etc.)
- `SearchResults.vue`: Displays search results with pagination
- `ProgressIndicator.vue`: Shows real-time search progress
- `CodeModal.vue`: File preview with syntax highlighting

**Key Features:**
- Real-time search progress updates
- Syntax highlighting for code preview
- Pagination for large result sets
- Recent searches history
- Responsive design

### Communication Layer (Wails)

Uses Wails framework to connect Go backend with Vue.js frontend:
- Generated TypeScript bindings for Go functions
- Real-time event system for progress updates
- Type-safe communication between backend and frontend

## Performance Optimizations

### Backend:
- File size limits (default 10MB) to prevent memory issues
- Result limits (default 1000) to prevent overwhelming response
- Parallel file processing using Go routines
- Binary file detection and skipping

### Frontend:
- Pagination (10 results per page) to limit DOM elements
- Efficient rendering with Vue's reactivity system
- Large file handling with truncation limits (10,000 lines max)

## Security Considerations

- Input validation for all search parameters
- File path validation to prevent directory traversal
- HTML sanitization to prevent XSS
- Proper handling of special characters and Unicode

## Testing Strategy

### Backend Tests
- Unit tests for individual functions (directory validation, file operations)
- Integration tests for search functionality
- Edge case testing (large files, Unicode, special characters)
- Security testing (path traversal, injection attacks)

### Frontend Tests
- Component tests for UI elements
- Integration tests for backend communication
- Composable tests for business logic
- Mocked Wails function calls for isolated testing

## Key Features

### Search Capabilities:
- Text and regex search
- Case-sensitive/insensitive options
- File extension filtering
- Exclude patterns (node_modules, .git, etc.)
- Context lines (before/after matches)

### User Experience:
- Native directory selection dialogs
- Real-time progress visualization
- Code syntax highlighting
- Recent searches with local storage
- Responsive design for different screen sizes

## Development Workflow

### Building:
1. Run `wails dev` for development with hot reload
2. Run `wails build` for production builds
3. Execute `go test -v` for backend tests
4. Run `npm test` in frontend directory for frontend tests

### Architecture Benefits:
- Clear separation between backend (Go) and frontend (Vue.js)
- Type safety with TypeScript and Go typing
- Cross-platform compatibility
- Scalable architecture with parallel processing
- Comprehensive test coverage