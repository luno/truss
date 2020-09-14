# truss

> truss /trŭs/
>
>  n. A rigid framework, as of wooden beams or metal bars, designed to support a structure, such as a roof.
>  
>  v. to support, strengthen, or stiffen by or as if by a truss.

Truss is a simple golang mysql schema management library that provides the following features:
- `truss.Connect` returns a `*sql.DB` for a production use.
- `truss.Migrate` schema management via migration queries (roll forward only). 
- `truss.ConnectForTesting` returns testing `*sql.DB` for a temp database with the current schema.
- `truss.TestSchema` provides a snapshot of the current schema.

## TL;DR

Common usage is to wrap `truss` in your `db` package:

`db/migrations.go`
```
package db

// migrations is an append-only list of all migrations over time.
var migrations = []string{`

CREATE TABLE users (
  id BIGINT NOT NULL AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  type INT NOT NULL,

  PRIMARY KEY (id),
  INDEX by_name (name)
);`, `

ALTEST TABLE users ADD COLUMN surname VARCHAR(255) AFTER name;
`,
}
```

`db/db.go`
```
// Connect returns a database connection and ensures latest migrations are applied.
func Connect(uri string) (*sql.DB, error) {
	dbc, err := truss.Connect(uri)
	if err != nil {
		return nil, err
	}

	err = truss.Migrate(context.Background(), dbc, migrations)
	if err != nil {
		return nil, err
	}

	return dbc, nil
}

// ConnectForTesting returns a database connection for a temp database with latest schema. 
func ConnectForTesting(t *testing.T) *sql.DB {
	return truss.ConnectForTesting(t, migrations...)
}
```
`migrations_test.go`
```
package db

var update = flag.Bool("update", false, "update schema file")

//go:generate go test -update -run=TestSchema

func TestSchema(t *testing.T) {
	truss.TestSchema(t, "schema.sql", *update, migrations...)
}
```
Which will generate the following `db/schema.sql` snapshot of current schema (checked into the git).
```
CREATE TABLE `users` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `surname` varchar(255),
  `type` int(11) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `by_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4

CREATE TABLE `migrations` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `query_hash` char(64) NOT NULL,
  `schema_hash` char(64) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```
