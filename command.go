package cassmig

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func handleCreateCmd(cmd *cobra.Command, args []string) error {
	name := args[0]
	hasExt := filepath.Ext(name) == ".cql"

	dir, filename := filepath.Split(name)

	name = strconv.Itoa(int(time.Now().UnixNano())) + "_" + filename
	if !hasExt {
		name = name + ".cql"
	}

	path := filepath.Join(dir, name)

	if len(dir) > 0 {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write([]byte(exampleMigration)); err != nil {
		return err
	}

	fmt.Printf("%s created successfully.\n", path)
	return nil
}

func handleUpCmd(cmd *cobra.Command, args []string) error {
	dir := args[0]
	root := os.DirFS(dir)

	migs, err := GetMigrationsFS(root)
	if err != nil {
		return err
	}

	session, err := createSessionWithFlags(cmd.Flags())
	if err != nil {
		return err
	}
	defer session.Close()

	migrator, err := NewMigrator(session)
	if err != nil {
		return err
	}
	defer migrator.Close()

	if err := migrator.Execute(cmd.Context(), MigCmdUp, migs); err != nil {
		return err
	}

	fmt.Printf("%d migrations applied successfully.\n", len(migrator.toBeApplied))
	return nil
}

func handleDownCmd(cmd *cobra.Command, args []string) error {
	dir := args[0]
	root := os.DirFS(dir)

	migs, err := GetMigrationsFS(root)
	if err != nil {
		return err
	}

	session, err := createSessionWithFlags(cmd.Flags())
	if err != nil {
		return err
	}
	defer session.Close()

	migrator, err := NewMigrator(session)
	if err != nil {
		return err
	}
	defer migrator.Close()

	if err := migrator.Execute(cmd.Context(), MigCmdDown, migs); err != nil {
		return err
	}

	fmt.Print("migration rollback successful.\n")
	return nil
}

func handleResetCmd(cmd *cobra.Command, args []string) error {
	dir := args[0]
	root := os.DirFS(dir)

	migs, err := GetMigrationsFS(root)
	if err != nil {
		return err
	}

	session, err := createSessionWithFlags(cmd.Flags())
	if err != nil {
		return err
	}
	defer session.Close()

	migrator, err := NewMigrator(session)
	if err != nil {
		return err
	}
	defer migrator.Close()

	if err := migrator.Execute(cmd.Context(), MigCmdReset, migs); err != nil {
		return err
	}

	fmt.Print("reset successful.\n")
	return nil
}

type comparison struct {
	*Migration
	IsApplied bool
}

func handleStatusCmd(cmd *cobra.Command, args []string) error {
	dir := args[0]
	root := os.DirFS(dir)

	migs, err := GetMigrationsFS(root)
	if err != nil {
		return err
	}

	session, err := createSessionWithFlags(cmd.Flags())
	if err != nil {
		return err
	}
	defer session.Close()

	migrator, err := NewMigrator(session)
	if err != nil {
		return err
	}
	defer migrator.Close()

	var comps []comparison
	_, notApplied := migrator.Compare(migs)
	for _, mig := range migs {
		var comp comparison
		if _, ok := notApplied[mig.Version]; !ok {
			comp.IsApplied = true
		}

		comp.Migration = mig
		comps = append(comps, comp)
	}

	for _, comp := range comps {
		msg := "pending"
		if comp.IsApplied {
			msg = "applied"
		}

		fmt.Printf("%s | %s\n", comp.GetOriginalFilename(), msg)
	}

	return nil
}

func GetMigrationsFS(root fs.FS) ([]*Migration, error) {
	migrations := make([]*Migration, 0)
	err := fs.WalkDir(root, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s does not exist", root)
			}
			return fmt.Errorf("failed to read dir entry from %s", root)
		}
		if !d.IsDir() {
			data, err := fs.ReadFile(root, path)
			if err != nil {
				return err
			}

			dataParts := bytes.Split(data, []byte("-- +cassmig Down"))
			if len(dataParts) != 2 {
				return fmt.Errorf("missing or invalid down statement in %s", path)
			}

			upParts := bytes.Split(dataParts[0], []byte("-- +cassmig Up"))
			if len(dataParts) != 2 {
				return fmt.Errorf("missing or invalid up statement in %s", path)
			}

			up := strings.TrimSpace(string(upParts[1]))
			down := strings.TrimSpace(string(dataParts[1]))

			checksum, err := createFileChecksum(bytes.NewReader(data))
			if err != nil {
				return err
			}
			if len(up) == 0 && len(down) == 0 {
				return fmt.Errorf("no statement found for migration %s", path)
			}

			parts := strings.SplitN(path, "_", 2)
			mig := &Migration{
				Version:    parts[0],
				Name:       parts[1],
				Checksum:   checksum,
				Statements: []string{up, down},
				AppliedAt:  time.Now(),
			}

			migrations = append(migrations, mig)
		}
		return nil
	})
	return migrations, err
}

func createSessionWithFlags(flags *pflag.FlagSet) (*gocql.Session, error) {
	var (
		hosts, _    = flags.GetString("hosts")
		portStr, _  = flags.GetString("port")
		keyspace, _ = flags.GetString("keyspace")
		port, _     = strconv.Atoi(portStr)
		username, _ = flags.GetString("username")
		password, _ = flags.GetString("password")
	)

	cluster := gocql.NewCluster(strings.Split(hosts, ",")...)
	cluster.Port = port
	cluster.Keyspace = keyspace
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: username,
		Password: password,
	}

	return cluster.CreateSession()
}
