package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"analytics-dashboard-be/internal/repository/postgres"
	"analytics-dashboard-be/internal/service"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Load the CSV dataset into the orders table (replaces existing data)",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := postgres.Connect(cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer db.Close()

		f, err := os.Open(cfg.SeedCSVPath)
		if err != nil {
			return fmt.Errorf("open csv: %w", err)
		}
		defer f.Close()

		// Shared parser (also used by the /admin/orders/import endpoint).
		orders, rowErrs, err := service.ParseOrdersCSV(f)
		if err != nil {
			return err
		}
		n, err := postgres.NewOrderRepo(db).ImportOrders(cmd.Context(), orders, true /* replace */)
		if err != nil {
			return err
		}
		log.Printf("seeded %d orders from %s (%d rows skipped)", n, cfg.SeedCSVPath, len(rowErrs))
		return nil
	},
}
