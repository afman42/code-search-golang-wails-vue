# Code Search Golang

A powerful, feature-rich desktop code search application built with Wails (Go backend + Vue.js frontend). This application allows users to search for text patterns, keywords, and regular expressions across code files in specified directories with advanced filtering, security, and performance optimizations.

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
- **File Type Allow-Lists**: Restrict searches to specific file extensions (e.g., only .go, .js, .py files)
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

### Live Development
To run in live development mode with hot reloading:
```bash
wails dev
```

This starts a Vite development server with hot reload for frontend changes. The Go backend communicates with the frontend through Wails bindings.

### Testing
Run backend tests:
```bash
go test -v
```

Run frontend tests:
```bash
cd frontend
npm test
```

### Project Structure
```
├── app.go              # Backend Go application logic
├── main.go             # Wails application entry point
├── frontend/           # Vue.js frontend components
│   ├── src/
│   │   ├── components/     # Main components
│   │   │   └── CodeSearch.vue  # Main search component
│   │   ├── composables/    # Shared logic (useSearch composable)
│   │   ├── types/          # TypeScript interfaces
│   │   └── ui/             # UI components (SearchForm, SearchResults, etc.)
│   ├── tests/              # Jest unit tests
│   │   ├── unit/           # Individual component/composable tests
│   │   └── setup.ts        # Test setup with Wails mocks
│   └── wailsjs/            # Generated Wails bindings
└── build/                  # Build outputs
```

### Key Architecture Components

#### Frontend Components
- **CodeSearch.vue**: Main application component
- **SearchForm.vue**: Handles all search parameters and options
- **SearchResults.vue**: Displays results with pagination
- **ProgressIndicator.vue**: Shows search progress in real-time
- **CodeModal.vue**: File preview with syntax highlighting and match navigation
- **useSearch.ts**: Composition composable with all search logic

#### Backend Components  
- **App struct**: Main backend application
- **SearchWithProgress()**: Core search functionality with real-time progress updates
- **ValidateDirectory()**: Directory validation with security checks
- **SelectDirectory()**: Native directory selection across platforms (including Windows PowerShell implementation)
- **ShowInFolder()**: Open files in system file manager with path traversal protection
- **processFileLineByLine()**: Memory-efficient streaming for large files
- **isBinary()**: Binary file detection with multiple safety checks

## Architecture & Performance

### Backend Architecture
- **Parallel Processing**: Uses Go goroutines for concurrent file processing
- **Memory-Efficient Streaming**: Large files are processed line-by-line to prevent memory issues
- **Early Termination**: Search stops when max results are reached using context cancellation
- **Security Hardening**: Path traversal protection, input validation, and file type allow-lists
- **Cross-Platform**: Native directory selection for Windows, macOS, and Linux

### Frontend Architecture
- **Code Splitting**: Application is split into smaller chunks for faster loading
- **Dynamic Imports**: Syntax highlighting libraries loaded on-demand
- **Async Operations**: Non-blocking UI updates with proper loading states
- **Responsive Design**: Works seamlessly across different screen sizes
- **Vue 3 Composition API**: Modern component architecture with reusable business logic

### Performance Optimizations
- **File Size Limits**: Prevent memory issues with large files (default 10MB)
- **Result Limits**: Prevent overwhelming result sets (default 1000 results)
- **Parallel Processing**: Multiple files processed simultaneously using goroutines
- **Binary Detection**: Skip binary files when not needed
- **Streaming for Large Files**: Files >1MB processed line-by-line to conserve memory
- **Chunked Bundling**: Frontend assets split into smaller chunks for faster loading
- **Dynamic Syntax Highlighting**: Only load languages needed for the current file

## Security Features
- **Input Validation**: All user inputs are validated and sanitized
- **Path Traversal Protection**: Prevents directory traversal attacks in file operations
- **File Type Allow-Lists**: Restrict searches to specific file extensions for security
- **File Path Sanitization**: All file paths are sanitized before system operations

## Configuration

Edit `wails.json` to configure project settings:
- Application name and executable filename
- Window dimensions and properties
- Frontend build settings
- Development server configuration

## Troubleshooting

### Search Returns No Results
1. Verify the directory path is correct
2. Check the search query (typos occur)
3. Ensure file extensions match if using extension filtering
4. Verify files aren't larger than the maximum file size limit
5. Check case sensitivity settings

### Performance Issues
- The application limits file size to 10MB by default
- Result count is limited to 1000 by default
- Adjust these limits in the UI if needed

### Directory Selection Issues
- On Linux: Ensure you have one of the required tools installed (zenity, kdialog, yad)
- On Windows: Directory selection now uses PowerShell for native experience

## Contributing

Feel free to submit issues or pull requests. When submitting code changes:
1. Ensure all tests pass
2. Follow the existing code style
3. Update documentation as needed
4. Add tests for new functionality

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [Wails](https://wails.io/) framework
- Uses Vue 3 and TypeScript for the frontend
- Leverages Go for high-performance file system operations