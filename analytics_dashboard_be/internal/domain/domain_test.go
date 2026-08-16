package domain

import (
	"net/http"
	"testing"
)

func TestClaimsFor(t *testing.T) {
	if !HasClaim(RoleUser, ClaimDashboardView) {
		t.Fatal("USER should have dashboard:view")
	}
	if HasClaim(RoleUser, ClaimAdminManage) {
		t.Fatal("USER must not have admin:manage")
	}
	if !HasClaim(RoleAdmin, ClaimAdminManage) || !HasClaim(RoleAdmin, ClaimDashboardView) {
		t.Fatal("ADMIN should have both claims")
	}
	if len(ClaimsFor(RoleAdmin)) != 2 {
		t.Fatalf("admin claims = %d, want 2", len(ClaimsFor(RoleAdmin)))
	}
	if HasClaim("NONSENSE", ClaimDashboardView) {
		t.Fatal("unknown role should have no claims")
	}
}

func TestAPIError(t *testing.T) {
	e := NewAPIError(http.StatusTeapot, "TEAPOT", "i am a teapot")
	if e.HTTPStatus() != http.StatusTeapot {
		t.Fatalf("status = %d", e.HTTPStatus())
	}
	if e.Error() != "TEAPOT: i am a teapot" {
		t.Fatalf("error string = %q", e.Error())
	}
}

func TestPredefinedErrorsHaveCodes(t *testing.T) {
	cases := []struct {
		err  *APIError
		code string
	}{
		{ErrInvalidCredentials, "AUTH_INVALID_CREDENTIALS"},
		{ErrTokenMissing, "AUTH_TOKEN_MISSING"},
		{ErrForbidden, "AUTH_FORBIDDEN"},
		{ErrRateLimited, "RATE_LIMITED"},
		{ErrPayloadTooLarge, "PAYLOAD_TOO_LARGE"},
		{ErrValidation, "VALIDATION_ERROR"},
		{ErrInternal, "INTERNAL_ERROR"},
	}
	for _, c := range cases {
		if c.err.Code != c.code {
			t.Errorf("code = %q, want %q", c.err.Code, c.code)
		}
	}
}
