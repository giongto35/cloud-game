// Package auth implements accounts, sessions and access control.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/store"
)

// Error is a user-facing error with a stable code, a pt-BR message and an HTTP status.
type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func newErr(status int, code, msg string) *Error {
	return &Error{Code: code, Message: msg, Status: status}
}

var (
	ErrInvalidName       = newErr(400, "invalid_name", "Informe um nome entre 2 e 40 caracteres.")
	ErrInvalidEmail      = newErr(400, "invalid_email", "Informe um e-mail válido.")
	ErrWeakPassword      = newErr(400, "weak_password", "A senha precisa ter entre 8 e 128 caracteres.")
	ErrEmailTaken        = newErr(409, "email_taken", "Já existe uma conta com este e-mail.")
	ErrBadCredentials    = newErr(401, "bad_credentials", "E-mail ou senha incorretos.")
	ErrWrongPassword     = newErr(400, "wrong_password", "A senha atual está incorreta.")
	ErrUnauthenticated   = newErr(401, "unauthenticated", "Sua sessão expirou. Entre novamente.")
	ErrForbidden         = newErr(403, "forbidden", "Você não tem permissão para esta ação.")
	ErrNotFound          = newErr(404, "not_found", "Usuário não encontrado.")
	ErrProtectedAccount  = newErr(403, "protected_account", "O administrador principal não pode ser alterado.")
	ErrSelfAction        = newErr(400, "self_action", "Você não pode fazer isso com a sua própria conta.")
	ErrInvalidRole       = newErr(400, "invalid_role", "Papel inválido.")
	ErrModeratorsOff     = newErr(400, "moderators_disabled", "O papel de moderador está desativado na configuração.")
	ErrAdminReserved     = newErr(400, "admin_reserved", "Somente a conta principal pode ser administradora.")
	ErrTooManyAttempts   = newErr(429, "too_many_attempts", "Muitas tentativas. Aguarde um minuto e tente novamente.")
	ErrPreferencesTooBig = newErr(413, "preferences_too_big", "As preferências são grandes demais.")
)

// Account is the authenticated identity attached to requests and connections.
type Account struct {
	Id      int64
	Email   string
	Name    string
	Role    store.Role
	Status  store.Status
	Perms   PermSet
	Primary bool // the protected primary admin
}

func (a *Account) Can(p Perm) bool {
	return a != nil && a.Status == store.StatusApproved && a.Perms.Has(p)
}

// Events lets the rest of the system react to account changes.
type Events struct {
	// PendingChanged is called when the number of pending accounts changes.
	PendingChanged func(count int)
	// AccountChanged is called when an account was blocked, rejected or its role changed.
	AccountChanged func(id int64)
}

type Service struct {
	store *store.Store
	conf  config.Auth
	log   *logger.Logger

	sessionTTL time.Duration

	LoginLimiter    *Limiter
	RegisterLimiter *Limiter

	mu     sync.RWMutex
	events Events
}

const (
	sessionTouchEvery = 5 * time.Minute
	maxPreferences    = 64 * 1024
)

func NewService(st *store.Store, conf config.Auth, log *logger.Logger) *Service {
	if conf.SessionDays <= 0 {
		conf.SessionDays = 30
	}
	conf.AdminEmail = strings.ToLower(strings.TrimSpace(conf.AdminEmail))
	return &Service{
		store:           st,
		conf:            conf,
		log:             log,
		sessionTTL:      time.Duration(conf.SessionDays) * 24 * time.Hour,
		LoginLimiter:    NewLimiter(10, 10),
		RegisterLimiter: NewLimiter(5, 5),
	}
}

func (s *Service) SetEvents(e Events) { s.mu.Lock(); s.events = e; s.mu.Unlock() }

func (s *Service) SessionTTL() time.Duration { return s.sessionTTL }

func (s *Service) ModeratorsEnabled() bool { return s.conf.Moderators }

// EnsureAdmin creates the primary admin on the first start and makes sure it keeps admin rights.
func (s *Service) EnsureAdmin(ctx context.Context) error {
	if s.conf.AdminEmail == "" {
		return errors.New("auth: admin email is not configured")
	}
	u, err := s.store.UserByEmail(ctx, s.conf.AdminEmail)
	if errors.Is(err, store.ErrNotFound) {
		password := s.conf.AdminPassword
		generated := password == ""
		if generated {
			password = RandomPassword(16)
		}
		hash, err := HashPassword(password)
		if err != nil {
			return err
		}
		now := time.Now()
		admin := &store.User{
			Email: s.conf.AdminEmail, Name: "Administrador", PasswordHash: hash,
			Role: store.RoleAdmin, Status: store.StatusApproved, ApprovedAt: &now,
		}
		if err := s.store.CreateUser(ctx, admin); err != nil {
			return fmt.Errorf("create admin: %w", err)
		}
		_ = s.store.Audit(ctx, nil, "admin.created", &admin.Id, "")
		if generated {
			s.log.Warn().Msgf("==================================================================")
			s.log.Warn().Msgf(" Conta de administrador criada: %s", s.conf.AdminEmail)
			s.log.Warn().Msgf(" Senha inicial: %s", password)
			s.log.Warn().Msgf(" Altere a senha em Configurações > Conta após o primeiro acesso.")
			s.log.Warn().Msgf("==================================================================")
		} else {
			s.log.Info().Msgf("Admin account created: %s", s.conf.AdminEmail)
		}
		return nil
	}
	if err != nil {
		return err
	}
	// the primary admin must always stay an approved admin
	if u.Role != store.RoleAdmin {
		if err := s.store.SetUserRole(ctx, u.Id, store.RoleAdmin); err != nil {
			return err
		}
	}
	if u.Status != store.StatusApproved {
		if err := s.store.SetUserStatus(ctx, u.Id, store.StatusApproved, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) toAccount(u *store.User) *Account {
	return &Account{
		Id: u.Id, Email: u.Email, Name: u.Name, Role: u.Role, Status: u.Status,
		Perms:   PermsFor(u.Role, s.conf.Moderators),
		Primary: strings.EqualFold(u.Email, s.conf.AdminEmail),
	}
}

// Validation

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) > 254 {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email[strings.LastIndex(email, "@"):], ".") {
		return "", ErrInvalidEmail
	}
	return email, nil
}

func normalizeName(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if n := utf8.RuneCountInString(name); n < 2 || n > 40 {
		return "", ErrInvalidName
	}
	return name, nil
}

func validatePassword(p string) error {
	if n := utf8.RuneCountInString(p); n < 8 || n > 128 {
		return ErrWeakPassword
	}
	return nil
}

// Registration and login

// Register creates a pending account.
func (s *Service) Register(ctx context.Context, name, email, password string) (*Account, error) {
	var err error
	if name, err = normalizeName(name); err != nil {
		return nil, err
	}
	if email, err = normalizeEmail(email); err != nil {
		return nil, err
	}
	if err = validatePassword(password); err != nil {
		return nil, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &store.User{Email: email, Name: name, PasswordHash: hash, Role: store.RoleUser, Status: store.StatusPending}
	if err := s.store.CreateUser(ctx, u); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	_ = s.store.Audit(ctx, &u.Id, "user.registered", &u.Id, "")
	s.notifyPending(ctx)
	return s.toAccount(u), nil
}

// dummyHash is used to keep login timing similar for unknown emails.
var dummyHash, _ = HashPassword("belko-arcade-dummy-password")

// Login verifies credentials and returns a new session token.
func (s *Service) Login(ctx context.Context, email, password, ip, ua string) (*Account, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		_, _ = VerifyPassword(password, dummyHash)
		return nil, "", ErrBadCredentials
	}
	if err != nil {
		return nil, "", err
	}
	ok, err := VerifyPassword(password, u.PasswordHash)
	if err != nil || !ok {
		return nil, "", ErrBadCredentials
	}
	token, err := s.createSession(ctx, u.Id, ip, ua)
	if err != nil {
		return nil, "", err
	}
	_ = s.store.TouchLogin(ctx, u.Id)
	return s.toAccount(u), token, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Service) createSession(ctx context.Context, userId int64, ip, ua string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now()
	if len(ua) > 255 {
		ua = ua[:255]
	}
	err := s.store.CreateSession(ctx, store.Session{
		TokenHash: hashToken(token), UserId: userId, CreatedAt: now, ExpiresAt: now.Add(s.sessionTTL),
		LastSeenAt: now, Ip: ip, UserAgent: ua,
	})
	return token, err
}

// Authenticate resolves a session token into an account (of any status).
func (s *Service) Authenticate(ctx context.Context, token string) (*Account, error) {
	if token == "" || len(token) > 128 {
		return nil, ErrUnauthenticated
	}
	h := hashToken(token)
	ss, err := s.store.SessionByHash(ctx, h)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrUnauthenticated
	}
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if now.After(ss.ExpiresAt) {
		_ = s.store.DeleteSession(ctx, h)
		return nil, ErrUnauthenticated
	}
	u, err := s.store.UserById(ctx, ss.UserId)
	if err != nil {
		return nil, ErrUnauthenticated
	}
	if now.Sub(ss.LastSeenAt) > sessionTouchEvery {
		_ = s.store.TouchSession(ctx, h, now, now.Add(s.sessionTTL))
	}
	return s.toAccount(u), nil
}

// AccountById loads the current state of an account.
func (s *Service) AccountById(ctx context.Context, id int64) (*Account, error) {
	u, err := s.store.UserById(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.toAccount(u), nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, hashToken(token))
}

// Account self-service

func (s *Service) UpdateName(ctx context.Context, acc *Account, name string) (string, error) {
	name, err := normalizeName(name)
	if err != nil {
		return "", err
	}
	return name, s.store.SetUserName(ctx, acc.Id, name)
}

func (s *Service) ChangePassword(ctx context.Context, acc *Account, token, current, next string) error {
	u, err := s.store.UserById(ctx, acc.Id)
	if err != nil {
		return err
	}
	if ok, _ := VerifyPassword(current, u.PasswordHash); !ok {
		return ErrWrongPassword
	}
	if err := validatePassword(next); err != nil {
		return err
	}
	hash, err := HashPassword(next)
	if err != nil {
		return err
	}
	if err := s.store.SetUserPassword(ctx, acc.Id, hash); err != nil {
		return err
	}
	// sign out every other device
	_ = s.store.DeleteUserSessions(ctx, acc.Id, hashToken(token))
	_ = s.store.Audit(ctx, &acc.Id, "user.password_changed", &acc.Id, "")
	return nil
}

func (s *Service) Preferences(ctx context.Context, acc *Account) (string, error) {
	return s.store.Preferences(ctx, acc.Id)
}

func (s *Service) SetPreferences(ctx context.Context, acc *Account, data string) error {
	if len(data) > maxPreferences {
		return ErrPreferencesTooBig
	}
	return s.store.SetPreferences(ctx, acc.Id, data)
}

// Administration

func (s *Service) ListUsers(ctx context.Context, actor *Account, status store.Status) ([]store.User, error) {
	if !actor.Can(PermUsersApprove) && !actor.Can(PermUsersManage) {
		return nil, ErrForbidden
	}
	return s.store.ListUsers(ctx, status)
}

func (s *Service) PendingCount(ctx context.Context) int {
	n, _ := s.store.CountUsers(ctx, store.StatusPending)
	return n
}

func (s *Service) IsPrimary(u *store.User) bool { return strings.EqualFold(u.Email, s.conf.AdminEmail) }

// target loads and checks an account that is about to be changed by an actor.
func (s *Service) target(ctx context.Context, actor *Account, id int64) (*store.User, error) {
	if actor.Id == id {
		return nil, ErrSelfAction
	}
	u, err := s.store.UserById(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if s.IsPrimary(u) {
		return nil, ErrProtectedAccount
	}
	return u, nil
}

// SetStatus approves, rejects, blocks or unblocks an account.
func (s *Service) SetStatus(ctx context.Context, actor *Account, id int64, status store.Status) (*store.User, error) {
	switch status {
	case store.StatusApproved, store.StatusRejected:
		if !actor.Can(PermUsersApprove) {
			return nil, ErrForbidden
		}
	case store.StatusBlocked:
		if !actor.Can(PermUsersManage) {
			return nil, ErrForbidden
		}
	default:
		return nil, ErrForbidden
	}
	u, err := s.target(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	// moderators can only decide on pending accounts
	if !actor.Can(PermUsersManage) && u.Status != store.StatusPending {
		return nil, ErrForbidden
	}
	// only admins can touch other privileged accounts
	if u.Role != store.RoleUser && !actor.Can(PermUsersManage) {
		return nil, ErrForbidden
	}
	if err := s.store.SetUserStatus(ctx, id, status, &actor.Id); err != nil {
		return nil, err
	}
	if status != store.StatusApproved {
		_ = s.store.DeleteUserSessions(ctx, id, "")
	}
	action := map[store.Status]string{
		store.StatusApproved: "user.approved", store.StatusRejected: "user.rejected", store.StatusBlocked: "user.blocked",
	}[status]
	if status == store.StatusApproved && u.Status == store.StatusBlocked {
		action = "user.unblocked"
	}
	_ = s.store.Audit(ctx, &actor.Id, action, &id, "")
	s.notifyPending(ctx)
	s.notifyAccount(id)
	return s.store.UserById(ctx, id)
}

func (s *Service) SetRole(ctx context.Context, actor *Account, id int64, role store.Role) (*store.User, error) {
	if !actor.Can(PermUsersManage) {
		return nil, ErrForbidden
	}
	if !role.Valid() {
		return nil, ErrInvalidRole
	}
	// the primary account is the only administrator
	if role == store.RoleAdmin {
		return nil, ErrAdminReserved
	}
	if role == store.RoleModerator && !s.conf.Moderators {
		return nil, ErrModeratorsOff
	}
	u, err := s.target(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if err := s.store.SetUserRole(ctx, id, role); err != nil {
		return nil, err
	}
	_ = s.store.Audit(ctx, &actor.Id, "user.role_changed", &id, string(u.Role)+" → "+string(role))
	s.notifyAccount(id)
	return s.store.UserById(ctx, id)
}

// ResetPassword sets a new random password and returns it to show once to the admin.
func (s *Service) ResetPassword(ctx context.Context, actor *Account, id int64) (string, error) {
	if !actor.Can(PermUsersManage) {
		return "", ErrForbidden
	}
	if _, err := s.target(ctx, actor, id); err != nil {
		return "", err
	}
	password := RandomPassword(12)
	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}
	if err := s.store.SetUserPassword(ctx, id, hash); err != nil {
		return "", err
	}
	_ = s.store.DeleteUserSessions(ctx, id, "")
	_ = s.store.Audit(ctx, &actor.Id, "user.password_reset", &id, "")
	s.notifyAccount(id)
	return password, nil
}

func (s *Service) AuditLog(ctx context.Context, actor *Account, limit int) ([]store.AuditEntry, error) {
	if !actor.Can(PermUsersManage) {
		return nil, ErrForbidden
	}
	return s.store.AuditLog(ctx, limit)
}

func (s *Service) Audit(ctx context.Context, actor *Account, action string, details string) {
	var id *int64
	if actor != nil {
		id = &actor.Id
	}
	_ = s.store.Audit(ctx, id, action, nil, details)
}

func (s *Service) notifyPending(ctx context.Context) {
	s.mu.RLock()
	fn := s.events.PendingChanged
	s.mu.RUnlock()
	if fn != nil {
		fn(s.PendingCount(ctx))
	}
}

func (s *Service) notifyAccount(id int64) {
	s.mu.RLock()
	fn := s.events.AccountChanged
	s.mu.RUnlock()
	if fn != nil {
		fn(id)
	}
}

// RunJanitor periodically removes expired sessions until ctx is done.
func (s *Service) RunJanitor(ctx context.Context) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		_ = s.store.PurgeExpiredSessions(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
