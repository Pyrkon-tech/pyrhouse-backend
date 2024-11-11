package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"warehouse/internal/database/migration"
	"warehouse/internal/logger"

	"github.com/spf13/cobra"
)

var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run migrations manually.",
	Long:  `Command that exists and should be used only for development purposes.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dbURL := os.Getenv("DATABASE_URL")
		migrationDir, _ := cmd.Flags().GetString("dir")

		err := migration.Migrate(
			dbURL,
			fmt.Sprintf("file://%s", migrationDir),
			true,
			logger.NewLogger(),
		)
		if err != nil {
			log.Println(err.Error())
			return fmt.Errorf("migrate database: %w", err)
		}

		return nil
	},
}

func Execute(ctx context.Context) {
	rootCmd := &cobra.Command{
		Use:   "pyrhouse",
		Short: "Pyrhouse management service",
	}
	MigrateCmd.Flags().String("dir", "../../migrations", "Directory containing the migration files")
	rootCmd.AddCommand(MigrateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
