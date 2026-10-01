## How to use?

```sh
Usage:
  cassmig [command]

Available Commands:
  create      creates .cql migration file
  down        Executes Down statement for latest migration file
  reset       Executes Down statement for all migration files
  status      Shows a list of applied & un-applied migrations
  up          Executes Up statement in migration files
  help        Help about any command
```

### Create Migration file

```sh
cassmig create ./path-to-migrations-dir/name
```

### Status (Compare DB vs Local Migrations)

```sh
cassmig status ./path-to-migrations-dir --hosts=127.0.0.1 --port=9042
```

### Up (Apply un-applied migrations)

```sh
cassmig up ./path-to-migrations-dir --hosts=127.0.0.1 --port=9042
```

### Down (Rollback latest migration)

```sh
cassmig down ./path-to-migrations-dir --hosts=127.0.0.1 --port=9042
```

### Reset (Delete all migration state)

```sh
cassmig reset ./path-to-migrations-dir --hosts=127.0.0.1 --port=9042
```
