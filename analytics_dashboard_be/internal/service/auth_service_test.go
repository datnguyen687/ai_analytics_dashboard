package service

import (
	"context"
	"testing"

	"analytics-dashboard-be/internal/domain"
)

func newAuthSvc(t *testing.T) (*AuthService, *fakeUserRepo) {
	t.Helper()
	repo := &fakeUserRepo{users: map[string]domain.User{}}
	hash, err := HashPassword("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	repo.users["alice"] = domain.User{Username: "alice", PasswordHash: hash, Role: domain.RoleUser}
	repo.users["boss"] = domain.User{Username: "boss", PasswordHash: hash, Role: domain.RoleAdmin}
	return NewAuthService(repo, "test-secret", 24), repo
}

func TestHashPasswordSaltsAndVerifies(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Fatal("identical passwords must hash differently (salt)")
	}
	if len(h1) < 40 || h1[:4] != "$2a$" {
		t.Fatalf("not a bcrypt hash: %q", h1)
	}
}

func TestLoginSuccessAndTokenRoundtrip(t *testing.T) {
	svc, _ := newAuthSvc(t)

	token, user, err := svc.Login(context.Background(), "alice", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "alice" || user.Role != domain.RoleUser {
		t.Fatalf("identity = %+v", user)
	}
	if !hasClaim(user.Claims, domain.ClaimDashboardView) {
		t.Fatal("USER should have dashboard:view")
	}

	got, err := svc.Validate(token)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "alice" {
		t.Fatalf("validated username = %q", got.Username)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc, _ := newAuthSvc(t)
	_, _, err := svc.Login(context.Background(), "alice", "nope")
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	svc, _ := newAuthSvc(t)
	_, _, err := svc.Login(context.Background(), "ghost", "s3cret")
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("err = %v, want ErrInvalidCredentials (no user enumeration)", err)
	}
}

func TestAdminHasManageClaim(t *testing.T) {
	svc, _ := newAuthSvc(t)
	_, user, err := svc.Login(context.Background(), "boss", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if !hasClaim(user.Claims, domain.ClaimAdminManage) {
		t.Fatal("ADMIN should have admin:manage")
	}
}

func TestValidateRejectsGarbage(t *testing.T) {
	svc, _ := newAuthSvc(t)
	if _, err := svc.Validate("not.a.jwt"); err != domain.ErrTokenInvalid {
		t.Fatalf("err = %v, want ErrTokenInvalid", err)
	}
}

func TestValidateRejectsWrongSecret(t *testing.T) {
	svc, _ := newAuthSvc(t)
	token, _, _ := svc.Login(context.Background(), "alice", "s3cret")

	other := NewAuthService(&fakeUserRepo{users: map[string]domain.User{}}, "different-secret", 24)
	if _, err := other.Validate(token); err != domain.ErrTokenInvalid {
		t.Fatalf("err = %v, want ErrTokenInvalid for wrong signing key", err)
	}
}

func TestUsersList(t *testing.T) {
	svc, _ := newAuthSvc(t)
	users, err := svc.Users(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("users = %d, want 2", len(users))
	}
	for _, u := range users {
		if u.PasswordHash != "" {
			t.Fatal("List must not expose password hashes")
		}
	}
}

func hasClaim(claims []domain.Claim, want domain.Claim) bool {
	for _, c := range claims {
		if c == want {
			return true
		}
	}
	return false
}
