## Hammerpost Agent

The `hammerpost-agent` tool works with `hammerpost`. This tool adds parameters to PostgreSQL via `ALTER SYSTEM`.
If the database type is `MySQL` then it appends the given parameters to the `my.cnf` file for MySQL.
This tool also restarts PostgreSQL or MySQL after applying the given parameter set. It is also responsible for collecting node metrics like `CPU` and `Memory` while running the `HammerDB` workload.

## Quick Setup

        This tool requires a minimum version of Go 1.18 or higher to function.
        Please build the binaries using the supported Go version.


1. Install golang 1.18

        $ sudo yum install golang

2. Clone this repo and make

        $ git clone https://github.com/dineshkumar02/hammerpost-agent.git
        $ cd hammerpost-agent
        $ make

3. Start the agent by providing start-cmd, stop-cmd

## Example
```
./hammerpost-agent --stop-cmd "/usr/pgsql-15/bin/pg_ctl -D /var/lib/pgsql/15/data stop -mf" --start-cmd "/usr/pgsql-15/bin/pg_ctl -D /var/lib/pgsql/15/data start -l /tmp/startup.log" --pgdsn "postgres://postgres:postgres@localhost:5432/postgres" --db-type postgres
```

## Usage
| Option        | Usage                                               |
|---------------|-----------------------------------------------------|
| --listen      | listen address for the hammerpost-agent             |
| --start-cmd   | Start command which starts the database             |
| --stop-cmd    | Stop command which stops the database               |
| --pgdsn       | PostgreSQL connection string which is running local |
| --my-cnf-path | Path to mysqld.cnf file                             |
| --db-type     | mysql or postgres                                   |