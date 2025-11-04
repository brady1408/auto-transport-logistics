# Auto Transport Logistics

A CRUD web application for managing transportation logistics for auto transport companies.

## Tech Stack (GOTTH)

### Backend
- **Go 1.22+** - Using stdlib `net/http` for routing (no Chi needed)
- **PostgreSQL** - Primary database
- **SQLC** - Type-safe SQL code generation
- **golang-migrate** - Database migrations
- **pgx/v5** - PostgreSQL driver (used by SQLC)

### Frontend
- **Templ** - Type-safe Go templates (compiles to Go code)
- **HTMX** - Server-driven interactivity without heavy JavaScript
- **Tailwind CSS** - Utility-first CSS framework
- **Alpine.js** - Minimal JavaScript for client-side interactions (used sparingly)

### Development Tools
- **Air** - Live reload during development
- **Docker Compose** - Local PostgreSQL instance

## Multitenancy Architecture

This application is **multi-tenant** using **row-level isolation** (shared database, shared schema).

### Approach
- **Organizations table** - Represents each transport company using the system
- **Users table** - Each user belongs to ONE organization
- **All data tables** have `organization_id` foreign key (customers, carriers, shipments)
- **Session-based isolation** - organization_id stored in session after login
- **Application-level filtering** - All queries include `WHERE organization_id = $X`
- **Composite indexes** - (organization_id, other_field) for optimal query performance

### Why NOT Row-Level Security (RLS)?
We're **skipping PostgreSQL RLS** for the MVP to keep things simple:
- Avoids complexity of passing session variables to PostgreSQL
- Easier debugging (can see queries directly)
- Can add RLS later if needed as an extra safety layer

### Data Isolation Rules
- Users can ONLY see data from their organization
- organization_id is passed through session/JWT to all queries
- Foreign keys ensure data integrity
- NOT NULL constraints prevent missing organization_id

### Security Considerations
- **Always include organization_id in WHERE clauses** (critical!)
- Middleware extracts organization_id from session
- Validate user's organization_id matches requested resource
- Use prepared statements (SQLC handles this)

## Project Structure

```
/
├── .claude/              # Claude context and documentation
│   └── claude.md         # This file
├── cmd/
│   └── server/
│       └── main.go       # Application entry point
├── internal/
│   ├── handlers/         # HTTP request handlers
│   ├── repository/       # SQLC generated code (DB layer)
│   └── services/         # Business logic layer
├── templates/            # Templ template files (.templ)
├── static/
│   ├── css/             # Tailwind CSS (input.css → output.css)
│   └── js/              # HTMX, Alpine.js, custom JS
├── migrations/          # Database migration files
├── sql/
│   ├── schema.sql       # Database schema DDL
│   └── queries/         # SQLC query files (.sql)
├── tools/               # Build scripts and utilities
├── go.mod               # Go module definition
├── sqlc.yaml           # SQLC configuration
├── tailwind.config.js  # Tailwind configuration
├── docker-compose.yml  # Local dev PostgreSQL
└── .air.toml           # Air live reload config
```

## Core Domain Entities

**Note:** All entities except Organizations have `organization_id` for multi-tenant isolation.

### 1. Organizations
The top-level tenant entity. Each organization is a separate auto transport company.

**Fields:**
- ID (UUID)
- Name
- Slug (unique, for URLs if needed)
- Active status
- Created/Updated timestamps

### 2. Users
Users who log in to the system. Each user belongs to ONE organization.

**Fields:**
- ID (UUID)
- Organization ID (FK) - **CRITICAL: defines data access scope**
- Email (unique)
- Password Hash
- First Name, Last Name
- Role (admin, user)
- Active status
- Created/Updated timestamps

### 3. Shipments
Primary entity representing an auto transport job.

**Fields:**
- ID (UUID)
- **Organization ID (FK)** - Multi-tenant isolation
- Customer ID (FK)
- Carrier ID (FK, nullable)
- Pickup Location (address fields)
- Delivery Location (address fields)
- Status (enum: pending, assigned, in_transit, delivered, cancelled)
- Pickup Date (requested & actual)
- Delivery Date (estimated & actual)
- Price (decimal)
- Notes (text)
- Created/Updated timestamps

### 4. Vehicles
Vehicles being transported in shipments.

**Fields:**
- ID (UUID)
- Shipment ID (FK)
- Make
- Model
- Year
- VIN
- Color
- Condition (enum: running, non-running, damaged)
- Notes

### 5. Customers
Individuals or companies requesting transport services.

**Fields:**
- ID (UUID)
- **Organization ID (FK)** - Multi-tenant isolation
- Name (company or individual)
- Contact Name
- Email
- Phone
- Address fields
- Created/Updated timestamps

### 6. Carriers
Transport companies/drivers performing shipments.

**Fields:**
- ID (UUID)
- **Organization ID (FK)** - Multi-tenant isolation
- Company Name
- Contact Name
- Email
- Phone
- MC Number (Motor Carrier number)
- DOT Number
- Insurance Info
- Active status
- Created/Updated timestamps

### 7. Routes (Future - can add later for real-time tracking)
- ID
- Shipment ID
- Current Location
- ETA
- Stops/Waypoints

## Embedded Migrations

**Migrations are embedded in the binary** for zero-dependency deployment.

### How it Works
- Migration files located in `internal/database/migrations/`
- Embedded using Go's `embed` package (`//go:embed` directive)
- Automatically run on server startup via `database.RunMigrations()`
- Uses `golang-migrate/migrate` with `iofs` source driver
- Single binary deployment (no separate .sql files needed)

### Creating New Migrations
1. Create files in `internal/database/migrations/`:
   - `000002_description.up.sql` - Forward migration
   - `000002_description.down.sql` - Rollback migration
2. Restart server - migrations apply automatically
3. Update `sql/schema.sql` to reflect current state (for reference)

### Available Functions
```go
// Run all pending migrations
database.RunMigrations(databaseURL)

// Rollback last migration
database.RollbackMigration(databaseURL)

// Get current migration version
database.MigrationVersion(databaseURL)
```

## Development Workflow

### 1. Database Changes
```bash
# 1. Create new migration files in internal/database/migrations/
# Example: 000002_add_notes_table.up.sql and .down.sql

# 2. Write SQL in migration files

# 3. Restart server (migrations run automatically)
go run cmd/server/main.go
# OR
air

# 4. Update sql/schema.sql with current schema (optional, for reference)

# 5. Write queries in sql/queries/*.sql

# 6. Generate Go code
sqlc generate
```

### 2. Adding New Features
```bash
# 1. Write SQL queries in sql/queries/
# 2. Run sqlc generate
# 3. Create/update Templ templates in templates/
# 4. Run templ generate
# 5. Implement handlers in internal/handlers/
# 6. Wire up routes in cmd/server/main.go
# 7. Test in browser
```

### 3. Running Locally
```bash
# Start PostgreSQL
docker-compose up -d

# Install dependencies (first time)
npm install
go mod download

# Build Tailwind CSS
npm run build:css
# OR watch mode
npm run watch:css

# Run with live reload
air

# OR run directly
templ generate
go run cmd/server/main.go
```

### 4. Building for Production
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

## Routing Conventions

Using Go 1.22+ stdlib routing with method-based routing:

```go
// RESTful resource routing
mux.HandleFunc("GET /shipments", handlers.ListShipments)
mux.HandleFunc("GET /shipments/{id}", handlers.GetShipment)
mux.HandleFunc("POST /shipments", handlers.CreateShipment)
mux.HandleFunc("PUT /shipments/{id}", handlers.UpdateShipment)
mux.HandleFunc("DELETE /shipments/{id}", handlers.DeleteShipment)

// HTMX partial endpoints (return HTML fragments)
mux.HandleFunc("GET /shipments/{id}/edit", handlers.EditShipmentForm)
mux.HandleFunc("POST /shipments/{id}/assign", handlers.AssignCarrier)
```

## HTMX Patterns

### List with inline editing
```html
<!-- List item with edit button -->
<tr id="shipment-{id}" hx-target="this" hx-swap="outerHTML">
  <td>{data}</td>
  <td>
    <button hx-get="/shipments/{id}/edit">Edit</button>
  </td>
</tr>

<!-- Edit endpoint returns editable form -->
<tr id="shipment-{id}">
  <td><input name="field" value="{data}"></td>
  <td>
    <button hx-put="/shipments/{id}" hx-target="#shipment-{id}">Save</button>
    <button hx-get="/shipments/{id}" hx-target="#shipment-{id}">Cancel</button>
  </td>
</tr>
```

### Modal forms
```html
<button hx-get="/shipments/new" hx-target="#modal">New Shipment</button>

<div id="modal"></div>
```

### Live search/filter
```html
<input
  type="search"
  name="q"
  hx-get="/shipments/search"
  hx-trigger="keyup changed delay:300ms"
  hx-target="#results">

<div id="results"></div>
```

## Code Style

### Handler Pattern
```go
func ListShipments(w http.ResponseWriter, r *http.Request) {
    // 1. Parse query params
    // 2. Call service/repository
    // 3. Render template
    // 4. Handle errors appropriately
}
```

### Repository Pattern (SQLC)
- Use SQLC generated code directly in handlers for simple CRUD
- Create service layer for complex business logic
- Keep handlers thin

### Template Organization
```
templates/
  ├── layouts/
  │   └── base.templ      # Base HTML structure
  ├── components/
  │   ├── nav.templ       # Reusable components
  │   └── table.templ
  └── pages/
      ├── shipments/
      │   ├── list.templ
      │   ├── detail.templ
      │   └── form.templ
      └── home.templ
```

## Environment Configuration

```bash
# .env file (not committed)
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/auto_transport?sslmode=disable
PORT=8080
ENV=development
```

## Real-time Features (Future)

Can be added later using:
- **Server-Sent Events (SSE)** with HTMX extensions for live updates
- **WebSockets** using `gorilla/websocket` for bi-directional communication
- Location tracking updates
- Status change notifications

## Deployment Strategy

**Single binary deployment** - Migrations are embedded, no separate files needed!

**Simple VPS deployment:**
1. Build binary: `go build -o bin/server cmd/server/main.go`
2. SCP single binary to VPS (that's it - migrations are embedded!)
3. Set `DATABASE_URL` environment variable
4. Run as systemd service
5. Nginx reverse proxy on port 80/443
6. PostgreSQL on same VPS or managed service
7. SSL with Let's Encrypt

**On first run**, migrations apply automatically. No separate migration step!

**Docker deployment (alternative):**
```dockerfile
# Multi-stage build
FROM golang:1.22 AS builder
# Build Go binary

FROM alpine:latest
# Copy binary and static assets
# Run
```

## Testing Strategy

- **Unit tests** for services/business logic
- **Integration tests** for repository/database layer
- **E2E tests** (optional) for critical user flows
- Manual testing for MVP

## Security Considerations

- [ ] CSRF protection for forms
- [ ] Input validation and sanitization
- [ ] SQL injection prevention (SQLC handles this)
- [ ] Authentication/Authorization (JWT, sessions)
- [ ] HTTPS in production
- [ ] Rate limiting
- [ ] Secure headers

## Performance Considerations

- Pagination for large lists
- Database indexes on frequently queried fields
- Connection pooling (pgx handles this)
- Static asset caching
- GZIP compression

## Dependencies to Install

```bash
# Go tools
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/a-h/templ/cmd/templ@latest
go install github.com/cosmtrek/air@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Node/Tailwind
npm install

# Go packages (will be added as needed)
go get github.com/jackc/pgx/v5
go get github.com/google/uuid
```

## Next Steps After Setup

1. Create initial database schema and migrations
2. Generate SQLC code for first entity (Shipments)
3. Create base Templ layout and components
4. Implement first CRUD resource (Shipments)
5. Add authentication/authorization
6. Deploy MVP

## Resources

- [HTMX Documentation](https://htmx.org/docs/)
- [Templ Documentation](https://templ.guide/)
- [SQLC Documentation](https://docs.sqlc.dev/)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)
- [Go net/http Documentation](https://pkg.go.dev/net/http)
