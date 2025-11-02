# Code Search Golang

A powerful desktop code search application built with Wails (Go backend + Vue.js frontend). This application allows users to search for text patterns, keywords, and regular expressions across code files in specified directories.

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
- **Configurable Limits**: Adjustable maximum file size and result count limits
- **Subdirectory Search**: Toggle to search in subdirectories or only in the selected directory
- **Unicode Support**: Proper handling of Unicode characters in search queries and files
- **Performance Optimizations**: File size limits to prevent memory issues with large files
- **Result Truncation**: Automatic truncation to prevent overwhelming result sets
- **Search History**: Recent searches saved in local storage for quick access

### User Interface
- **Intuitive Design**: Clean, modern UI optimized for code search workflows
- **Real-time Feedback**: Shows search progress and result counts
- **Highlighted Matches**: Visual highlighting of search terms in results
- **Line Numbers**: Shows exact line numbers where matches were found
- **Copy Functionality**: Easy copying of matched lines to clipboard
- **Responsive Layout**: Works well on different screen sizes

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

### Advanced Options
- **Case Sensitive**: Check this box for case-sensitive searches
- **Regex Search**: Enable to use regular expressions in your search query
- **Include Binary**: Include binary files in the search (disabled by default)
- **Search Subdirs**: Search in subdirectories (enabled by default)
- **Max File Size**: Limit file size to include in search (default 10MB)
- **Max Results**: Limit number of results returned (default 1000)

### Search Results
- Results display file path, line number, and matched content
- Click on file path to open the containing folder
- Use "Copy" button to copy matched lines to clipboard
- "Matched" text shows the actual text that matched your query
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
│   │   └── components/ 
│   │       └── CodeSearch.vue  # Main search component
│   └── wailsjs/        # Generated Wails bindings
└── build/              # Build outputs
```

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
- On Windows: Directory selection dialog requires Windows API calls (not implemented in this version)

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
