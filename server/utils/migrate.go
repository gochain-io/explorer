//from https://raw.githubusercontent.com/appleboy/mgo-migrate/master/migrate.go had to replace imports
package utils

import (
	"context"
	"errors"

	"go.uber.org/zap"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// MigrateFunc is the func signature for migrating.
type MigrateFunc func(context.Context, *mgo.Database, *zap.Logger) error

// RollbackFunc is the func signature for rollbacking.
type RollbackFunc func(context.Context, *mgo.Database, *zap.Logger) error

// InitSchemaFunc is the func signature for initializing the schema.
type InitSchemaFunc func(*mgo.Database) error

// Options define options for all migrations.
type Options struct {
	// TableName is the migration table.
	TableName string
	// IDColumnName is the name of column where the migration id will be stored.
	IDColumnName string
}

// Migration represents a database migration (a modification to be made on the database).
type Migration struct {
	// ID is the migration identifier. Usually a timestamp like "201601021504".
	ID int
	// Comment is the comment for the migration - you could explain the purpose and goal of the migration.
	Comment string
	// Migrate is a function that will br executed while running this migration.
	Migrate MigrateFunc
	// Rollback will be executed on rollback. Can be nil.
	Rollback RollbackFunc
}

// Migrate represents a collection of all migrations of a database schema.
type Migrate struct {
	database   *mgo.Database
	collection *mgo.Collection
	options    *Options
	migrations []*Migration
	initSchema InitSchemaFunc
	ctx        context.Context
	lgr        *zap.Logger
}

var (
	// DefaultOptions can be used if you don't want to think about options.
	DefaultOptions = &Options{
		TableName:    "Migrations",
		IDColumnName: "id",
	}

	// ErrRollbackImpossible is returned when trying to rollback a migration
	// that has no rollback function.
	ErrRollbackImpossible = errors.New("It's impossible to rollback this migration")

	// ErrNoMigrationDefined is returned when no migration is defined.
	ErrNoMigrationDefined = errors.New("No migration defined")

	// ErrMissingID is returned when the ID od migration is equal to ""
	ErrMissingID = errors.New("Missing ID in migration")

	// ErrNoRunnedMigration is returned when any runned migration was found while
	// running RollbackLast
	ErrNoRunnedMigration = errors.New("Could not find last runned migration")
)

// New returns a new Gormigrate.
func New(ctx context.Context, db *mgo.Database, lgr *zap.Logger, options *Options, migrations []*Migration) *Migrate {
	collection := db.C(options.TableName)
	return &Migrate{
		database:   db,
		collection: collection,
		options:    options,
		migrations: migrations,
		ctx:        ctx,
		lgr:        lgr,
	}
}

// InitSchema sets a function that is run if no migration is found.
// The idea is preventing to run all migrations when a new clean database
// is being migrating. In this function you should create all tables and
// foreign key necessary to your application.
func (m *Migrate) InitSchema(initSchema InitSchemaFunc) {
	m.initSchema = initSchema
}

// Migrate executes all migrations that did not run yet.
func (m *Migrate) Migrate() error {
	if m.initSchema != nil && m.isFirstRun() {
		return m.runInitSchema()
	}

	for _, migration := range m.migrations {
		if err := m.runMigration(migration); err != nil {
			return err
		}
	}
	return nil
}

// RollbackLast undo the last migration
func (m *Migrate) RollbackLast() error {
	if len(m.migrations) == 0 {
		return ErrNoMigrationDefined
	}

	lastRunnedMigration, err := m.getLastRunnedMigration()
	if err != nil {
		return err
	}

	return m.RollbackMigration(lastRunnedMigration)
}

func (m *Migrate) getLastRunnedMigration() (*Migration, error) {
	for i := len(m.migrations) - 1; i >= 0; i-- {
		migration := m.migrations[i]
		if m.migrationDidRun(migration) {
			return migration, nil
		}
	}
	return nil, ErrNoRunnedMigration
}

// RollbackMigration undo a migration.
func (m *Migrate) RollbackMigration(mig *Migration) error {
	if mig.Rollback == nil {
		return ErrRollbackImpossible
	}

	if err := mig.Rollback(m.ctx, m.database, m.lgr); err != nil {
		return err
	}

	return m.collection.Remove(bson.M{
		m.options.IDColumnName: mig.ID,
	})
}

func (m *Migrate) runInitSchema() error {
	if err := m.initSchema(m.database); err != nil {
		return err
	}

	for _, migration := range m.migrations {
		if err := m.insertMigration(migration.ID, migration.Comment); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrate) runMigration(migration *Migration) error {
	if migration.ID <= 0 {
		return ErrMissingID
	}

	if !m.migrationDidRun(migration) {
		if err := migration.Migrate(m.ctx, m.database, m.lgr); err != nil {
			return err
		}

		if err := m.insertMigration(migration.ID, migration.Comment); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrate) migrationDidRun(mig *Migration) bool {
	count, _ := m.collection.Find(bson.M{
		m.options.IDColumnName: mig.ID,
	}).Count()
	return count > 0
}

func (m *Migrate) isFirstRun() bool {
	count, _ := m.collection.Count()
	return count == 0
}

func (m *Migrate) insertMigration(id int, comment string) error {
	return m.collection.Insert(bson.M{
		m.options.IDColumnName: id, "comment": comment,
	})
}
