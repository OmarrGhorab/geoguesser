package auth_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/auth"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgresSessionStoreCreateAndGet(t *testing.T) {
	db, mock := newMockGormDB(t)
	store := auth.NewPostgresSessionStore(db)
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	session := &auth.RefreshSession{
		UserID:    userID,
		CreatedAt: time.Date(2026, 6, 26, 1, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 7, 3, 1, 0, 0, 0, time.UTC),
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "auth_sessions" ("user_id","refresh_token_hash","user_agent_hash","ip_address","expires_at","revoked_at","last_used_at","id","created_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING "id","created_at"`)).
		WithArgs(userID, "test-hash-create", nil, nil, session.ExpiresAt, nil, nil, sqlmock.AnyArg(), session.CreatedAt).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(uuid.MustParse("22222222-2222-2222-2222-222222222222"), session.CreatedAt))
	mock.ExpectCommit()

	if err := store.Create(ctx, "test-hash-create", session, time.Hour); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	rows := sqlmock.NewRows([]string{"id", "user_id", "refresh_token_hash", "user_agent_hash", "ip_address", "expires_at", "revoked_at", "created_at", "last_used_at"}).
		AddRow(uuid.MustParse("22222222-2222-2222-2222-222222222222"), userID, "test-hash-create", nil, nil, session.ExpiresAt, nil, session.CreatedAt, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "auth_sessions" WHERE refresh_token_hash = $1 AND revoked_at IS NULL ORDER BY "auth_sessions"."id" LIMIT $2`)).
		WithArgs("test-hash-create", 1).
		WillReturnRows(rows)

	got, err := store.Get(ctx, "test-hash-create")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.UserID != userID {
		t.Fatalf("user id = %v, want %v", got.UserID, userID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPostgresSessionStoreRevokedUserID(t *testing.T) {
	db, mock := newMockGormDB(t)
	store := auth.NewPostgresSessionStore(db)
	ctx := context.Background()
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	rows := sqlmock.NewRows([]string{"user_id"}).AddRow(userID)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "user_id" FROM "auth_sessions" WHERE refresh_token_hash = $1 AND revoked_at IS NOT NULL ORDER BY "auth_sessions"."id" LIMIT $2`)).
		WithArgs("revoked-hash", 1).
		WillReturnRows(rows)

	got, reused, err := store.RevokedUserID(ctx, "revoked-hash")
	if err != nil {
		t.Fatalf("revoked user lookup failed: %v", err)
	}
	if !reused {
		t.Fatal("expected revoked token to be detected")
	}
	if got != userID {
		t.Fatalf("user id = %v, want %v", got, userID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock setup failed: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       sqlDB,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm setup failed: %v", err)
	}

	return db, mock
}
