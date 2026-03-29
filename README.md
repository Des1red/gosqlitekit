# sqlitekit

A small, simple SQLite initialization and migration helper for Go.

### sqlitekit provides

* SQLite connection initialization
* Safe pragmas (`WAL`, foreign keys, busy timeout)
* Ordered SQL migrations
* Append-only migration tracking with per-file and per-statement metadata
* Safe additive updates inside existing migration files
* Loud rejection of incompatible edits to previously applied statements
* Colored CLI output with status icons
* Supports filesystem and embedded migrations
* No ORM, no query builder — just a lightweight infrastructure layer

### Installation

```go
go get github.com/Des1red/gosqlitekit
```

## Usage

### Filesystem migrations

Initialize the database and run migrations from a directory.

```go
package main

import (
	"database/sql"
	"log"

	"github.com/Des1red/gosqlitekit/sqlitekit"
)

var DB *sql.DB

func InitDb() {
	err := sqlitekit.Initialize(
		"data/app.db",
		"internal/database/schema",
	)
	if err != nil {
		log.Fatal(err)
	}

	DB = sqlitekit.DB()
}
```

### Embedded migrations

You can embed migrations directly into the binary using Go's `embed` package.

```go
package main

import (
	"embed"
	"log"

	"github.com/Des1red/gosqlitekit/sqlitekit"
)

//go:embed schema/*.sql
var migrations embed.FS

func main() {
	err := sqlitekit.InitializeEmbedded(
		"data/app.db",
		migrations,
	)
	if err != nil {
		log.Fatal(err)
	}
}
```

This allows distributing a single binary without needing a schema folder.

## Configuration

sqlitekit provides a configurable database setup with safe defaults.

Default configuration:

```go
sqlitekit.Config{
	WAL:          true,
	ForeignKeys:  true,
	MaxOpenConns: 1,
	MaxIdleConns: 1,
}
```

You can override the configuration before initialization:

```go
package main

import (
	"database/sql"
	"log"

	"github.com/Des1red/gosqlitekit/sqlitekit"
)

var DB *sql.DB

func main() {
	err := sqlitekit.SetConfig(sqlitekit.Config{
		WAL:          true,
		ForeignKeys:  true,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	})
	if err != nil {
		log.Fatal(err)
	}

	err = sqlitekit.Initialize(
		"data/app.db",
		"internal/database/schema",
	)
	if err != nil {
		log.Fatal(err)
	}

	DB = sqlitekit.DB()
}
```

Configuration is locked after initialization and cannot be modified afterwards.

## Migrations

Place migration files in a schema directory.

### Example layout

```text
internal/database/schema/
├── 001_tokens.sql
├── 002_metrics.sql
└── 003_users.sql
```

### Migration behavior

Migrations run:

* in sorted order
* transactionally per file application step
* with per-file checksum tracking
* with per-statement tracking inside each file

sqlitekit supports **append-only updates** inside an already applied migration file.

That means this is allowed:

* keep previously applied statements unchanged
* append new `CREATE TABLE IF NOT EXISTS ...` or `CREATE INDEX IF NOT EXISTS ...` statements
* sqlitekit will apply only the newly appended statements

This is rejected:

* editing a previously applied statement
* removing an old statement
* reordering old statements

When an old applied statement changes, sqlitekit fails loudly instead of silently mutating existing schema state.

### Migration metadata

sqlitekit stores internal tracking metadata in:

* `schema_meta`
* `schema_migration_files`
* `schema_migration_statements`

These are sqlitekit-owned tables used for migration/version bookkeeping only.

## Example output

```text
┌──────────┬───────────────────────────────┐
│ STATUS   │ MIGRATION                     │
├──────────┼───────────────────────────────┤
│ ✔ APPLY  │ 001_tokens.sql                │
│ ✔ APPLY  │ 002_metrics.sql               │
│ ↺ SKIP   │ 003_users.sql                 │
└──────────┴───────────────────────────────┘
```

```text
✔ 2 applied  ↺ 1 skipped
```

## Design goals

* minimal
* predictable
* production-safe
* reusable across Go services

## Example

A full runnable example is included in the repository:

```text
example/
├── example.go
└── database/
    ├── db.go
    └── schema/
```

Run it with:

```bash
go run example/example.go
```

Example entry point:

```go
package main

import "github.com/Des1red/gosqlitekit/example/database"

func main() {
	database.InitDb()
}
```

Example database initialization:

```go
package database

import (
	"database/sql"
	"log"

	"github.com/Des1red/gosqlitekit/sqlitekit"
)

var DB *sql.DB

func InitDb() {
	err := sqlitekit.SetConfig(sqlitekit.Config{
		WAL:          false,
		ForeignKeys:  true,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	})
	if err != nil {
		panic(err)
	}

	err = sqlitekit.Initialize(
		"socialnet.db",
		"example/database/schema",
	)
	if err != nil {
		log.Fatal(err)
	}

	DB = sqlitekit.DB()
}
```

This example shows:

* configuration with `sqlitekit.SetConfig(...)`
* initialization with `sqlitekit.Initialize(...)`
* ordered schema application from `example/database/schema`

## License

MIT License.

## Author

Des1red
