package cassmig

const dbStateKeyspace = "cassmig_keyspace"
const dbStateTableName = "cassmig_state"

const createDBStateKeyspace = `
CREATE KEYSPACE IF NOT EXISTS cassmig_keyspace
WITH replication = {
    'class': 'SimpleStrategy',
    'replication_factor' : 1
};`
const createDBStateMigration = `
CREATE TABLE IF NOT EXISTS
cassmig_keyspace.cassmig_state (
	version BIGINT PRIMARY KEY,
	name TEXT,
	checksum TEXT,
	applied_at TIMESTAMP
);`

const exampleMigration = `-- +cassmig Up
SELECT 'up CQL query';

-- +cassmig Down
SELECT 'down CQL query';`
