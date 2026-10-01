package cassmig

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

type MigCmd string

const (
	MigCmdUp    MigCmd = "up"
	MigCmdDown  MigCmd = "down"
	MigCmdReset MigCmd = "reset"
)

type StatementType int

const (
	UpStatement StatementType = iota
	DownStatement
)

type Migration struct {
	Version    string
	Name       string
	Checksum   string
	AppliedAt  time.Time
	Statements []string
}

func (mig Migration) GetOriginalFilename() string {
	return fmt.Sprintf("%s_%s", mig.Version, mig.Name)
}

type Migrator struct {
	session *gocql.Session
	// db state
	migrationsMap map[string]*Migration
	migrations    []*Migration

	// migrations that needs to be applied after comparing
	toBeApplied    []*Migration
	toBeAppliedMap map[string]*Migration
}

func NewMigrator(session *gocql.Session) (*Migrator, error) {
	migrator := &Migrator{
		session:        session,
		toBeAppliedMap: make(map[string]*Migration),
		migrationsMap:  make(map[string]*Migration),
		migrations:     make([]*Migration, 0),
		toBeApplied:    make([]*Migration, 0),
	}

	if err := migrator.createDBStateTable(); err != nil {
		return nil, fmt.Errorf("create db state table aka %s failed with error: %v", dbStateTableName, err)
	}
	migs, err := migrator.getAll()
	if err != nil {
		return nil, fmt.Errorf("migration state retrieval from %s failed with error: %v", dbStateTableName, err)
	}
	for _, m := range migs {
		migrator.migrationsMap[m.Version] = m
		migrator.migrations = append(migrator.migrations, m)
	}

	return migrator, nil
}

func (m *Migrator) Execute(ctx context.Context, cmd MigCmd, migrations []*Migration) error {
	if cmd == MigCmdReset {
		for _, mig := range migrations {
			for _, stmt := range mig.Statements {
				if err := m.session.Query(stmt).ExecContext(ctx); err != nil {
					return fmt.Errorf("%s statement error: %v", mig.GetOriginalFilename(), err)
				}
			}
		}
		return m.storeState(ctx, cmd, migrations)
	}
	if cmd == MigCmdDown {
		if len(m.migrations) == 0 {
			return nil
		}

		sort.Slice(m.migrations, func(i, j int) bool {
			v1, _ := strconv.ParseInt(m.migrations[i].Version, 10, 64)
			v2, _ := strconv.ParseInt(m.migrations[j].Version, 10, 64)
			return v1 > v2
		})

		latest := m.migrations[0]
		var ok bool
		latest, ok = sliceToMap(migrations)[latest.Version]
		if !ok {
			return fmt.Errorf("latest migration not found in filesystem")
		}

		if err := m.session.Query(latest.Statements[DownStatement]).ExecContext(ctx); err != nil {
			return fmt.Errorf("%s statement error: %v", latest.GetOriginalFilename(), err)
		}
		return m.storeState(ctx, cmd, []*Migration{latest})
	}

	m.compare(migrations)
	if len(m.toBeApplied) == 0 {
		return nil
	}

	for _, mig := range m.toBeApplied {
		if err := m.session.Query(mig.Statements[UpStatement]).ExecContext(ctx); err != nil {
			return fmt.Errorf("%s statement error: %v", mig.GetOriginalFilename(), err)
		}
	}
	return m.storeState(ctx, cmd, m.toBeApplied)
}

// Compare compares DB applied migration state with local files and returns
// migrations that has not been applied yet.
func (m *Migrator) Compare(migrations []*Migration) ([]*Migration, map[string]*Migration) {
	notApplied := make([]*Migration, 0)
	notAppliedMap := make(map[string]*Migration)
	for _, new := range migrations {
		// compare with db mig checksum
		if old, ok := m.migrationsMap[new.Version]; ok {
			if old.Checksum == new.Checksum {
				continue
			}
		}

		notAppliedMap[new.Version] = new
		notApplied = append(notApplied, new)
	}
	return notApplied, notAppliedMap
}

func (m *Migrator) Close() error {
	//m.session.Close()
	return nil
}

func (m *Migrator) compare(migrations []*Migration) {
	for _, new := range migrations {
		// compare with db mig checksum
		if old, ok := m.migrationsMap[new.Version]; ok {
			if old.Checksum == new.Checksum {
				continue
			}
		}

		m.toBeAppliedMap[new.Version] = new
		m.toBeApplied = append(m.toBeApplied, new)
	}
}

func (m *Migrator) storeState(ctx context.Context, cmd MigCmd, applied []*Migration) error {
	if cmd == MigCmdReset {
		return m.session.Query(fmt.Sprintf("TRUNCATE TABLE %s.%s", dbStateKeyspace, dbStateTableName)).ExecContext(ctx)
	}
	if cmd == MigCmdDown {
		for _, mig := range applied {
			err := m.session.Query(
				fmt.Sprintf("DELETE FROM %s.%s WHERE version = ?", dbStateKeyspace, dbStateTableName),
				mig.Version,
			).ExecContext(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, mig := range applied {
		err := m.session.Query(
			fmt.Sprintf("INSERT INTO %s.%s (version, name, checksum, applied_at) VALUES (?, ?, ?, ?)", dbStateKeyspace, dbStateTableName),
			mig.Version, mig.Name, mig.Checksum, mig.AppliedAt,
		).ExecContext(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) createDBStateTable() error {
	if err := m.session.Query(createDBStateKeyspace).Exec(); err != nil {
		return err
	}
	return m.session.Query(createDBStateMigration).Exec()
}

func (m *Migrator) getAll() ([]*Migration, error) {
	q := m.session.Query(fmt.Sprintf("SELECT version, name, checksum, applied_at FROM %s.%s", dbStateKeyspace, dbStateTableName))
	it := q.Iter()
	defer it.Close()

	sc := it.Scanner()

	migs := make([]*Migration, 0)
	for sc.Next() {
		var m Migration
		if err := sc.Scan(&m.Version, &m.Name, &m.Checksum, &m.AppliedAt); err != nil {
			return nil, err
		}

		migs = append(migs, &m)
	}

	return migs, sc.Err()
}

func createFileChecksum(r io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sliceToMap(items []*Migration) map[string]*Migration {
	m := make(map[string]*Migration)
	for _, item := range items {
		m[item.Version] = item
	}
	return m
}
