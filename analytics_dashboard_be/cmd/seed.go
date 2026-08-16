package cmd

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"analytics-dashboard-be/internal/repository/postgres"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Load the CSV dataset into the orders table (idempotent: truncates first)",
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

		rows, err := csv.NewReader(f).ReadAll()
		if err != nil {
			return fmt.Errorf("read csv: %w", err)
		}
		if len(rows) < 2 {
			return fmt.Errorf("csv has no data rows")
		}

		tx, err := db.Beginx()
		if err != nil {
			return err
		}
		if _, err := tx.Exec("TRUNCATE orders"); err != nil {
			tx.Rollback()
			return err
		}

		const insert = `INSERT INTO orders
			(client_id, order_id, order_date, delivery_date, carrier, origin_city, destination_city,
			 status, sku, product_category, quantity, unit_price_usd, order_value_usd, is_promo,
			 promo_discount_pct, region, warehouse, transit_days)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`
		stmt, err := tx.Preparex(insert)
		if err != nil {
			tx.Rollback()
			return err
		}

		count := 0
		for _, r := range rows[1:] { // skip header
			orderDate := r[2]
			var deliveryDate interface{}
			var transitDays interface{}
			if r[3] != "" {
				deliveryDate = r[3]
				if od, err1 := time.Parse("2006-01-02", orderDate); err1 == nil {
					if dd, err2 := time.Parse("2006-01-02", r[3]); err2 == nil {
						transitDays = int(dd.Sub(od).Hours() / 24)
					}
				}
			}
			if _, err := stmt.Exec(
				r[0], r[1], orderDate, deliveryDate, r[4], r[5], r[6], r[7], r[8], r[9],
				atoi(r[10]), atof(r[11]), atof(r[12]), r[13] == "1", atof(r[14]), r[15], r[16], transitDays,
			); err != nil {
				tx.Rollback()
				return fmt.Errorf("insert row %s: %w", r[1], err)
			}
			count++
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Printf("seeded %d orders from %s", count, cfg.SeedCSVPath)
		return nil
	},
}

func atoi(s string) int       { n, _ := strconv.Atoi(s); return n }
func atof(s string) float64   { f, _ := strconv.ParseFloat(s, 64); return f }
