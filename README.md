# Code Search Golang

A powerful, feature-rich desktop code search application built with Wails (Go backend + Vue.js frontend). This application allows users to search for text patterns, keywords, and regular expressions across code files in specified directories with advanced filtering, security, and performance optimizations.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Installation](#installation)
- [Usage](#usage)
- [Development](#development)
- [Performance Optimizations](#performance-optimizations)
- [Security Features](#security-features)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

## Features

### Core Search Functionality

- **Text Search**: Search for specific text patterns in your codebase
- **Case Sensitivity Control**: Toggle between case-sensitive and case-insensitive searches
- **File Extension Filtering**: Filter search results by file extension (e.g., `.go`, `.js`, `.py`)
- **Regex Support**: Use regular expressions for advanced pattern matching
- **Directory Browsing**: Native directory selection dialog for easy path selection
- **File Location**: Open the containing folder of search results

### Enhanced Features

- **Binary File Handling**: Option to include or exclude binary files from search
- **File Type Allow-Lists**: Restrict searches to specific file types for enhanced security
- **Configurable Limits**: Adjustable maximum file size and result count limits
- **Subdirectory Search**: Toggle to search in subdirectories or only in the selected directory
- **Unicode Support**: Proper handling of Unicode characters in search queries and files
- **Performance Optimizations**: File size limits, parallel processing, and memory-efficient streaming for large files to prevent memory issues
- **Result Truncation**: Automatic truncation to prevent overwhelming result sets
- **Search History**: Recent searches saved in local storage for quick access
- **Exclude Patterns**: Multi-select dropdown with common patterns (node_modules, .git, etc.) and custom patterns
- **Pagination**: Results are paginated (10 per page) with navigation controls for better performance and usability
- **Early Termination**: Search stops when maximum results are reached, saving computation time
- **Security Hardening**: Path traversal protection and input sanitization

### User Interface

- **Intuitive Design**: Clean, modern UI optimized for code search workflows
- **Real-time Feedback**: Shows search progress and result counts
- **Highlighted Matches**: Visual highlighting of search terms in results
- **Line Numbers**: Shows exact line numbers where matches were found
- **Copy Functionality**: Easy copying of matched lines to clipboard
- **Responsive Layout**: Works well on different screen sizes
- **Progress Visualization**: Visual progress bar with percentage and file count
- **Context Display**: Shows surrounding lines before and after matches for context
- **Syntax Highlighting**: Enhanced code display with language-specific syntax highlighting using highlight.js with Agate theme
- **Code Modal**: Improved file preview modal with line-by-line syntax highlighting and search match highlighting
- **Performance Optimized**: Efficient rendering for large files with truncation limits to prevent performance issues
- **Navigation Features**: Enhanced navigation with ability to go to next match in search results
- **Visual Enhancements**: Better visual separation between line numbers and code content for improved readability

## Architecture

### Backend (Go)

The backend is built with Go and handles all file system operations and search logic with extensive security and performance optimizations. The architecture follows these core principles:

- **Parallel Processing**: Uses Go goroutines for concurrent file processing
- **Memory Efficiency**: Implements streaming for large files to prevent memory issues
- **Security First**: Built-in protections against path traversal and malicious inputs
- **Cross-Platform Compatibility**: Native experience across Windows, Linux, and macOS
- **Scalability**: Designed to handle large codebases efficiently

**Key Components:**

- `App struct`: Main application with methods for search and system integration
- `SearchRequest`: Contains search parameters with added `AllowedFileTypes` for security (directory, query, extensions, etc.)
- `SearchResult`: Represents individual matches with file path, line number, content
- `SearchWithProgress`: Enhanced search function with real-time progress updates and cancellation support
- `processFileLineByLine`: Memory-efficient streaming function for large files to prevent memory issues
- `isBinary`: Binary file detection with multiple validation layers

**Core Features:**

- File system traversal and search with parallel processing using worker pools
- Cross-platform directory selection (Windows PowerShell, Linux zenity/kdialog/yad, macOS AppleScript\*)
- File manager integration with comprehensive path traversal protection
- Performance optimizations (file size limits, result limits, early termination with context cancellation)
- Progress tracking and reporting with real-time events
- Binary file detection and intelligent filtering
- Memory-efficient streaming for large files (>1MB threshold)
- File type allow-lists for enhanced security and performance
- Context-aware file processing with before/after line capture

### Frontend (Vue.js)

The frontend is built with Vue.js 3 and TypeScript with comprehensive code splitting and optimized performance:

**Architecture Principles:**

- **Component-Based Design**: Modular, reusable components for maintainability
- **Composition API**: Centralized business logic using composables pattern
- **Performance Optimization**: Code splitting, dynamic imports, and virtual rendering
- **Type Safety**: Comprehensive TypeScript coverage for reliability
- **Responsive Design**: Optimized for different screen sizes and devices

**Key Components:**

- `CodeSearch.vue`: Main orchestrator component that composes the entire UI
- `SearchForm.vue`: Comprehensive form for all search parameters and options with validation
- `SearchResults.vue`: Displays results with pagination and interactive features
- `ProgressIndicator.vue`: Real-time search progress visualization with detailed metrics
- `CodeModal.vue`: Advanced file preview with syntax highlighting and match navigation
- `useSearch.ts`: Composition composable with all search business logic, state management, and Wails integration

**Key Features:**

- Real-time search progress updates with detailed metrics and visual feedback
- Advanced syntax highlighting for code preview with dynamic language loading
- Pagination system for large result sets (10 results per page) with navigation controls
- Recent searches history with localStorage persistence and intelligent deduplication
- Fully responsive design that works seamlessly across different screen sizes
- Code splitting and dynamic imports for optimized initial loading performance
- Asynchronous operations with proper loading states and error handling
- Accessibility features with proper ARIA attributes and keyboard navigation

### Communication Layer (Wails)

Uses Wails framework to connect Go backend with Vue.js frontend with type safety and real-time capabilities:

- **Generated TypeScript bindings**: Type-safe communication between Go and TypeScript
- **Real-time event system**: Efficient progress updates and status notifications without blocking operations
- **Cross-platform compatibility**: Native system integration across all platforms
- **Performance optimized**: Low-latency communication with efficient data serialization
- **Error handling**: Comprehensive error propagation from backend to frontend

## Installation

### Prerequisites

- Go 1.23 or higher
- Node.js 16.x or higher
- Wails CLI: Install using `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Building from Source

1. Clone the repository:

   ```bash
   git clone <repository-url>
   cd code-search-golang
   ```

2. Install Go dependencies:

   ```bash
   go mod tidy
   ```

3. Install frontend dependencies:

   ```bash
   cd frontend
   npm install
   ```

4. Build the application:
   ```bash
   wails build
   ```

The executable will be created in the `build/bin/` directory.

## Usage

### Basic Search

1. Click "Browse" to select a directory to search in
2. Enter your search query in the "Search Query" field
3. Optionally specify a file extension to filter results
4. Configure search options as needed
5. Click "Search Code" to begin the search

### Code Preview Features

The application includes enhanced code preview capabilities:

- **Syntax Highlighting**: Code files are displayed with proper syntax highlighting based on file extension using highlight.js with the Agate theme
- **Line Numbers**: Each line is numbered for easy reference
- **Search Match Highlighting**: Search terms are highlighted in yellow for easy identification
- **Performance Optimized**: Large files are processed efficiently with a maximum limit of 10,000 lines to prevent browser crashes
- **Navigation**: Navigate between search matches using the "Next Match" button
- **File Preview**: Click on any file in search results to open a detailed preview modal
- **Readability**: Enhanced spacing between line numbers and code content for better readability

### Advanced Search Options

- **Case Sensitive**: Check this box for case-sensitive searches
- **Regex Search**: Enable to use regular expressions in your search query
- **Include Binary**: Include binary files in the search (disabled by default)
- **Search Subdirs**: Search in subdirectories (enabled by default)
- **Max File Size**: Limit file size to include in search (default 10MB)
- **Max Results**: Limit number of results returned (default 1000)
- **Min File Size**: Minimum file size to include in search
- **File Type Allow-Lists**: Restrict searches to specific file extensions (e.g., only .go, .js, .py files) with UI controls for easy selection
- **Multi-part Extension Support**: Support for double extensions like .tar.gz, .min.js, .config.bak, etc.
- **Exclude Patterns**: Multi-select dropdown to choose common patterns to exclude (e.g., node_modules, .git) or add custom patterns

### Search Results

- Results display file path, line number, and matched content
- Click on file path to open the containing folder
- Use "Copy" button to copy matched lines to clipboard
- "Matched" text shows the actual text that matched your query
- Results are paginated for better performance (10 per page)
- Use pagination controls to navigate through results
- Results are limited to prevent performance issues

## Development

### Prerequisites

Before starting development, ensure you have the following installed:

- Go 1.23 or higher with proper GOPATH configuration
- Node.js 16.x or higher with npm
- Wails CLI: Install using `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- System dependencies for your platform (zenity/kdialog/yad for Linux, PowerShell for Windows)

### Live Development

To run in live development mode with hot reloading:

```bash
wails dev
```

This starts a Vite development server with hot reload for frontend changes. The Go backend communicates with the frontend through Wails bindings. Changes to both Go and Vue.js code will automatically reload in the development application.

### Testing Strategy

Comprehensive testing approach ensures code quality and security:

**Backend Tests:**

- Run all backend tests: `go test -v ./...`
- Execute specific test: `go test -v -run TestFunctionName`
- Run tests with coverage: `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`

**Frontend Tests:**

- Run all frontend tests: `cd frontend && npm test`
- Run tests in watch mode: `cd frontend && npm run test:watch`
- Generate test coverage: `cd frontend && npm run test:coverage`

**Integration Tests:**

- Cross-platform functionality testing
- End-to-end search workflow validation
- Performance and security testing

### Project Structure

```
├── main.go                 # Wails application entry point
├── app_core.go             # Core application logic and search engine
├── app.go                  # Linux-specific system integration
├── appWindows.go           # Windows-specific system integration
├── app_test.go             # Core backend tests
├── *.go                    # Additional Go source files (binary_file_test.go, search_with_progress_test.go, etc.)
├── go.mod                  # Go module dependencies
├── go.sum                  # Go module checksums
├── wails.json              # Wails configuration
├── frontend/               # Vue.js frontend components
│   ├── src/
│   │   ├── main.ts        # Frontend entry point
│   │   ├── App.vue        # Root Vue component
│   │   ├── assets/        # Static assets
│   │   ├── components/    # Main components
│   │   │   ├── CodeSearch.vue     # Main search orchestrator
│   │   │   ├── StartupLoader.vue  # Initial loading state
│   │   │   └── ui/                # UI components
│   │   │       ├── CodeModal.vue      # File preview modal
│   │   │       ├── ProgressIndicator.vue # Progress visualization
│   │   │       ├── SearchForm.vue      # Search parameter form
│   │   │       └── SearchResults.vue   # Results display
│   │   ├── composables/   # Shared logic (useSearch composable)
│   │   ├── types/         # TypeScript interfaces
│   │   └── style.css      # Global styles
│   ├── tests/             # Jest unit tests and configuration
│   ├── public/            # Static assets
│   ├── package.json       # NPM dependencies
│   ├── tsconfig.json      # TypeScript configuration
│   ├── vite.config.ts     # Vite build configuration
│   └── wailsjs/           # Generated Wails bindings
├── build/                 # Build outputs
└── README.md              # Project documentation
```

### Key Architecture Components

#### Backend Components (Go)

- **App struct**: Main application managing context and search lifecycle
- **SearchWithProgress()**: Core search engine with real-time progress, cancellation, and parallel processing
- **ValidateDirectory()**: Comprehensive directory validation with security checks
- **SelectDirectory()**: Cross-platform native directory selection (Windows PowerShell, Linux zenity/kdialog/yad, macOS AppleScript\*)
- **ShowInFolder()**: Secure file manager integration with path traversal protection
- **processFileLineByLine()**: Memory-efficient streaming processor for large files
- **isBinary()**: Advanced binary file detection with multiple validation layers
- **SearchRequest/SearchResult**: Type-safe data structures for search operations

#### Frontend Components (Vue.js)

- **CodeSearch.vue**: Main application orchestrator component
- **SearchForm.vue**: Comprehensive form for search parameters with validation and recent search integration
- **SearchResults.vue**: Paginated results display with interactive features
- **ProgressIndicator.vue**: Real-time visual progress tracking
- **CodeModal.vue**: Advanced file preview with syntax highlighting and match navigation
- **useSearch.ts**: Composition composable containing all search business logic, state management, and Wails integration
- **TypeScript interfaces**: Strong typing for all data structures (SearchRequest, SearchResult, etc.)

### Development Best Practices

#### Go Backend Development

- **Context Usage**: Always use context for cancellation and timeout handling
- **Error Handling**: Comprehensive error handling with meaningful messages
- **Memory Management**: Efficient memory usage with streaming for large files
- **Concurrency**: Proper use of goroutines and channels for parallel processing
- **Testing**: Write comprehensive unit and integration tests
- **Documentation**: Add godoc comments for all exported functions

#### Vue.js Frontend Development

- **Composition API**: Use composables for shared logic and state management
- **Type Safety**: Leverage TypeScript for type-safe development
- **Performance**: Optimize rendering and component updates
- **Accessibility**: Implement proper ARIA attributes and keyboard navigation
- **Component Design**: Create modular, reusable components
- **Testing**: Write comprehensive unit and integration tests

#### Cross-Platform Considerations

- **System Integration**: Test directory selection and file operations across platforms
- **UI Consistency**: Ensure consistent user experience across different OS
- **Performance**: Optimize for different system capabilities
- **Build Process**: Verify builds work correctly on different platforms

### Build and Deployment

#### Development Build

- Use `wails dev` for development with hot reload
- Frontend changes auto-reload in development mode
- Go backend changes require restart of dev server

#### Production Build

- Execute `wails build` to create production executables
- Output executables will be in `build/bin/` directory
- Executables are self-contained with all necessary dependencies
- Builds are created for the host platform by default

## Performance Optimizations

### Backend Optimizations:

- **File size limits**: Configurable maximum file size (default 10MB) to prevent memory issues and performance degradation
- **Result limits**: Adjustable result count limits (default 1000) to prevent overwhelming responses and maintain UI responsiveness
- **Parallel processing**: Efficient use of Go goroutines with dynamic worker pools sized to CPU cores
- **Binary detection**: Intelligent skipping of binary files to reduce processing time and memory usage
- **Memory-efficient streaming**: Large files (>1MB threshold) processed line-by-line to prevent memory issues
- **Early termination**: Context cancellation used to stop searches when max results reached, saving computation time
- **File system optimization**: Efficient file traversal with intelligent filtering and exclusion patterns
- **Buffer management**: Configurable buffer sizes (default 1MB) for optimal I/O performance
- **Resource management**: Proper cleanup of file handles and memory during operations

### Frontend Optimizations:

- **Code splitting**: Strategic splitting of application components to reduce initial bundle size and improve load time
- **Dynamic imports**: On-demand loading of syntax highlighting libraries and other heavy components
- **Efficient rendering**: Optimized Vue.js reactivity system with virtual scrolling for large result sets
- **Pagination system**: Results segmented into pages (10 per page) to limit DOM elements and maintain performance
- **Large file handling**: Truncation limits (10,000 lines max) to prevent browser crashes and maintain responsiveness
- **Asynchronous operations**: Non-blocking UI updates with proper loading states and progress indicators
- **Memory management**: Efficient state management to prevent memory leaks and optimize garbage collection
- **Bundle optimization**: Tree shaking and minification to reduce final bundle size

## Security Features

### Backend Security Measures:

- **Input Validation & Sanitization**: Comprehensive validation and sanitization of all user inputs, search queries, and file paths
- **Path Traversal Protection**: Robust protection against directory traversal attacks using filepath.Clean and validation checks
- **File Access Control**: File type allow-lists to restrict searches to specific extensions for enhanced security
- **File Path Sanitization**: All file paths sanitized before any system operations to prevent malicious access
- **Binary File Detection**: Prevention of processing of potentially dangerous binary files when not required
- **Permission Validation**: Verification of file access permissions before any read operations
- **Sandboxing**: Isolated search operations limited to specified directories only

### Frontend Security Measures:

- **Content Security Policy**: Implementation of strict CSP to prevent injection attacks
- **XSS Prevention**: Sanitization of all content before rendering to prevent cross-site scripting
- **Input Sanitization**: Validation and sanitization of all user inputs before transmission to backend
- **Secure State Management**: Proper handling of sensitive data in application state
- **Trusted Types**: Ensuring only safe content is rendered in the browser

### Communication Security:

- **Type Safety**: Wails-generated TypeScript bindings ensure type-safe communication
- **Secure Event Handling**: Proper validation of all real-time events from backend
- **Data Integrity**: Protected communication channel between frontend and backend

## Configuration

Edit `wails.json` to configure project settings:

- Application name and executable filename
- Window dimensions and properties
- Frontend build settings
- Development server configuration

## Troubleshooting

### Common Search Issues

#### Search Returns No Results

1. **Verify directory path**: Ensure the selected directory exists and is accessible
2. **Check search query**: Look for typos or overly restrictive patterns in your query
3. **Extension filtering**: Confirm file extensions match if using extension filtering
4. **File size limits**: Verify files aren't larger than the maximum file size limit (10MB default)
5. **Case sensitivity**: Check if case sensitivity settings are affecting results
6. **Regex patterns**: Validate regex patterns aren't malformed or too restrictive
7. **Exclude patterns**: Check if exclude patterns are filtering out expected files
8. **Binary files**: Ensure binary file handling settings are configured as expected

#### Performance and Memory Issues

- **Large files**: The application limits file size to 10MB by default to prevent memory issues
- **Result limits**: Result count is limited to 1000 by default to maintain performance
- **Large codebases**: For very large directories, consider using exclude patterns (node_modules, .git, etc.)
- **System resources**: Monitor CPU and memory usage during searches; close other applications if needed
- **Complex regex**: Simplify overly complex regex patterns that may cause performance issues
- **Streaming threshold**: Files larger than 1MB are streamed line-by-line for memory efficiency

### Platform-Specific Issues

#### Directory Selection Problems

- **Linux systems**: Ensure one of the following tools is installed (in order of preference):
  - `zenity` (GNOME desktop environments)
  - `kdialog` (KDE desktop environments)
  - `yad` (multi-desktop environments)
  - Install with: `sudo apt install zenity` or equivalent for your package manager
- **Windows systems**: PowerShell is used for native directory selection (included with Windows 7+)
- **macOS systems**: AppleScript integration for native experience (functionality pending implementation)

#### System Integration Issues

- **File manager integration**: `Show in Folder` functionality may not work in all CI/test environments
- **Permissions**: Ensure application has read permissions for the directories being searched
- **Path length**: On Windows, very long file paths may cause issues; use shorter directory paths if possible

### UI/UX Issues

#### Frontend Performance

- **Large result sets**: Use pagination (10 results per page) to handle many results efficiently
- **Browser crashes**: Large files are limited to 10,000 lines maximum to prevent browser issues
- **Slow rendering**: UI remains responsive during searches with real-time progress updates
- **Memory usage**: Monitor browser memory for very large result sets

#### Search Form Issues

- **Invalid inputs**: Form validation prevents search execution with invalid parameters
- **Recent searches**: Clear localStorage if recent searches aren't appearing correctly
- **State persistence**: Settings saved in browser localStorage between sessions

### Development and Build Issues

#### Build Problems

- **Missing dependencies**: Run `go mod tidy` and `npm install` to resolve dependency issues
- **Wails version**: Ensure Wails CLI is up-to-date with `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Platform-specific**: Check system requirements for your target platform

#### Development Environment

- **Hot reload**: `wails dev` provides hot reload for both Go and Vue.js files
- **Debugging**: Use browser developer tools and Go debugging capabilities
- **Testing**: Run both backend (`go test -v`) and frontend (`npm test`) tests before committing

### Advanced Troubleshooting

#### Performance Tuning

- **Worker pool size**: Automatically adjusted based on CPU cores, but can be monitored
- **Buffer sizes**: Configured for optimal I/O performance (1MB default)
- **Memory profiling**: Use Go's built-in profiling tools to diagnose memory issues
- **CPU profiling**: Identify performance bottlenecks in search algorithms

#### Security Considerations

- **Path traversal**: All file paths are sanitized - ensure your search paths are valid
- **File access**: Application respects system file permissions - ensure proper access
- **Input validation**: All user inputs are validated - check for special characters that might be filtered

## Contributing

We welcome contributions from the community! Here's how you can help improve the code search application:

### Getting Started

#### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/code-search-golang.git`
3. Add the original repository as upstream: `git remote add upstream https://github.com/ORIGINAL_OWNER/code-search-golang.git`
4. Create a development branch: `git checkout -b feature/your-feature-name`

#### Development Environment Setup

1. Install Go 1.23+ and Node.js 16.x+
2. Install Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
3. Navigate to the project directory and run:
   - Backend dependencies: `go mod tidy`
   - Frontend dependencies: `cd frontend && npm install`
4. Test the development setup: `wails dev`

### Contribution Guidelines

#### Code Quality Standards

- **Go Code**: Follow the official Go formatting guidelines (`go fmt`)
- **Vue.js/TypeScript**: Maintain consistency with existing code style
- **Documentation**: Update relevant documentation for new features
- **Comments**: Add meaningful comments for complex logic
- **Testing**: Include tests for all new functionality

#### Architecture Adherence

- **Separation of Concerns**: Keep frontend and backend concerns separate
- **Wails Patterns**: Follow Wails framework conventions for Go-Vue.js communication
- **Security First**: Maintain security measures in all code changes
- **Performance**: Consider performance implications of changes
- **Cross-Platform**: Ensure changes work across all supported platforms

#### Testing Requirements

1. **Backend Tests**: Add unit tests for new Go functions using Go's testing package
2. **Frontend Tests**: Add Jest tests for new Vue.js components and composables
3. **Integration Tests**: Verify end-to-end functionality where appropriate
4. **Performance Tests**: Consider adding performance tests for performance-critical code
5. **Run All Tests**: Ensure all existing tests pass before submitting

### Development Workflow

#### Before Submitting Changes

1. **Run Tests**: Execute `go test -v ./...` and `cd frontend && npm test`
2. **Format Code**: Use `go fmt` for Go files and `npm run format` for frontend files
3. **Update Documentation**: Update README.md, ARCHITECTURE_AND_TESTING_SUMMARY.md if needed
4. **Performance Testing**: Verify changes don't negatively impact performance
5. **Security Review**: Ensure security measures remain intact

#### Pull Request Process

1. **Sync with Upstream**: `git fetch upstream && git rebase upstream/main`
2. **Squash Commits**: Combine related commits into meaningful units
3. **Write Good Commit Messages**: Follow conventional commit format when possible
4. **Create PR**: Submit pull request with clear description of changes
5. **Address Feedback**: Respond to code review comments promptly

### Areas Needing Contributions

#### Feature Development

- **Search Enhancements**: Additional search capabilities and pattern matching
- **UI/UX Improvements**: Better user experience and interface design
- **Performance Optimizations**: Further improvements to search speed and memory usage
- **Platform Support**: Additional platform-specific features and fixes
- **Export Features**: Options to export search results in various formats

#### Bug Fixes

- **Cross-Platform Issues**: Platform-specific bugs and inconsistencies
- **Performance Bottlenecks**: Issues with speed or memory usage
- **UI/UX Issues**: Problems with user interface or experience
- **Security Vulnerabilities**: Any identified security issues
- **Edge Case Handling**: Issues with unusual file types or search patterns

#### Documentation

- **User Guides**: Enhanced user documentation and tutorials
- **Architecture Documents**: Updates to design documents and explanations
- **API Documentation**: Go function documentation and TypeScript interfaces
- **Troubleshooting**: Additional troubleshooting scenarios and solutions

### Code Review Process

#### What We Look For

- **Functionality**: Does the code work as intended?
- **Security**: Are security measures maintained and enhanced?
- **Performance**: Does the code maintain good performance characteristics?
- **Maintainability**: Is the code clean, well-organized, and easy to maintain?
- **Test Coverage**: Are appropriate tests included?
- **Documentation**: Is necessary documentation updated?

#### Review Timeline

- Initial review: Within 3-5 business days
- Follow-up reviews: Within 1-2 business days
- Final approval: After all feedback is addressed

### Questions and Support

If you have questions about contributing:

- Open an issue with the "question" tag
- Join our community discussions (if available)
- Check existing documentation and issues for answers
- Examine existing pull requests to understand patterns

### Code of Conduct

Please note that this project follows a Code of Conduct to ensure a welcoming environment for all contributors. By participating, you agree to maintain professional and respectful communication.

Thank you for your interest in contributing to the Code Search Golang project!

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [Wails](https://wails.io/) framework
- Uses Vue 3 and TypeScript for the frontend
- Leverages Go for high-performance file system operations
- Vibe Coding from Qwen Coder
