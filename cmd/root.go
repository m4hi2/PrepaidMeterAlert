package cmd

import (
	"github.com/m4hi2/MeterAlertBot/internal/config"
	"github.com/m4hi2/MeterAlertBot/internal/utils/logger"
	"github.com/muesli/coral"
)

var rootCmd = &coral.Command{
	Use:   "meterbot",
	Short: "Prepaid meter balance alert bot",
	PersistentPreRunE: func(cmd *coral.Command, args []string) error {
		cfg := config.Load()
		logger.InitLogger(cfg.Log.Level, cfg.Log.Format)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		return
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(alertCmd)
}
