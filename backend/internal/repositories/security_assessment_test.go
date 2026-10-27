package repositories

import (
	"context"
	"strings"
	"testing"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Inspect production GORM statements without connecting to or modifying a database.
func TestSecuritySQLInjectionRemainsBoundData(t *testing.T) {
	db, err := gorm.Open(postgres.Open("postgres://unused:unused@127.0.0.1:1/unused"), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	type captured struct {
		sql  string
		vars []any
	}
	var statements []captured
	if err := db.Callback().Query().After("gorm:query").Register("assessment:sql", func(tx *gorm.DB) {
		statements = append(statements, captured{tx.Statement.SQL.String(), append([]any(nil), tx.Statement.Vars...)})
	}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, payload := range []string{"' OR '1'='1' --", "'; SELECT pg_sleep(2); --", "admin' UNION SELECT NULL --", "%_\\"} {
		statements = nil
		_, err := NewCredentials(db).FindByUsername(ctx, payload)
		if err != nil {
			t.Fatal(err)
		}
		_, err = NewUsers(db).UserByEmail(ctx, payload)
		if err != nil {
			t.Fatal(err)
		}
		_, err = NewCompanies(db).Companies(ctx, dtos.ListQuery{Search: payload, Page: 1, PageSize: 10, Direction: dtos.SortAscending})
		if err != nil {
			t.Fatal(err)
		}
		if len(statements) < 3 {
			t.Fatal("production queries not exercised")
		}
		bound := false
		for _, statement := range statements {
			if strings.Contains(statement.sql, payload) || strings.Contains(statement.sql, "pg_sleep") {
				t.Fatal("payload embedded as SQL")
			}
			if !strings.Contains(statement.sql, "$1") {
				t.Fatal("missing placeholders")
			}
			for _, v := range statement.vars {
				if value, ok := v.(string); ok && (value == payload || value == searchTerm(payload)) {
					bound = true
				}
			}
		}
		if !bound {
			t.Fatal("attack payload not carried as a bound value")
		}
	}
	for _, sort := range []string{"id; SELECT pg_sleep(2)", "name DESC --", "(SELECT 1)", "id, password_hash"} {
		_, err := NewCompanies(db).Companies(ctx, dtos.ListQuery{Sort: sort, Page: 1, PageSize: 10, Direction: dtos.SortAscending})
		if err == nil {
			t.Fatal("untrusted sort accepted")
		}
	}
}
