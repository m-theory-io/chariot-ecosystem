# Charioteer

A web-based code editor for the Chariot programming language.

## Features

- **Monaco Editor Integration**: Full-featured code editor with syntax highlighting
- **Chariot Language Support**: Custom syntax highlighting and tokenization for Chariot
- **File Management**: Load, save, save as, rename, and delete Chariot files (.ch)
- **Authentication**: Secure login/logout functionality
- **Code Execution**: Execute Chariot code via API integration with Chariot runtime server
- **Responsive UI**: Modern, responsive interface that works on various screen sizes
- **Real-time Feedback**: Output panel with execution results and error messages

## Requirements

- Go 1.19 or later
- A running Chariot runtime server (default: localhost:8087) for code execution and authentication

## Configuration

Charioteer can be configured using command line flags or environment variables:

### Backend Server
- **Flag**: `-backend=<URL>`
- **Environment**: `CHARIOT_BACKEND_URL=<URL>`
- **Default**: `http://localhost:8087`

### Web Server Port
- **Flag**: `-port=<PORT>`
- **Environment**: `CHARIOT_PORT=<PORT>`
- **Default**: `8080`

### Request Timeout
- **Flag**: `-timeout=<SECONDS>`
- **Environment**: `CHARIOT_TIMEOUT=<SECONDS>`
- **Default**: `30`

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/bhouse1273/charioteer.git
   cd charioteer
   ```

2. Build the application:
   ```bash
   go build -o charioteer main.go
   ```

3. Run the server:
   ```bash
   # Using defaults
   ./charioteer
   
   # Using command line flags
   ./charioteer -backend=http://my-chariot-server:8087 -port=3000 -timeout=60
   
   # Using environment variables
   export CHARIOT_BACKEND_URL=http://my-chariot-server:8087
   export CHARIOT_PORT=3000
   export CHARIOT_TIMEOUT=60
   ./charioteer
   ```

4. Open your browser and navigate to:
   ```
   http://localhost:8080/editor
   ```
   Or the configured port if different.

## Usage

1. **Login**: Use your Chariot credentials to log in
2. **File Management**: 
   - Select files from the dropdown to load them
   - Save changes to existing files
   - Save as new files with different names
   - Rename existing files
   - Delete files (with confirmation)
3. **Code Editing**: Write Chariot code with full syntax highlighting
4. **Code Execution**: Run your Chariot programs and see results in the output panel

## Project Structure

- `main.go` - Main server application with embedded HTML/CSS/JavaScript
- `files/` - Directory containing Chariot source files (.ch)
- `go.mod` - Go module definition

## API Integration

Charioteer acts as a proxy/frontend for a Chariot runtime server that should be running on `localhost:8087`. The runtime server handles:
- User authentication (`/login`, `/logout`)
- Code execution (`/api/execute`)

## Development

The application is a single Go file that serves both the web interface and API endpoints. The frontend uses:
- Monaco Editor for code editing
- Custom Chariot language tokenizer
- Responsive CSS design
- Vanilla JavaScript (no external frameworks)

## Security

- All file operations are restricted to the `files/` directory
- Path traversal protection prevents access to files outside the allowed directory
- Authentication required for all file operations and code execution
- CORS headers configured for cross-origin requests
