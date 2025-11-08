# Code Search Golang - Vibe Coding from Qwen Coder

A powerful, feature-rich desktop code search application built with Wails (Go backend + Vue.js frontend). This application allows users to search for text patterns, keywords, and regular expressions across code files in specified directories with advanced filtering, security, and performance optimizations. The application combines the performance and security of Go for backend operations with the modern, responsive UI capabilities of Vue.js, creating an efficient and user-friendly code search experience.

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

- **Text Search**: Search for specific text patterns in your codebase with high accuracy and performance
- **Case Sensitivity Control**: Toggle between case-sensitive and case-insensitive searches for flexible matching
- **File Extension Filtering**: Filter search results by specific file extensions (e.g., `.go`, `.js`, `.py`) for targeted searches
- **Regex Support**: Use regular expressions for advanced pattern matching with support for complex search patterns
- **Directory Browsing**: Native directory selection dialog for easy path selection with cross-platform compatibility
- **File Location**: Open the containing folder of search results directly from the application for quick navigation

### Enhanced Features

- **Binary File Handling**: Option to include or exclude binary files from search with intelligent detection algorithms
- **File Type Allow-Lists**: Restrict searches to specific file types for enhanced security and performance by limiting scope
- **Configurable Limits**: Adjustable maximum file size and result count limits to optimize performance and resource usage
- **Subdirectory Search**: Toggle to search in subdirectories or limit to the selected directory for focused search results
- **Unicode Support**: Proper handling of Unicode characters in search queries and files for global language compatibility
- **Performance Optimizations**: File size limits, parallel processing, and memory-efficient streaming for large files to prevent memory issues
- **Result Truncation**: Automatic truncation to prevent overwhelming result sets while maintaining usability
- **Search History**: Recent searches saved in local storage for quick access and improved user workflow
- **Exclude Patterns**: Multi-select dropdown with common patterns (node_modules, .git, etc.) and custom patterns for intelligent filtering
- **Pagination**: Results are paginated (10 per page) with navigation controls for better performance and usability
- **Early Termination**: Search stops when maximum results are reached, saving computation time and resources
- **Security Hardening**: Path traversal protection and input sanitization to prevent malicious file access

### User Interface

- **Intuitive Design**: Clean, modern UI optimized specifically for code search workflows with developer-focused interactions
- **Real-time Feedback**: Shows search progress and result counts with detailed metrics and visual indicators
- **Highlighted Matches**: Visual highlighting of search terms in results for quick identification and context
- **Line Numbers**: Shows exact line numbers where matches were found for precise reference and navigation
- **Copy Functionality**: Easy copying of matched lines to clipboard with one-click copy buttons for efficient workflow
- **Responsive Layout**: Works well on different screen sizes with adaptive design that maintains usability on all devices
- **Progress Visualization**: Visual progress bar with percentage and file count for clear understanding of search status
- **Context Display**: Shows surrounding lines before and after matches for full context and better understanding
- **Syntax Highlighting**: Enhanced code display with language-specific syntax highlighting using highlight.js with Agate theme for readability
- **Code Modal**: Improved file preview modal with line-by-line syntax highlighting and search match highlighting for detailed inspection
- **Performance Optimized**: Efficient rendering for large files with truncation limits to prevent browser crashes and maintain responsiveness
- **Navigation Features**: Enhanced navigation with ability to go to next match in search results for streamlined browsing
- **Visual Enhancements**: Better visual separation between line numbers and code content for improved readability and reduced eye strain

## Architecture

### Backend (Go)

The backend is built with Go and handles all file system operations and search logic with extensive security and performance optimizations. The architecture follows these core principles:

- **Parallel Processing**: Uses Go goroutines for concurrent file processing, maximizing CPU utilization and search performance
- **Memory Efficiency**: Implements streaming for large files to prevent memory issues, processing line-by-line without loading entire files
- **Security First**: Built-in protections against path traversal and malicious inputs with comprehensive validation and sanitization
- **Cross-Platform Compatibility**: Native experience across Windows, Linux, and macOS with platform-specific optimizations
- **Scalability**: Designed to handle large codebases efficiently with worker pools and context-aware operations

**Key Components:**

- `App struct`: Main application with methods for search and system integration, managing application lifecycle and context
- `SearchRequest`: Contains search parameters with added `AllowedFileTypes` for security (directory, query, extensions, etc.) - provides flexible configuration
- `SearchResult`: Represents individual matches with file path, line number, content - includes context lines for better understanding
- `SearchWithProgress`: Enhanced search function with real-time progress updates and cancellation support for user experience
- `processFileLineByLine`: Memory-efficient streaming function for large files to prevent memory issues, processes content without loading everything
- `isBinary`: Binary file detection with multiple validation layers to identify non-text files and optimize search performance

**Core Features:**

- File system traversal and search with parallel processing using worker pools optimized for available CPU cores
- Cross-platform directory selection (Windows PowerShell, Linux zenity/kdialog/yad, macOS AppleScript*) ensuring native user experience
- File manager integration with comprehensive path traversal protection to prevent unauthorized file system access
- Performance optimizations (file size limits, result limits, early termination with context cancellation) for efficient resource usage
- Progress tracking and reporting with real-time events providing users with detailed search metrics and status
- Binary file detection and intelligent filtering to avoid processing non-text files and improve search speed
- Memory-efficient streaming for large files (>1MB threshold) preventing memory overflow during processing
- File type allow-lists for enhanced security and performance by restricting search scope to relevant file types
- Context-aware file processing with before/after line capture providing additional context for search matches

### Frontend (Vue.js)

The frontend is built with Vue.js 3 and TypeScript with comprehensive code splitting and optimized performance, providing an intuitive and responsive user interface:

**Architecture Principles:**

- **Component-Based Design**: Modular, reusable components for maintainability with clear separation of concerns and responsibilities
- **Composition API**: Centralized business logic using composables pattern for better code organization and reusability
- **Performance Optimization**: Code splitting, dynamic imports, and virtual rendering to minimize initial load time and memory usage
- **Type Safety**: Comprehensive TypeScript coverage for reliability, preventing runtime errors and improving development experience
- **Responsive Design**: Optimized for different screen sizes and devices ensuring consistent user experience across platforms

**Key Components:**

- `CodeSearch.vue`: Main orchestrator component that composes the entire UI, managing state and coordinating interactions
- `SearchForm.vue`: Comprehensive form for all search parameters and options with validation and recent search integration
- `SearchResults.vue`: Displays results with pagination and interactive features, providing efficient navigation and result management
- `ProgressIndicator.vue`: Real-time search progress visualization with detailed metrics showing file counts, progress percentage, and time estimates
- `CodeModal.vue`: Advanced file preview with syntax highlighting and match navigation enabling detailed code inspection and analysis
- `useSearch.ts`: Composition composable with all search business logic, state management, and Wails integration providing centralized functionality

**Key Features:**

- Real-time search progress updates with detailed metrics and visual feedback showing file processed counts and estimated remaining time
- Advanced syntax highlighting for code preview with dynamic language loading using highlight.js and automatic language detection
- Pagination system for large result sets (10 results per page) with navigation controls preventing UI slowdown with many results
- Recent searches history with localStorage persistence and intelligent deduplication for faster access to previous searches
- Fully responsive design that works seamlessly across different screen sizes adapting layout and interaction patterns accordingly
- Code splitting and dynamic imports for optimized initial loading performance reducing bundle size and improving startup speed
- Asynchronous operations with proper loading states and error handling ensuring smooth user experience without interface blocking
- Accessibility features with proper ARIA attributes and keyboard navigation supporting users with different accessibility needs

### Communication Layer (Wails)

Uses Wails framework to connect Go backend with Vue.js frontend with type safety and real-time capabilities, ensuring efficient and secure communication:

- **Generated TypeScript bindings**: Type-safe communication between Go and TypeScript preventing runtime errors and ensuring data integrity
- **Real-time event system**: Efficient progress updates and status notifications without blocking operations enabling smooth user experience
- **Cross-platform compatibility**: Native system integration across all platforms maintaining consistent functionality and user experience
- **Performance optimized**: Low-latency communication with efficient data serialization minimizing overhead and maximizing responsiveness
- **Error handling**: Comprehensive error propagation from backend to frontend ensuring proper error display and user feedback

## Installation

### Prerequisites

- Go 1.23 or higher with proper GOPATH configuration and available in system PATH
- Node.js 16.x or higher with npm package manager for frontend development
- Wails CLI: Install using `go install github.com/wailsapp/wails/v2/cmd/wails@latest` for project building and development
- System dependencies: Platform-specific tools for directory selection (zenity/kdialog/yad for Linux, PowerShell for Windows, AppleScript for macOS)

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
   This command downloads and verifies all required Go modules, ensuring consistent dependency management across environments.

3. Install frontend dependencies:

   ```bash
   cd frontend
   npm install
   ```
   This installs all necessary Node.js packages, including Vue.js, TypeScript, and other development dependencies.

4. Build the application:
   ```bash
   wails build
   ```
   This command compiles both the Go backend and Vue.js frontend into a single executable binary with all dependencies included.

The executable will be created in the `build/bin/` directory with a platform-specific name (e.g., `code-search-golang.exe` on Windows).

## Usage

### Basic Search

1. Click "Browse" to select a directory to search in - this opens a native system dialog for secure directory selection
2. Enter your search query in the "Search Query" field - supports plain text or regular expressions depending on your settings
3. Optionally specify a file extension to filter results (e.g., ".go" to search only Go files) for more targeted results
4. Configure search options as needed according to your specific search requirements and preferences
5. Click "Search Code" to begin the search - the application will display real-time progress while searching

### Code Preview Features

The application includes enhanced code preview capabilities:

- **Syntax Highlighting**: Code files are displayed with proper syntax highlighting based on file extension using highlight.js with the Agate theme for optimal readability
- **Line Numbers**: Each line is numbered for easy reference and precise location identification within the file
- **Search Match Highlighting**: Search terms are highlighted in yellow for easy identification of matches within the code context
- **Performance Optimized**: Large files are processed efficiently with a maximum limit of 10,000 lines to prevent browser crashes and maintain responsiveness
- **Navigation**: Navigate between search matches using the "Next Match" button for efficient code exploration and review
- **File Preview**: Click on any file in search results to open a detailed preview modal with full syntax highlighting and context
- **Readability**: Enhanced spacing between line numbers and code content for better readability and reduced eye strain during extended code review sessions

### Advanced Search Options

- **Case Sensitive**: Check this box for case-sensitive searches where uppercase and lowercase letters are treated as distinct characters
- **Regex Search**: Enable to use regular expressions in your search query for complex pattern matching and advanced search capabilities
- **Include Binary**: Include binary files in the search (disabled by default) with intelligent binary detection to identify non-text files
- **Search Subdirs**: Search in subdirectories (enabled by default) to recursively search through all nested directories within your selected directory
- **Max File Size**: Limit file size to include in search (default 10MB) to prevent performance issues with very large files
- **Max Results**: Limit number of results returned (default 1000) to manage large result sets and maintain UI performance
- **Min File Size**: Minimum file size to include in search to exclude very small files like temporary files or configuration snippets
- **File Type Allow-Lists**: Restrict searches to specific file extensions (e.g., only .go, .js, .py files) with UI controls for easy selection and security
- **Multi-part Extension Support**: Support for double extensions like .tar.gz, .min.js, .config.bak, etc. for more precise file type filtering
- **Exclude Patterns**: Multi-select dropdown to choose common patterns to exclude (e.g., node_modules, .git) or add custom patterns to filter unwanted directories

### Search Results

- Results display file path, line number, and matched content with full context for accurate identification and location
- Click on file path to open the containing folder in your system's file manager for quick navigation to the file location
- Use "Copy" button to copy matched lines to clipboard for easy sharing or further processing in other applications
- "Matched" text shows the actual text that matched your query with visual highlighting for immediate identification
- Results are paginated for better performance (10 per page) to maintain responsive UI even with large result sets
- Use pagination controls to navigate through results efficiently with clear indicators of total results and current position
- Results are limited to prevent performance issues while maintaining usability by managing the maximum number of results displayed

## Development

### Prerequisites

Before starting development, ensure you have the following installed:

- Go 1.23 or higher with proper GOPATH configuration and available in system PATH for Go module management
- Node.js 16.x or higher with npm package manager for frontend dependencies and build tools
- Wails CLI: Install using `go install github.com/wailsapp/wails/v2/cmd/wails@latest` for project building and development tools
- System dependencies for your platform (zenity/kdialog/yad for Linux desktop environments, PowerShell for Windows, AppleScript for macOS)

### Live Development

To run in live development mode with hot reloading:

```bash
wails dev
```

This starts a Vite development server with hot reload for frontend changes, automatically detecting and updating changes to Vue.js components and TypeScript code. The Go backend communicates with the frontend through Wails bindings. Changes to both Go and Vue.js code will automatically reload in the development application, though Go backend changes may require restarting the development server for changes to take effect.

### Testing Strategy

Comprehensive testing approach ensures code quality, security, and reliability with multiple layers of validation:

**Backend Tests:**

- Run all backend tests: `go test -v ./...` - executes all test files in the project with verbose output
- Execute specific test: `go test -v -run TestFunctionName` - runs only tests matching the specified name pattern
- Run tests with coverage: `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out` - generates coverage report in browser

**Frontend Tests:**

- Run all frontend tests: `cd frontend && npm test` - executes all Jest tests with configured settings
- Run tests in watch mode: `cd frontend && npm run test:watch` - automatically re-runs tests when files change
- Generate test coverage: `cd frontend && npm run test:coverage` - creates detailed coverage reports for TypeScript code

**Integration Tests:**

- Cross-platform functionality testing across Windows, Linux, and macOS environments
- End-to-end search workflow validation ensuring complete search functionality works as expected
- Performance and security testing to validate optimizations and protective measures

### Project Structure

```
├── main.go                 # Wails application entry point - initializes and starts the application
├── app_core.go             # Core application logic and search engine - contains main search functionality
├── app.go                  # Linux-specific system integration - handles Linux-specific functionality
├── appWindows.go           # Windows-specific system integration - handles Windows-specific functionality
├── app_test.go             # Core backend tests - includes unit and integration tests for main functions
├── *.go                    # Additional Go source files (binary_file_test.go, search_with_progress_test.go, etc.)
├── go.mod                  # Go module dependencies - defines module name and required dependencies
├── go.sum                  # Go module checksums - ensures dependency integrity
├── wails.json              # Wails configuration - defines application metadata, build settings, and frontend configuration
├── frontend/               # Vue.js frontend components - all frontend source code and assets
│   ├── src/
│   │   ├── main.ts        # Frontend entry point - initializes Vue application and Wails bindings
│   │   ├── App.vue        # Root Vue component - main application shell and router configuration
│   │   ├── assets/        # Static assets - images, icons, and other static resources
│   │   ├── components/    # Main components - primary application components
│   │   │   ├── CodeSearch.vue     # Main search orchestrator - coordinates search UI and logic
│   │   │   ├── StartupLoader.vue  # Initial loading state - shows loading indicators during startup
│   │   │   └── ui/                # UI components - reusable user interface elements
│   │   │       ├── CodeModal.vue      # File preview modal - displays file content with syntax highlighting
│   │   │       ├── ProgressIndicator.vue # Progress visualization - shows real-time search progress
│   │   │       ├── SearchForm.vue      # Search parameter form - handles all search input fields
│   │   │       └── SearchResults.vue   # Results display - shows and manages search results with pagination
│   │   ├── composables/   # Shared logic (useSearch composable) - centralized business logic and state management
│   │   ├── types/         # TypeScript interfaces - defines type definitions for all data structures
│   │   └── style.css      # Global styles - application-wide CSS styling
│   ├── tests/             # Jest unit tests and configuration - frontend testing setup and test files
│   ├── public/            # Static assets - files copied directly to output directory
│   ├── package.json       # NPM dependencies - lists all Node.js dependencies and scripts
│   ├── tsconfig.json      # TypeScript configuration - defines TypeScript compilation options
│   ├── vite.config.ts     # Vite build configuration - defines frontend build and development server settings
│   └── wailsjs/           # Generated Wails bindings - automatically generated TypeScript bindings for Go functions
├── build/                 # Build outputs - compiled executables and build artifacts
└── README.md              # Project documentation - comprehensive documentation for users and developers
```

### Key Architecture Components

#### Backend Components (Go)

- **App struct**: Main application managing context and search lifecycle, coordinating all backend operations and state management
- **SearchWithProgress()**: Core search engine with real-time progress, cancellation, and parallel processing using worker pools
- **ValidateDirectory()**: Comprehensive directory validation with security checks to prevent unauthorized access
- **SelectDirectory()**: Cross-platform native directory selection (Windows PowerShell, Linux zenity/kdialog/yad, macOS AppleScript*)
- **ShowInFolder()**: Secure file manager integration with path traversal protection to prevent malicious directory access
- **processFileLineByLine()**: Memory-efficient streaming processor for large files that prevents memory overflow
- **isBinary()**: Advanced binary file detection with multiple validation layers to identify and handle non-text files
- **SearchRequest/SearchResult**: Type-safe data structures for search operations ensuring data integrity and validation

#### Frontend Components (Vue.js)

- **CodeSearch.vue**: Main application orchestrator component that coordinates the entire search interface and workflow
- **SearchForm.vue**: Comprehensive form for search parameters with validation and recent search integration providing user-friendly input
- **SearchResults.vue**: Paginated results display with interactive features enabling efficient browsing of large result sets
- **ProgressIndicator.vue**: Real-time visual progress tracking showing file counts, percentage completion, and time estimates
- **CodeModal.vue**: Advanced file preview with syntax highlighting and match navigation for detailed code inspection
- **useSearch.ts**: Composition composable containing all search business logic, state management, and Wails integration
- **TypeScript interfaces**: Strong typing for all data structures (SearchRequest, SearchResult, etc.) ensuring type safety

### Development Best Practices

#### Go Backend Development

- **Context Usage**: Always use context for cancellation and timeout handling to ensure proper resource cleanup and operation termination
- **Error Handling**: Comprehensive error handling with meaningful messages that provide clear information for debugging and user feedback
- **Memory Management**: Efficient memory usage with streaming for large files to prevent memory overflow in resource-constrained environments
- **Concurrency**: Proper use of goroutines and channels for parallel processing while maintaining thread safety and resource management
- **Testing**: Write comprehensive unit and integration tests covering edge cases, error conditions, and security scenarios
- **Documentation**: Add godoc comments for all exported functions to maintain clear API documentation for team members

#### Vue.js Frontend Development

- **Composition API**: Use composables for shared logic and state management to promote code reusability and maintainability
- **Type Safety**: Leverage TypeScript for type-safe development preventing runtime errors and improving development experience
- **Performance**: Optimize rendering and component updates using virtual scrolling and efficient reactivity patterns
- **Accessibility**: Implement proper ARIA attributes and keyboard navigation to ensure the application is usable by all users
- **Component Design**: Create modular, reusable components with clear interfaces and separation of concerns
- **Testing**: Write comprehensive unit and integration tests covering component behavior and user interactions

#### Cross-Platform Considerations

- **System Integration**: Test directory selection and file operations across platforms to ensure consistent behavior on Windows, Linux, and macOS
- **UI Consistency**: Ensure consistent user experience across different OS by using platform-appropriate UI patterns and behaviors
- **Performance**: Optimize for different system capabilities considering varying CPU, memory, and storage performance on different systems
- **Build Process**: Verify builds work correctly on different platforms by testing compilation and execution on each target platform

### Build and Deployment

#### Development Build

- Use `wails dev` for development with hot reload to enable rapid iteration during development
- Frontend changes auto-reload in development mode without requiring application restart
- Go backend changes require restart of dev server for changes to take effect

#### Production Build

- Execute `wails build` to create production executables with all dependencies bundled into a single file
- Output executables will be in `build/bin/` directory with platform-specific names and formats
- Executables are self-contained with all necessary dependencies eliminating the need for runtime installations
- Builds are created for the host platform by default, requiring cross-compilation for different target platforms

## Performance Optimizations

### Backend Optimizations:

- **File size limits**: Configurable maximum file size (default 10MB) to prevent memory issues and performance degradation by avoiding processing of extremely large files that could overwhelm system resources
- **Result limits**: Adjustable result count limits (default 1000) to prevent overwhelming responses and maintain UI responsiveness by limiting the amount of data sent to the frontend
- **Parallel processing**: Efficient use of Go goroutines with dynamic worker pools sized to CPU cores for maximum performance while avoiding resource contention
- **Binary detection**: Intelligent skipping of binary files to reduce processing time and memory usage by identifying and excluding non-text files early in the search process
- **Memory-efficient streaming**: Large files (>1MB threshold) processed line-by-line to prevent memory issues using buffered readers that maintain low memory footprint
- **Early termination**: Context cancellation used to stop searches when max results reached, saving computation time and system resources while providing faster response times
- **File system optimization**: Efficient file traversal with intelligent filtering and exclusion patterns to minimize unnecessary file system operations and I/O
- **Buffer management**: Configurable buffer sizes (default 1MB) for optimal I/O performance balancing memory usage with read efficiency
- **Resource management**: Proper cleanup of file handles and memory during operations to prevent resource leaks and ensure long-running searches don't consume excessive system resources

### Frontend Optimizations:

- **Code splitting**: Strategic splitting of application components to reduce initial bundle size and improve load time by loading only necessary code for the initial view
- **Dynamic imports**: On-demand loading of syntax highlighting libraries and other heavy components to reduce initial load time and memory usage
- **Efficient rendering**: Optimized Vue.js reactivity system with virtual scrolling for large result sets to maintain smooth UI performance even with thousands of results
- **Pagination system**: Results segmented into pages (10 per page) to limit DOM elements and maintain performance by only rendering visible results
- **Large file handling**: Truncation limits (10,000 lines max) to prevent browser crashes and maintain responsiveness when previewing extremely large files
- **Asynchronous operations**: Non-blocking UI updates with proper loading states and progress indicators to maintain responsive interface during long-running operations
- **Memory management**: Efficient state management to prevent memory leaks and optimize garbage collection by properly disposing of unused data and component references
- **Bundle optimization**: Tree shaking and minification to reduce final bundle size, removing unused code and optimizing assets for faster loading

## Security Features

### Backend Security Measures:

- **Input Validation & Sanitization**: Comprehensive validation and sanitization of all user inputs, search queries, and file paths to prevent injection attacks and unauthorized access
- **Path Traversal Protection**: Robust protection against directory traversal attacks using filepath.Clean and multiple validation checks to ensure paths remain within intended directories
- **File Access Control**: File type allow-lists to restrict searches to specific extensions for enhanced security, preventing access to sensitive file types
- **File Path Sanitization**: All file paths sanitized before any system operations using Go's filepath package to prevent malicious path manipulation
- **Binary File Detection**: Prevention of processing of potentially dangerous binary files when not required using multiple detection algorithms to identify non-text content
- **Permission Validation**: Verification of file access permissions before any read operations to ensure the application respects system-level file permissions
- **Sandboxing**: Isolated search operations limited to specified directories only, preventing access to system or user directories outside the search scope

### Frontend Security Measures:

- **Content Security Policy**: Implementation of strict CSP to prevent injection attacks by controlling which resources can be loaded and executed
- **XSS Prevention**: Sanitization of all content before rendering to prevent cross-site scripting by filtering potentially dangerous HTML and JavaScript content
- **Input Sanitization**: Validation and sanitization of all user inputs before transmission to backend to prevent malicious payloads from reaching the server
- **Secure State Management**: Proper handling of sensitive data in application state to prevent information leakage through browser storage or memory
- **Trusted Types**: Ensuring only safe content is rendered in the browser using browser security features to prevent DOM-based XSS attacks

### Communication Security:

- **Type Safety**: Wails-generated TypeScript bindings ensure type-safe communication preventing runtime errors and data corruption during Go-Vue.js communication
- **Secure Event Handling**: Proper validation of all real-time events from backend to prevent malicious data injection through the event system
- **Data Integrity**: Protected communication channel between frontend and backend using Wails' secure communication layer to prevent data tampering

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
