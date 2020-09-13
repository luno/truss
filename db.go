package truss

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/jtest"
	"github.com/stretchr/testify/require"
)

func Connect(uri string) (*sql.DB, error) {
	const prefix = "mysql://"
	if !strings.HasPrefix(uri, prefix) {
		return nil, errors.New("connect string missing mysql:// prefix")
	}
	uri = uri[len(prefix):]

	if uri[len(uri)-1] != '?' {
		uri += "&"
	}
	uri += defaultOptions()

	dbc, err := sql.Open("mysql", uri)
	if err != nil {
		return nil, err
	}

	return dbc, nil
}

// ConnectForTesting returns a connection to a newly created database
// with migration queries applied. Test cleanup automatically drops the database.
func ConnectForTesting(t *testing.T, queries ...string) *sql.DB {
	dbc, err := Connect(getTestURI())
	jtest.RequireNil(t, err)

	ctx := context.Background()

	// Multiple connections are problematic for unit tests since they
	// introduce concurrency issues.
	dbc.SetMaxOpenConns(1)

	_, err = dbc.ExecContext(ctx, "set time_zone='+00:00';")
	jtest.RequireNil(t, err)

	dbName := fmt.Sprintf("test_%d", time.Now().UnixNano())

	_, err = dbc.ExecContext(ctx, "CREATE DATABASE "+dbName+";")
	jtest.RequireNil(t, err)
	_, err = dbc.ExecContext(ctx, "USE "+dbName+";")
	jtest.RequireNil(t, err)

	err = Migrate(ctx, dbc, queries)
	jtest.RequireNil(t, err)

	t.Cleanup(func() {
		_, err = dbc.ExecContext(ctx, "DROP DATABASE "+dbName+";")
		jtest.RequireNil(t, err)

		jtest.RequireNil(t, dbc.Close())
	})

	return dbc
}

// TestSchema ensures that the schema file is up to date with the queries. It updates
// it if update is true.
func TestSchema(t *testing.T, schemapath string, update bool, queries ...string) {
	dbc := ConnectForTesting(t)
	ctx := context.Background()

	err := Migrate(ctx, dbc, queries)
	jtest.RequireNil(t, err)

	schema, err := MakeCreateSchema(ctx, dbc)
	jtest.RequireNil(t, err)

	if update {
		err := ioutil.WriteFile(schemapath, []byte(schema), 0644)
		jtest.RequireNil(t, err)
		return
	}

	actual, err := ioutil.ReadFile(schemapath)
	jtest.RequireNil(t, err)

	require.Equal(t, schema, string(actual))
}

func defaultOptions() string {
	// parseTime: Allows using time.Time for datetime
	return "parseTime=true"
}

func sockFile() string {
	sock := "/tmp/mysql.sock"
	if _, err := os.Stat(sock); os.IsNotExist(err) {
		// try common linux/Ubuntu socket file location
		return "/var/run/mysqld/mysqld.sock"
	}
	return sock
}

const envTestURI = "TRUSS_TEST_URI"

func getTestURI() string {
	if uri, ok := os.LookupEnv(envTestURI); ok {
		return uri
	}

	return "mysql://root@unix(" + sockFile() + ")/?"
}
