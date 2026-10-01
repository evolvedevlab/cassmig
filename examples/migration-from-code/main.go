package main

import (
	"context"
	"log"
	"os"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/evolvedevlab/cassmig"
)

const migrationsDir = "./migrations"

// Do this if you want to embed migrations in the binary
//go:embed ./migrations
// var migsFS embed.FS

func main() {
	migs, err := cassmig.GetMigrationsFS(os.DirFS(migrationsDir))
	if err != nil {
		log.Fatal(err)
	}

	cluster := gocql.NewCluster("127.0.0.1:9042")
	sess, err := cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close()

	migrator, err := cassmig.NewMigrator(sess)
	if err != nil {
		log.Fatal(err)
	}
	defer migrator.Close()

	if err := migrator.Execute(context.Background(), cassmig.MigCmdUp, migs); err != nil {
		log.Fatal(err)
	}
}
