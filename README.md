# pg-trigger-logs
pg-trigger-logs is a PostgreSQL change extraction driver using triggers, listen and notify.

Setup
-
pg-trigger-logs uses triggers and listeners for generating logs. <br>
Firstly, create a triggering function that notifies the log in the ``event`` channel. <br>
Now, create trigger for each table in the database, you want to get logs for.<br>
Finally, start listening the event and use your logs.

```
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
LANGUAGE plpgsql;

CREATE TRIGGER notifier_trigger
AFTER INSERT OR UPDATE OR DELETE ON stest.test
FOR EACH ROW EXECUTE PROCEDURE notifier();

LISTEN event;

```

Contributing
-
Changes and improvements are more than welcome! 
Feel free to fork and open a pull request. 
And Please make your changes in a specific branch and request to pull into master! If you can, please make sure the game fully works before sending the PR, as that will help speed up the process.

License
-
GO-REST-API is licensed under the [MIT license](https://github.com/Shivam010/pg-trigger-logs/blob/master/LICENSE).
