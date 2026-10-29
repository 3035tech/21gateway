package database

import (
	"embed"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const (
	initialSMSMigration = "202610140001"
	migrationLockID     = 210014
)

// Migrate is an explicit operator action, never part of API startup.
func Migrate(db *gorm.DB) error {
	ids := []string{initialSMSMigration, "202610140002", "202610140003", "202610140004", "202610140005", "202610140006", "202610140007", "202610140008", "202610140009", "202610150010", "202610150011", "202610270012", "202610270013", "202610270014", "202610270015", "202610270016", "202610270017", "202610270018", "202610270019", "202610270020", "202610270021", "202610280022", "202610280023", "202610290024"}
	names := []string{"sms", "backoffice", "callbacks", "account_settings", "batch_metadata", "legacy_status", "mo", "mo_queries", "operations", "scheduled_sms", "dispatch_outbox", "request_reservations", "list_order_indexes", "schedule_listing", "user_mfa", "password_recovery", "credential_expiry", "user_company_access", "sms_history_content", "company_provider_token", "public_refs", "send_reference_outbox", "company_test_sends", "callback_auth"}
	migrations := make([]*gormigrate.Migration, 0, len(ids))
	for i, id := range ids {
		script, err := migrationFiles.ReadFile("migrations/" + id + "_" + names[i] + ".sql")
		if err != nil {
			return err
		}
		migrations = append(migrations, &gormigrate.Migration{ID: id, Migrate: func(db *gorm.DB) error { return db.Exec(string(script)).Error }})
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", migrationLockID).Error; err != nil {
			return err
		}
		options := *gormigrate.DefaultOptions
		options.UseTransaction = false // The outer transaction owns the advisory lock and all migration writes.
		options.ValidateUnknownMigrations = true
		return gormigrate.New(tx, &options, migrations).Migrate()
	})
}
