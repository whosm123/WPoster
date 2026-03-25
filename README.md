# WPoster

A CLI tool for posting content to various platforms.

## Features

- Post content from command line
- Simple and intuitive interface
- Configurable platform support

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/wposter.git
cd wposter

# Build the binary
make build

# Install globally
make install
```

## Usage

```bash
# Show help
wposter --help

# Post content
wposter post "Hello, world!"

# Show version
wposter version
```

## Development

```bash
# Run tests
make test

# Build binary
make build

# Run in development mode
make dev
```

## Project Structure

```
.
├── cmd/           # Command implementations
│   ├── root.go    # Root command
│   └── post.go    # Post command
├── main.go        # Entry point
├── go.mod         # Go module file
├── Makefile       # Build commands
└── README.md      # This file
```

## License

MIT