package cassmig

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	rootCmd = &cobra.Command{
		Use:     "cassmig",
		Short:   "Migration tool for Cassandra DB",
		Long:    "Migration tool for Cassandra DB",
		Version: "1.0.0",
	}

	createCmd = &cobra.Command{
		Use:   "create",
		Short: "creates .cql migration file",
		Args:  cobra.ExactArgs(1),
		RunE:  handleCreateCmd,
	}

	upCmd = &cobra.Command{
		Use:   "up",
		Short: "Executes Up statement in migration files",
		Args:  cobra.ExactArgs(1),
		RunE:  handleUpCmd,
	}

	downCmd = &cobra.Command{
		Use:   "down",
		Short: "Executes Down statement for latest migration file",
		Args:  cobra.ExactArgs(1),
		RunE:  handleDownCmd,
	}

	resetCmd = &cobra.Command{
		Use:   "reset",
		Short: "Executes Down statement for all migration files",
		Args:  cobra.ExactArgs(1),
		RunE:  handleResetCmd,
	}

	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Shows a list of applied & un-applied migrations",
		Args:  cobra.ExactArgs(1),
		RunE:  handleStatusCmd,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cmds := []*pflag.FlagSet{
		upCmd.Flags(),
		downCmd.Flags(),
		resetCmd.Flags(),
		statusCmd.Flags(),
	}
	for _, flags := range cmds {
		flags.String("hosts", "", "hosts seperated by comma")
		flags.String("port", "", "port")
		flags.String("keyspace", "", "keyspace is optional")
		flags.String("username", "", "username for authentication")
		flags.String("password", "", "password for authentication")
	}

	rootCmd.AddCommand(
		createCmd,
		upCmd,
		downCmd,
		resetCmd,
		statusCmd,
	)
}
