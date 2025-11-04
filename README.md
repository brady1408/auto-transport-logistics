# Auto Transport Logistics

A CRUD web application for managing transportation logistics for auto transport companies, built with the GOTTH stack (Go + Templ + Tailwind + HTMX).

## Tech Stack

- **Backend:** Go 1.22+ (stdlib routing), PostgreSQL, SQLC
- **Frontend:** Templ, HTMX, Tailwind CSS, Alpine.js
- **Development:** Air (live reload), Docker Compose

## Prerequisites

- Go 1.22 or higher
- Node.js 18+ and npm
- Docker and Docker Compose
- PostgreSQL client tools (optional)

## Quick Start

### 1. Install Go Tools

```bash
# Install required Go tools
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/a-h/templ/cmd/templ@latest
go install github.com/cosmtrek/air@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Make sure `$GOPATH/bin` is in your PATH (usually `~/go/bin`).

### 2. Install Dependencies

```bash
# Install Node dependencies (Tailwind)
npm install

# Install Go dependencies
go mod download
```

### 3. Start PostgreSQL

```bash
# Start PostgreSQL with Docker Compose
docker-compose up -d

# Verify it's running
docker-compose ps
```

The database will be available at `localhost:5432` with:
- Database: `auto_transport`
- Username: `postgres`
- Password: `postgres`

### 4. Generate Code

**Note:** Migrations are embedded in the binary and run automatically on startup. No separate migration step needed!

```bash
# Generate SQLC code
sqlc generate

# Generate Templ templates
templ generate

# Build Tailwind CSS
npm run build:css
```

### 6. Run the Application

**Option A: With live reload (recommended for development)**

```bash
air
```

**Option B: Direct run**

```bash
go run cmd/server/main.go
```

The application will be available at `http://localhost:8080`

## Development Workflow

### Watch Mode

For the best development experience, run these in separate terminals:

```bash
# Terminal 1: Watch Tailwind CSS
npm run watch:css

# Terminal 2: Run server with live reload
air
```

Now you can edit `.templ` files, Go code, and see changes automatically.

### Adding a New Feature

1. **Database changes:**
   ```bash
   # Create migration files in internal/database/migrations/
   # Name them sequentially: 000002_description.up.sql and 000002_description.down.sql

   # Migrations run automatically on next server start
   # OR you can restart the server to apply them immediately
   go run cmd/server/main.go
   ```

2. **Write SQL queries:**
   - Add queries to `sql/queries/*.sql`
   - Run `sqlc generate` to create Go code

3. **Create templates:**
   - Add `.templ` files in `templates/`
   - Run `templ generate`

4. **Implement handlers:**
   - Add handlers in `internal/handlers/`
   - Wire routes in `cmd/server/main.go`

5. **Test in browser**

## Project Structure

```
/
├── cmd/server/           # Application entry point
├── internal/
│   ├── handlers/         # HTTP handlers
│   ├── repository/       # SQLC generated code
│   └── services/         # Business logic
├── templates/            # Templ templates
├── static/              # CSS, JS, images
├── migrations/          # Database migrations
├── sql/
│   ├── schema.sql       # Database schema
│   └── queries/         # SQLC queries
└── .claude/             # Project documentation
```

## Database Management

**Migrations are embedded in the binary** and run automatically when the server starts. No separate migration tool needed for deployment!

### How it works

- Migration files are in `internal/database/migrations/`
- Embedded using Go's `embed` package
- Automatically applied on server startup
- Single binary contains everything (migrations + application code)

### Creating a new migration

1. Create two files in `internal/database/migrations/`:
   - `000002_description.up.sql` - The migration
   - `000002_description.down.sql` - The rollback

2. Restart the server to apply:
   ```bash
   go run cmd/server/main.go
   # OR with Air
   air
   ```

### Manual migration management (optional)

If you still want to use the `migrate` CLI tool for development:

```bash
# The old migrations/ folder is kept for reference
# You can still use migrate CLI if needed:
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/auto_transport?sslmode=disable" up
```

### Connect to database

```bash
docker-compose exec postgres psql -U postgres -d auto_transport
```

## Building for Production

```bash
# Generate all code
templ generate
sqlc generate
npm run build:css

# Build binary
go build -o bin/server cmd/server/main.go

# Run
./bin/server
```

## Environment Variables

Create a `.env` file for local development:

```bash
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/auto_transport?sslmode=disable
PORT=8080
ENV=development
```

## Available Commands

### Development
- `air` - Run with live reload
- `npm run watch:css` - Watch and rebuild Tailwind CSS
- `templ generate` - Generate Go code from Templ templates
- `sqlc generate` - Generate Go code from SQL queries

### Database
- `docker-compose up -d` - Start PostgreSQL
- `docker-compose down` - Stop PostgreSQL
- `docker-compose down -v` - Stop and remove data volumes

### Build
- `npm run build:css` - Build Tailwind CSS for production
- `go build -o bin/server cmd/server/main.go` - Build binary

## Core Features (Planned)

- Shipment management (CRUD)
- Vehicle tracking
- Customer management
- Carrier management
- Route planning
- Status updates
- Real-time tracking (future)

## Documentation

See `.claude/claude.md` for detailed technical documentation, architecture decisions, and development guidelines.

## License

MIT
