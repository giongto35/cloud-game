package auth

import "github.com/giongto35/cloud-game/v3/pkg/store"

// Perm is a single capability that can be granted to a role.
type Perm string

const (
	PermPlay          Perm = "play"
	PermUsersApprove  Perm = "users.approve"
	PermUsersManage   Perm = "users.manage"
	PermRoomsModerate Perm = "rooms.moderate"
	PermAdminView     Perm = "admin.view"
)

// rolePerms is the single source of truth for what each role can do.
// Moderator permissions only apply when moderators are enabled in the config,
// which keeps the system admin-only at first but ready to delegate approvals.
var rolePerms = map[store.Role][]Perm{
	store.RoleAdmin:     {PermPlay, PermUsersApprove, PermUsersManage, PermRoomsModerate, PermAdminView},
	store.RoleModerator: {PermPlay, PermUsersApprove, PermRoomsModerate, PermAdminView},
	store.RoleUser:      {PermPlay},
}

// PermSet is a set of permissions.
type PermSet map[Perm]bool

func (p PermSet) Has(perm Perm) bool { return p[perm] }

func (p PermSet) List() []Perm {
	out := make([]Perm, 0, len(p))
	for _, perm := range []Perm{PermPlay, PermUsersApprove, PermUsersManage, PermRoomsModerate, PermAdminView} {
		if p[perm] {
			out = append(out, perm)
		}
	}
	return out
}

// PermsFor returns the effective permissions of a role.
func PermsFor(role store.Role, moderatorsEnabled bool) PermSet {
	set := PermSet{}
	r := role
	if r == store.RoleModerator && !moderatorsEnabled {
		r = store.RoleUser
	}
	for _, p := range rolePerms[r] {
		set[p] = true
	}
	return set
}
