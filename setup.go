package pgtl

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Table struct for table details
type Table struct {
	Name   string
	Schema string
}

// Trigger Details
const (
	ListenEvent     = `event`
	TriggerFunction = `notifier`
	Trigger         = `notifier_trigger`
)

// CreateTriggerFunction creates triggered function
func CreateTriggerFunction(ctx context.Context, db *sql.DB) error {
	const createTriggerFunQuery = `
CREATE OR REPLACE FUNCTION notifier() RETURNS TRIGGER AS
$body$
	DECLARE
		message json;
		old_row json;
		new_row json;
		data json;
	BEGIN
		IF (TG_OP = 'UPDATE') THEN
			old_row = row_to_json(OLD);
			new_row = row_to_json(NEW);
			data = json_build_object('old row', old_row,'new row', new_row);
		ELSIF (TG_OP = 'DELETE') THEN
			old_row = row_to_json(OLD);
			data = json_build_object('old row', old_row);
		ELSIF (TG_OP = 'INSERT') THEN
			new_row = row_to_json(NEW);
			data = json_build_object('new row', new_row);
		END IF;
		message = json_build_object(
						'table_name', TG_TABLE_NAME,
						'schema_name', TG_TABLE_SCHEMA,
						'operation', TG_OP,
						'data', data
					);
		PERFORM pg_notify('event',message::text);
		RETURN NULL;
	END;
$body$
LANGUAGE plpgsql;`
	if _, err := db.ExecContext(ctx, createTriggerFunQuery); err != nil {
		return err
	}
	return nil
}

// TablesInDatabase returns list of tables in connected database
func TablesInDatabase(ctx context.Context, db *sql.DB) ([]*Table, error) {
	const query = `SELECT array_agg(table_schema || '.' || table_name) 
				   FROM information_schema.tables WHERE
				   table_schema != 'pg_catalog' AND table_schema != 'information_schema';`
	list := make([]string, 0, 10)
	if err := db.QueryRowContext(ctx, query).Scan(pq.Array(&list)); err != nil {
		return nil, err
	}
	tablesList := make([]*Table, 0, len(list))
	for _, l := range list {
		t := strings.Split(l, ".")
		tablesList = append(tablesList, &Table{t[1], t[0]})
	}
	return tablesList, nil
}

// TriggerSomeTables adds triggers to the provided tables
func TriggerSomeTables(ctx context.Context, db *sql.DB, tables []*Table) error {
	const dropTrigger = `DROP TRIGGER IF EXISTS notifier_trigger ON `
	const triggerQuerry1stHalf = `
CREATE TRIGGER notifier_trigger
AFTER INSERT OR UPDATE OR DELETE ON `
	const triggerQuerry2ndHalf = `
    FOR EACH ROW EXECUTE PROCEDURE notifier();`

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, tb := range tables {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("%s%s.%s;", dropTrigger, tb.Schema, tb.Name)); err != nil {
			return err
		}
		query := fmt.Sprintf("%s%s.%s%s;", triggerQuerry1stHalf, tb.Schema, tb.Name, triggerQuerry2ndHalf)
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// TriggerAllTables adds triggers to all the tables in the connected database
func TriggerAllTables(ctx context.Context, db *sql.DB) error {
	tables, err := TablesInDatabase(ctx, db)
	if err != nil {
		return err
	}
	return TriggerSomeTables(ctx, db, tables)
}

// SetupEverything sets everything up for the whole database to notify logs
func SetupEverything(ctx context.Context, dbConfig string) (*pq.Listener, error) {
	db, err := sql.Open("postgres", dbConfig)
	if err != nil {
		return nil, fmt.Errorf("connection not open | %v", err.Error())
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}

	if err := CreateTriggerFunction(ctx, db); err != nil {
		return nil, err
	}
	if err := TriggerAllTables(ctx, db); err != nil {
		return nil, err
	}

	ls := pq.NewListener(dbConfig, 15*time.Second, time.Minute, func(event pq.ListenerEventType, err error) {
		if err != nil {
			panic(fmt.Errorf("%v with ListenerEventType = %v", err.Error(), event))
		}
	})
	if err := ls.Listen(ListenEvent); err != nil {
		return nil, fmt.Errorf("event failed to listen | %v", err)
	}
	return ls, nil
}

// Listen ...
func Listen(ls *pq.Listener) error {
	if err := ls.Listen(ListenEvent); err != nil {
		return fmt.Errorf("event failed to listen | %v", err)
	}
	return nil
}

// Unlisten ...
func Unlisten(ls *pq.Listener) error {
	if err := ls.Unlisten(ListenEvent); err != nil {
		return fmt.Errorf("event failed to unlisten | %v", err)
	}
	return nil
}
