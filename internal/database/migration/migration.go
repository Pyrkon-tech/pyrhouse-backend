package migration

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // register postgres DB and SQL
	_ "github.com/golang-migrate/migrate/v4/source/file"       // register postgres File
)

func Migrate(dbURL string, migrationsPath string, verbose bool, log *zap.Logger) error {
	log.Info("Running database migration")

	dbMigrate, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return err
	}
	log.Info("Run registered migrations under: " + dbURL)
	err = dbMigrate.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("Database migration: no change needed")
		} else {
			log.Error("Database migration failed", zap.Error(err))
			return err
		}
	}

	return nil
}

type Logger struct {
	logger  *zap.Logger
	verbose bool
}

func (l *Logger) Printf(format string, v ...any) {
	l.logger.Sugar().Infof("DB Migration: "+format, v...)
}

func (l *Logger) Verbose() bool {
	return l.verbose
}

func NewLogger(logger *zap.Logger, verbose bool) *Logger {
	return &Logger{
		logger:  logger,
		verbose: verbose,
	}
}
