package profile

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"app/models"
)

type fakeRepo struct {
	user            *models.User
	getErr          error
	updateErr       error
	deleteChatsErr  error
	softDeleteErr   error
	updatedName     string
	deletedChatsFor string
	deletedUserFor  string
}

func (f *fakeRepo) GetByID(ctx context.Context, tx *sql.Tx, userID string) (*models.User, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.user == nil {
		return nil, sql.ErrNoRows
	}
	return f.user, nil
}

func (f *fakeRepo) UpdateName(ctx context.Context, tx *sql.Tx, userID, name string) error {
	f.updatedName = name
	return f.updateErr
}

func (f *fakeRepo) DeleteUserChats(ctx context.Context, tx *sql.Tx, userID string) error {
	f.deletedChatsFor = userID
	return f.deleteChatsErr
}

func (f *fakeRepo) SoftDeleteAccount(ctx context.Context, tx *sql.Tx, userID string) error {
	f.deletedUserFor = userID
	return f.softDeleteErr
}

func TestUpdateNameValidation(t *testing.T) {
	repo := &fakeRepo{user: &models.User{ID: "u1", Name: "Alice", Email: "alice@test.local"}}
	svc := NewService(repo)

	_, err := svc.UpdateName(context.Background(), nil, "u1", " ")
	if err == nil || err.Code != "invalid_name" {
		t.Fatalf("expected invalid_name, got %#v", err)
	}
}

func TestUpdateNameSuccess(t *testing.T) {
	repo := &fakeRepo{user: &models.User{ID: "u1", Name: "Alice", Email: "alice@test.local"}}
	svc := NewService(repo)

	profile, err := svc.UpdateName(context.Background(), nil, "u1", "  Novo Nome  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updatedName != "Novo Nome" {
		t.Fatalf("expected trimmed name, got %q", repo.updatedName)
	}
	if profile.ID != "u1" {
		t.Fatalf("expected profile id u1, got %s", profile.ID)
	}
}

func TestDeleteAccountStopsOnDeleteChatError(t *testing.T) {
	repo := &fakeRepo{
		user:           &models.User{ID: "u1"},
		deleteChatsErr: errors.New("boom"),
	}
	svc := NewService(repo)

	err := svc.DeleteAccount(context.Background(), nil, "u1")
	if err == nil || err.Code != "internal_error" {
		t.Fatalf("expected internal error, got %#v", err)
	}
	if repo.deletedUserFor != "" {
		t.Fatal("soft delete must not run when chat delete fails")
	}
}
