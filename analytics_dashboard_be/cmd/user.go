package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"

	"analytics-dashboard-be/internal/domain"
	"analytics-dashboard-be/internal/repository/postgres"
	"analytics-dashboard-be/internal/service"
)

var (
	userName     string
	userPassword string
	userRole     string
)

// userCmd injects/updates an account. There is no public sign-up — this is the
// only way accounts are created. Re-running for the same email resets it.
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Create or update a login account (idempotent by email)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(userName) == "" || userPassword == "" {
			return fmt.Errorf("--username and --password are required")
		}
		role := domain.Role(strings.ToUpper(userRole))
		if role != domain.RoleUser && role != domain.RoleAdmin {
			return fmt.Errorf("--role must be USER or ADMIN")
		}

		db, err := postgres.Connect(cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer db.Close()

		// The password is hashed (bcrypt, cost 12, per-password salt) here and the
		// plaintext is never written anywhere.
		hash, err := service.HashPassword(userPassword)
		if err != nil {
			return err
		}
		username := strings.ToLower(strings.TrimSpace(userName))
		if err := postgres.NewUserRepo(db).Upsert(cmd.Context(), username, hash, role); err != nil {
			return err
		}
		log.Printf("account ready: %s (%s)", username, role)
		return nil
	},
}

func init() {
	userCmd.Flags().StringVar(&userName, "username", "", "account username")
	userCmd.Flags().StringVar(&userPassword, "password", "", "account password")
	userCmd.Flags().StringVar(&userRole, "role", "USER", "USER or ADMIN")
}
