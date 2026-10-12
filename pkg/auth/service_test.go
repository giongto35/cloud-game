package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/store"
)

const adminEmail = "wenderson@belko.com.br"

func newService(t *testing.T, moderators bool) *Service {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	s := NewService(st, config.Auth{AdminEmail: adminEmail, AdminPassword: "admin-pass-123", Moderators: moderators},
		logger.Default())
	if err := s.EnsureAdmin(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s
}

func login(t *testing.T, s *Service, email, pass string) (*Account, string) {
	t.Helper()
	acc, token, err := s.Login(context.Background(), email, pass, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("login %s: %v", email, err)
	}
	return acc, token
}

func TestPasswordHash(t *testing.T) {
	h, err := HashPassword("segredo-forte")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := VerifyPassword("segredo-forte", h); !ok {
		t.Fatal("expected password to match")
	}
	if ok, _ := VerifyPassword("outra", h); ok {
		t.Fatal("expected password mismatch")
	}
	if _, err := VerifyPassword("x", "garbage"); err == nil {
		t.Fatal("expected malformed hash error")
	}
}

func TestAdminSeedAndLogin(t *testing.T) {
	s := newService(t, false)
	ctx := context.Background()
	// idempotent
	if err := s.EnsureAdmin(ctx); err != nil {
		t.Fatal(err)
	}
	acc, token := login(t, s, "WENDERSON@belko.com.br", "admin-pass-123")
	if !acc.Primary || acc.Role != store.RoleAdmin || !acc.Can(PermUsersManage) {
		t.Fatalf("unexpected admin account: %+v", acc)
	}
	again, err := s.Authenticate(ctx, token)
	if err != nil || again.Id != acc.Id {
		t.Fatalf("authenticate: %v", err)
	}
	if err := s.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected unauthenticated after logout, got %v", err)
	}
	if _, _, err := s.Login(ctx, adminEmail, "wrong-password", "", ""); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("expected bad credentials, got %v", err)
	}
}

func TestRegisterApproveFlow(t *testing.T) {
	s := newService(t, false)
	ctx := context.Background()

	pendingNotified := -1
	s.SetEvents(Events{PendingChanged: func(n int) { pendingNotified = n }})

	if _, err := s.Register(ctx, "A", "x@y.com", "12345678"); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("expected invalid name, got %v", err)
	}
	if _, err := s.Register(ctx, "Ana", "not-an-email", "12345678"); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected invalid email, got %v", err)
	}
	if _, err := s.Register(ctx, "Ana", "ana@belko.com.br", "123"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected weak password, got %v", err)
	}
	user, err := s.Register(ctx, "  Ana   Souza ", "Ana@Belko.com.br", "senha-da-ana")
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Ana Souza" || user.Email != "ana@belko.com.br" || user.Status != store.StatusPending {
		t.Fatalf("unexpected user %+v", user)
	}
	if pendingNotified != 1 {
		t.Fatalf("expected pending notification 1, got %d", pendingNotified)
	}
	if _, err := s.Register(ctx, "Ana 2", "ana@belko.com.br", "senha-da-ana"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected email taken, got %v", err)
	}

	// pending user can log in but cannot play
	pending, _ := login(t, s, "ana@belko.com.br", "senha-da-ana")
	if pending.Can(PermPlay) {
		t.Fatal("pending user must not play")
	}

	admin, _ := login(t, s, adminEmail, "admin-pass-123")
	if _, err := s.SetStatus(ctx, pending, admin.Id, store.StatusApproved); !errors.Is(err, ErrForbidden) {
		t.Fatalf("pending user must not approve, got %v", err)
	}
	if _, err := s.SetStatus(ctx, admin, user.Id, store.StatusApproved); err != nil {
		t.Fatal(err)
	}
	if pendingNotified != 0 {
		t.Fatalf("expected pending notification 0, got %d", pendingNotified)
	}
	approved, _ := login(t, s, "ana@belko.com.br", "senha-da-ana")
	if !approved.Can(PermPlay) || approved.Can(PermAdminView) {
		t.Fatalf("unexpected perms %+v", approved.Perms)
	}
}

func TestPrimaryAdminProtected(t *testing.T) {
	s := newService(t, true)
	ctx := context.Background()
	admin, _ := login(t, s, adminEmail, "admin-pass-123")
	u, _ := s.Register(ctx, "Bruno", "bruno@belko.com.br", "senha-do-bruno")
	_, _ = s.SetStatus(ctx, admin, u.Id, store.StatusApproved)
	if _, err := s.SetRole(ctx, admin, u.Id, store.RoleModerator); err != nil {
		t.Fatal(err)
	}
	mod, _ := login(t, s, "bruno@belko.com.br", "senha-do-bruno")
	if !mod.Can(PermUsersApprove) || mod.Can(PermUsersManage) {
		t.Fatalf("unexpected moderator perms %+v", mod.Perms)
	}
	if _, err := s.SetStatus(ctx, mod, admin.Id, store.StatusBlocked); err == nil {
		t.Fatal("moderator must not block the admin")
	}
	if _, err := s.SetStatus(ctx, admin, admin.Id, store.StatusBlocked); !errors.Is(err, ErrSelfAction) {
		t.Fatalf("expected self action error, got %v", err)
	}
	// moderator approves a pending user
	c, _ := s.Register(ctx, "Carla", "carla@belko.com.br", "senha-da-carla")
	if _, err := s.SetStatus(ctx, mod, c.Id, store.StatusApproved); err != nil {
		t.Fatal(err)
	}
	// but cannot block approved ones
	if _, err := s.SetStatus(ctx, mod, c.Id, store.StatusBlocked); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestModeratorsDisabledByDefault(t *testing.T) {
	s := newService(t, false)
	ctx := context.Background()
	admin, _ := login(t, s, adminEmail, "admin-pass-123")
	u, _ := s.Register(ctx, "Davi", "davi@belko.com.br", "senha-do-davi")
	if _, err := s.SetRole(ctx, admin, u.Id, store.RoleModerator); !errors.Is(err, ErrModeratorsOff) {
		t.Fatalf("expected moderators disabled, got %v", err)
	}
	if _, err := s.SetRole(ctx, admin, u.Id, store.RoleAdmin); !errors.Is(err, ErrAdminReserved) {
		t.Fatalf("expected admin reserved, got %v", err)
	}
	if PermsFor(store.RoleModerator, false).Has(PermUsersApprove) {
		t.Fatal("disabled moderators must not approve")
	}
}

func TestBlockRevokesSessions(t *testing.T) {
	s := newService(t, false)
	ctx := context.Background()
	admin, _ := login(t, s, adminEmail, "admin-pass-123")
	u, _ := s.Register(ctx, "Eva", "eva@belko.com.br", "senha-da-eva")
	_, _ = s.SetStatus(ctx, admin, u.Id, store.StatusApproved)
	_, token := login(t, s, "eva@belko.com.br", "senha-da-eva")
	if _, err := s.SetStatus(ctx, admin, u.Id, store.StatusBlocked); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected revoked session, got %v", err)
	}
	pass, err := s.ResetPassword(ctx, admin, u.Id)
	if err != nil || len(pass) != 12 {
		t.Fatalf("reset password: %v %q", err, pass)
	}
	entries, err := s.AuditLog(ctx, admin, 50)
	if err != nil || len(entries) < 4 {
		t.Fatalf("audit: %v (%d entries)", err, len(entries))
	}
}

func TestLimiter(t *testing.T) {
	l := NewLimiter(60, 2)
	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("burst should be allowed")
	}
	if l.Allow("a") {
		t.Fatal("third immediate call must be limited")
	}
	if !l.Allow("b") {
		t.Fatal("other keys are independent")
	}
}
