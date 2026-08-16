package cmd

import (
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"analytics-dashboard-be/internal/config"
)

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "analytics",
	Short: "AI logistics analytics API",
	Long:  "Backend for the AI-powered logistics analytics dashboard (Gin + Postgres + Redis).",
}

// Execute is the entrypoint called from main.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(func() {
		// Load a local .env if present (gitignored) so secrets like GEMINI_API_KEY
		// stay out of the repo and out of launch.json. Real env vars still win.
		_ = godotenv.Load()
		cfg = config.Load()
	})
	rootCmd.AddCommand(serveCmd, seedCmd, userCmd)
}
