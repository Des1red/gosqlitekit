# sqlitekit

A small, simple SQLite initialization and migration helper for Go.

### sqlitekit provides:

- SQLite connection initialization

- Safe pragmas (WAL, foreign keys, busy timeout)

- Ordered SQL migrations

- Migration tracking (schema_migrations)

- Colored CLI output with status icons

- Supports filesystem and embedded migrations

- No ORM, no query builder — just a lightweight infrastructure layer.

### Installation
```go
go get github.com/Des1red/sqlitekit
```
### Usage
#### Filesystem migrations
Initialize the database and run migrations from a directory.
```go
package main

import "github.com/Des1red/sqlitekit"

func main() {

	err := sqlitekit.Initialize(
		"data/app.db",
		"internal/database/schema",
	)

	if err != nil {
		panic(err)
	}

}
```
#### Embedded migrations
You can embed migrations directly into the binary using Go's embed package.
```go
package main

import (
	"embed"
	"github.com/Des1red/sqlitekit"
)

//go:embed schema/*.sql
var migrations embed.FS

func main() {

	err := sqlitekit.InitializeEmbedded(
		"data/app.db",
		migrations,
	)

	if err != nil {
		panic(err)
	}

}
```
This allows distributing a single binary without needing a schema folder.

### Configuration

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

import "github.com/Des1red/sqlitekit"
var DB *sql.DB
func main() {

	err := sqlitekit.SetConfig(sqlitekit.Config{
		WAL:          true,
		ForeignKeys:  true,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	})

	if err != nil {
		panic(err)
	}

	err = sqlitekit.Initialize(
		"data/app.db",
		"internal/database/schema",
	)

	if err != nil {
		panic(err)
	}
	DB = sqlitekit.DB()
}
```
Configuration is locked after initialization and cannot be modified afterwards.
### Migrations

Place migration files in a schema directory.

#### Example:

internal/database/schema/

001_tokens.sql
002_metrics.sql
003_users.sql

### Migrations run:

- in sorted order

- only once

- tracked in the schema_migrations table

### Example Output
```
┌──────────┬───────────────────────────────┐
│ STATUS   │ MIGRATION                     │
├──────────┼───────────────────────────────┤
│ ✔ APPLY  │ 001_tokens.sql                │
│ ✔ APPLY  │ 002_metrics.sql               │
│ ↺ SKIP   │ 003_users.sql                 │
└──────────┴───────────────────────────────┘
```
✔ 2 applied  ↺ 1 skipped
### Design Goals

- minimal

- predictable

- production-safe

- reusable across Go services

### License

MIT License.

### Author
Des1red