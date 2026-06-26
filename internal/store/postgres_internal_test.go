package store

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

func TestAdminIdentityInsertConflictMapsKnownUniqueConstraints(t *testing.T) {
	for _, tc := range []struct {
		name       string
		constraint string
		code       string
		message    string
	}{
		{
			name:       "actor",
			constraint: "admin_identities_actor_key",
			code:       "ADMIN_IDENTITY_ACTOR_EXISTS",
			message:    "admin identity actor already exists",
		},
		{
			name:       "primary key",
			constraint: "admin_identities_pkey",
			code:       "ADMIN_IDENTITY_EXISTS",
			message:    "admin identity already exists",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := adminIdentityInsertConflict(&pgconn.PgError{Code: "23505", ConstraintName: tc.constraint})
			var appErr domain.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected app error, got %T %v", err, err)
			}
			if appErr.Status != 409 || appErr.Code != tc.code || appErr.Message != tc.message {
				t.Fatalf("unexpected conflict error: %#v", appErr)
			}
		})
	}
}

func TestAdminIdentityInsertConflictIgnoresUnknownDatabaseErrors(t *testing.T) {
	for _, err := range []error{
		errors.New("connect failed"),
		&pgconn.PgError{Code: "23505", ConstraintName: "other_unique_constraint"},
		&pgconn.PgError{Code: "23503", ConstraintName: "admin_identities_actor_key"},
	} {
		if got := adminIdentityInsertConflict(err); got != nil {
			t.Fatalf("expected no mapped conflict for %T %#v, got %v", err, err, got)
		}
	}
}
