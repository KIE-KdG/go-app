# go-webserver

This is the golang frontend web-app for the GeoAir project, to running it whole you also need this [python backend](https://gitlab.com/kdg-ti/the-lab/teams-24-25/k-nstliche-intelligenz-entwicklungsgruppe-charlemange/geo-ai-assistant): 

## Getting started

This web-application is displaying the frontend of the platform. It is written in Go and uses the [Tailwind CSS](https://tailwindcss.com/) framework with DaisyUI.
We are also using AlpineJS for some interactivity and dynamic UI components without requiring a full JavaScript framework. The application follows a modern Go web architecture with routing handled by httprouter and session management via SCS.

## Structure of the project

```bash
.
├── cmd                         # Command applications
│   ├── tmp                     # Temporary executables
│   │   └── main                # Temporary main executable
│   └── web                     # Main web application
│       ├── main.go             # Application entry point
│       ├── routes.go           # URL routing configuration
│       ├── middlewear.go       # HTTP middleware functions
│       ├── templates.go        # Template rendering logic
│       ├── helpers.go          # Helper functions
│       ├── apiHelpers.go       # API-specific helper functions
│       ├── authHandlers.go     # Authentication request handlers
│       ├── adminHandlers.go    # Admin panel request handlers
│       ├── chatHandlers.go     # Chat functionality handlers
│       ├── projectHandler.go   # Project management handlers
│       ├── schemaHandler.go    # Database schema handlers
│       ├── websocketHandlers.go # WebSocket communication handlers
│       └── externalClients.go  # External API client implementations
├── data                        # Application data
│   ├── dummy.geojson           # Sample GeoJSON data
│   ├── iRISExample.json        # Example JSON data
│   ├── sessions.db             # SQLite database for sessions
│   └── sqlite_lab.db           # Main SQLite database
├── go.mod                      # Go module definition
├── go.sum                      # Go module checksums
├── internal                    # Internal packages (not meant for external use)
│   ├── db                      # Database connection and management
│   │   ├── init.go             # Database initialization
│   │   ├── postgres.go         # PostgreSQL specific functionality
│   │   └── sqlite.go           # SQLite specific functionality
│   ├── fileprocessor           # File processing utilities
│   │   └── chunker.go          # File chunking for processing
│   ├── model                   # Data models for AI/ML components
│   │   ├── chat.go             # Chat model structures
│   │   └── ollama.go           # Ollama AI model integration
│   ├── models                  # Core application data models
│   │   ├── errors.go           # Custom error types and handling
│   │   ├── users.go            # User data models
│   │   ├── projects.go         # Project data models
│   │   ├── projectDatabase.go  # Project database operations
│   │   ├── chats.go            # Chat data models
│   │   ├── messages.go         # Message data models
│   │   ├── files.go            # File handling models
│   │   ├── geo.go              # Geospatial data models
│   │   └── schema.go           # Database schema models
│   ├── tui                     # Terminal UI components
│   │   ├── update.go           # TUI update logic
│   │   └── view.go             # TUI view rendering
│   └── validator               # Input validation
│       └── validator.go        # Validation logic
├── locales                     # Internationalization
│   ├── active.en.toml          # English translations
│   └── active.nl.toml          # Dutch translations
├── ui                          # User interface assets (not shown in your tree)
│   ├── html                    # HTML templates
│   ├── static                  # Static assets (CSS, JS, images)
│   └── js                      # JavaScript files
```

## Requirements

- Go 1.24
- SQLite3 for local development
- Node.js and npm for Tailwind CSS compilation
- ollama (if you use it as LLM provider)
  - Follow the tutorial [here](https://ollama.com/download)
- OBDC driver for mssql
    - See this [link](https://learn.microsoft.com/en-us/sql/connect/odbc/download-odbc-driver-for-sql-server?view=sql-server-ver16) for more information.
- Run the [python backend server](https://gitlab.com/kdg-ti/the-lab/teams-24-25/k-nstliche-intelligenz-entwicklungsgruppe-charlemange/geo-ai-assistant)

## Usage:

### Install dependencies

`npm i`

and:

`go mod tidy`

### Run the webserver

`go run ./cmd/web/`

#### And then open your browser and navigate to https://localhost:4000

You will also need to run the [python backend server](https://gitlab.com/kdg-ti/the-lab/teams-24-25/k-nstliche-intelligenz-entwicklungsgruppe-charlemange/geo-ai-assistant)

### For development, you can use live-reload

`go install github.com/air-verse/air@latest` to install

run `air` to start the live reload.

Also, you can run `npm run dev`

# License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details