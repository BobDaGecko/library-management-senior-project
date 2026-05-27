# Library Management System

A course-scale library management system with a Go backend, server-rendered UI, and SQLite storage. This repository contains the application code and the living documentation used for project delivery.

## Overview

- Core users: patrons, librarians, and admins.
- Goal: make catalog search, circulation workflows, and admin tasks simple and consistent.
- Docs: the full project documentation lives in the GitHub wiki and is used to generate the PDF handoff.

## Current status (snapshot)

- Auth and login flow are implemented.
- Admin and management templates are in progress.
- Circulation endpoints and UI stubs are still pending.

## Features

Implemented or partially implemented:

- User registration and login.
- Management pages for adding books, viewing users, transactions, and overdue items.
- Google Books integration for catalog metadata search.
- SQLite persistence with Gorm models.

Planned or in progress:

- Patron search and book detail pages.
- Account dashboard (loans, holds, fines, saved items).
- Full circulation flows (checkout, return, holds, fines).
- Staff picks and blog content.

## Tech stack

- Go (web server and application logic).
- Templ (server-rendered components).
- HTMX (partial updates).
- Semtx (semantic CSS library with native HTMX integration).
- Gorm + SQLite (data persistence).
- Google Books API (catalog seeding).

## Local development

Requirements:

- Go 1.26+.
- Templ CLI on your PATH.

Install dependencies:

```bash
go install github.com/a-h/templ/cmd/templ@latest
go mod download
```

Run the app (generates templ output automatically):

```bash
make run
```

Run with an optional Google Books API key:

```bash
go run main.go -gapi-token YOUR_TOKEN_HERE
```

Default address and port are `0.0.0.0:3000`. You can override them:

```bash
go run main.go -addr 127.0.0.1 -port 3000
```

## Tests

```bash
make test
```

Or run suites individually:

```bash
go test ./db
go test ./fetch
```

## Data and storage

- Local database file: `senior-library.db` in the repo root.
- Delete the file to reset local data and regenerate tables.

## Documentation

- Wiki entry point: [wiki/Home](wiki/Home)
- Architecture: [wiki/3.0-architecture](wiki/3.0-architecture)
- Data and storage: [wiki/4.0-data-and-storage](wiki/4.0-data-and-storage)
- UX and UI: [wiki/8.0-ux-ui.md](wiki/8.0-ux-ui)

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).

## Contributors

| Name | GitHub | Primary roles |
| --- | --- | --- |
| Kellen Siczka | @BobDaGecko | Scrum Master / PM, Repo Maintainer, Technical Writing |
| Daniel Petrov | @VoxelPrismatic | Backend Developer, Repo Maintainer |
| Alvar Kandikatla | @AlvarWolf75 | Quality Assurance / Testing |
| Dimitar Dimitrov | @nightfall2303 | UI/UX Design, Frontend Developer |
| Gauri Aklujkar | @GAkl224 | UI/UX Design, Frontend Developer |
| Ahnna Williams | @idkman9411 | UI/UX Design, Frontend Developer |
| Allen Vives | @spareought | Business Analysis, Technical Writing |
