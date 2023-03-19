## Hammerpost Agent

This agent has to be configured on the database, where PostgreSQL or mysql is running.

Below is an example to start the agent service on the PostgreSQL node.

```
./hammerpost-agent --stop-cmd "/usr/pgsql-15/bin/pg_ctl -D /var/lib/pgsql/15/data stop -mf" --start-cmd "/usr/pgsql-15/bin/pg_ctl -D /var/lib/pgsql/15/data start -l /tmp/startup.log" --pgdsn "postgres://postgres:postgres@localhost:5432/postgres" --db-type postgres
```