package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/m4hi2/MeterAlertBot/internal/config"
	"github.com/m4hi2/MeterAlertBot/migrations"
	"github.com/muesli/coral"
)

const migrationsDir = "migrations"

var migrateCmd = &coral.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateCreateCmd)
}

var migrateUpCmd = &coral.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	RunE: func(cmd *coral.Command, _ []string) error {
		return runMigration(func(m *migrate.Migrate) error { return m.Up() })
	},
}

var migrateDownCmd = &coral.Command{
	Use:   "down [n]",
	Short: "Roll back migrations (default: 1)",
	Args:  coral.MaximumNArgs(1),
	RunE: func(cmd *coral.Command, args []string) error {
		n := 1
		if len(args) == 1 {
			v, err := strconv.Atoi(args[0])
			if err != nil || v < 1 {
				return fmt.Errorf("n must be a positive integer, got %q", args[0])
			}
			n = v
		}
		return runMigration(func(m *migrate.Migrate) error { return m.Steps(-n) })
	},
}

var migrateCreateCmd = &coral.Command{
	Use:   "create <name>",
	Short: "Create a new migration file pair",
	Args:  coral.ExactArgs(1),
	RunE:  runMigrateCreate,
}

func runMigration(fn func(*migrate.Migrate) error) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, config.Get().DB.DSN)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := fn(m); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("migrations: no change")
			return nil
		}
		return err
	}

	slog.Info("migrations applied")
	return nil
}

var migrationFilename = regexp.MustCompile(`^([0-9]+)_.*\.(up|down)\.sql$`)

func runMigrateCreate(_ *coral.Command, args []string) error {
	name := strings.ReplaceAll(strings.TrimSpace(args[0]), " ", "_")
	if name == "" {
		return errors.New("migration name must not be empty")
	}

	next, err := nextMigrationVersion()
	if err != nil {
		return err
	}

	base := filepath.Join(migrationsDir, fmt.Sprintf("%06d_%s", next, name))
	upPath := base + ".up.sql"
	downPath := base + ".down.sql"

	const header = "-- From the ashes of defeat, knowledge rises\n"

	if err := os.WriteFile(upPath, []byte(header), 0644); err != nil {
		return fmt.Errorf("create up migration: %w", err)
	}
	if err := os.WriteFile(downPath, []byte(header), 0644); err != nil {
		return fmt.Errorf("create down migration: %w", err)
	}

	fmt.Println(upPath)
	fmt.Println(downPath)
	return nil
}

func nextMigrationVersion() (uint64, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", migrationsDir, err)
	}

	var max uint64
	for _, e := range entries {
		m := migrationFilename.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		v, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil {
			continue
		}
		if v > max {
			max = v
		}
	}
	return max + 1, nil
}
