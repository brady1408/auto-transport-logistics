# Linux Quick Start Guide

## What to copy to Linux

Copy the entire `Current ATA for Web Reference` folder. You need:
- `AutoStrap_DB.bak` - MSSQL backup for data migration
- `Stargazer_Source/Dev/ATLinks/SQLATLinks/` - Source files for reference
- `atlinks-migration/` - This folder (plan + data dictionary)

## Prerequisites (Ubuntu/Debian)

```bash
# Go 1.22+
sudo rm -rf /usr/local/go
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# Docker (for PostgreSQL and MSSQL)
sudo apt-get update
sudo apt-get install -y docker.io docker-compose-v2
sudo usermod -aG docker $USER
# Log out and back in for docker group

# Git
sudo apt-get install -y git
```

## First Steps on Linux

```bash
# 1. Start PostgreSQL
docker run -d --name atlinks-pg \
  -e POSTGRES_DB=atlinks \
  -e POSTGRES_USER=atlinks \
  -e POSTGRES_PASSWORD=atlinks_dev \
  -p 5432:5432 \
  postgres:16

# 2. Start MSSQL (for data migration reference)
docker run -d --name atlinks-mssql \
  -e 'ACCEPT_EULA=Y' \
  -e 'SA_PASSWORD=ATLinks2024!' \
  -p 1433:1433 \
  mcr.microsoft.com/mssql/server:2019-latest

# 3. Restore MSSQL backup
# Copy the .bak file into the container first:
docker cp /path/to/AutoStrap_DB.bak atlinks-mssql:/tmp/
docker exec -it atlinks-mssql /opt/mssql-tools/bin/sqlcmd \
  -S localhost -U SA -P 'ATLinks2024!' \
  -Q "RESTORE DATABASE ATLinks FROM DISK='/tmp/AutoStrap_DB.bak' WITH MOVE 'ATLinks' TO '/var/opt/mssql/data/ATLinks.mdf', MOVE 'ATLinks_log' TO '/var/opt/mssql/data/ATLinks_log.ldf'"

# 4. Initialize Go project
mkdir -p ~/projects/atlinks
cd ~/projects/atlinks
go mod init github.com/yourorg/atlinks

# 5. Install Go dependencies
go get github.com/jackc/pgx/v5
go get github.com/pressly/goose/v3
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt

# 6. Continue with Claude Code from here!
```

## What We've Completed So Far

- [x] Explored all source files in SQLATLinks/
- [x] Extracted complete data dictionary from ATLinks.TXD (all ~77 tables, all columns, all keys)
- [x] Documented full Clarion-to-PostgreSQL type mapping
- [x] Created comprehensive migration plan with 4 phases
- [x] Identified core table mapping (Clarion shorthand -> readable names)
- [x] Documented MVP scope vs post-MVP scope

## What's Next (Phase 0 continues)

1. Init Git repo + Go module + project directory structure
2. Write `001_initial_schema.up.sql` (use DATA_DICTIONARY.md as reference)
3. Write `002_seed_lookups.up.sql`
4. Build Go foundation: config, database pool, auth, audit, HTTP server
5. Start Phase 1: Foundation / Global Masters CRUD

## Key Files for Reference

When building the schema and app, refer to these source files:
- `DATA_DICTIONARY.md` - Complete schema reference (THIS IS THE PRIMARY REFERENCE)
- `PLAN.md` - Full implementation plan
- Original source (in the Stargazer_Source directory):
  - `ATLinks.TXD` - Raw Clarion data dictionary
  - `Trigger_UpdD00.clw` - Shows audit pattern and all D00 columns
  - `frame_atlinks_main.clw` - Menu structure / navigation blueprint
  - `ATLINKS.txa` - Full application export (all procedures/logic)
  - `ATLINKS.clw` - Main source with global declarations
  - `ATLSys.ini` - Connection strings, config
  - `Restore from BAK.sql` - MSSQL restore script
