package domain

import "time"

// Role is the user's access role. Each role maps to a set of claims.
type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

// Claim is a fine-grained permission. Routes require a claim; roles grant them.
type Claim string

const (
	ClaimDashboardView Claim = "dashboard:view" // read analytics — both roles
	ClaimAdminManage   Claim = "admin:manage"   // admin-only (reserved for future use)
)

// roleClaims is the authoritative role → claims mapping. Both roles can use the
// current UI, so both carry dashboard:view; only ADMIN carries admin:manage.
var roleClaims = map[Role][]Claim{
	RoleUser:  {ClaimDashboardView},
	RoleAdmin: {ClaimDashboardView, ClaimAdminManage},
}

// ClaimsFor returns the claims granted to a role.
func ClaimsFor(role Role) []Claim {
	return roleClaims[role]
}

// HasClaim reports whether a role grants a claim.
func HasClaim(role Role, claim Claim) bool {
	for _, c := range roleClaims[role] {
		if c == claim {
			return true
		}
	}
	return false
}

// User is an account. The password is NEVER stored or returned in plaintext —
// only a bcrypt hash is persisted (json:"-" keeps the hash out of responses).
type User struct {
	ID           int64     `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         Role      `db:"role" json:"role"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
}
