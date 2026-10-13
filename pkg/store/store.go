// Package store keeps the persistent coordinator state (accounts, sessions,
// preferences and the audit trail) in an embedded SQLite database.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate")
)

type Role string

const (
	RoleAdmin     Role = "admin"
	RoleModerator Role = "moderator"
	RoleUser      Role = "user"
)

func (r Role) Valid() bool { return r == RoleAdmin || r == RoleModerator || r == RoleUser }

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
	StatusBlocked  Status = "blocked"
)

type User struct {
	Id           int64
	Email        string
	Name         string
	PasswordHash string
	Role         Role
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ApprovedAt   *time.Time
	ApprovedBy   *int64
	LastLoginAt  *time.Time
}

type Session struct {
	TokenHash  string
	UserId     int64
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastSeenAt time.Time
	Ip         string
	UserAgent  string
}

type AuditEntry struct {
	Id        int64
	ActorId   *int64
	ActorName string
	Action    string
	TargetId  *int64
	Target    string
	Details   string
	CreatedAt time.Time
}

type Store struct {
	db *sql.DB
}

// Open opens (and creates if needed) the database at path and applies migrations.
// Use ":memory:" for an ephemeral database (tests).
func Open(path string) (*Store, error) {
	dsn := path
	if path != ":memory:" {
		if dir := filepath.Dir(path); dir != "" {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return nil, fmt.Errorf("create db dir: %w", err)
			}
		}
		dsn = "file:" + path
	}
	dsn += sep(dsn) + "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if path == ":memory:" {
		// every new connection of an in-memory database is a new empty database
		db.SetMaxOpenConns(1)
	}
	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func sep(dsn string) string {
	if strings.Contains(dsn, "?") {
		return "&"
	}
	return "?"
}

func (s *Store) Close() error { return s.db.Close() }

var migrations = []string{
	// 1: initial schema
	`CREATE TABLE users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		email         TEXT    NOT NULL UNIQUE COLLATE NOCASE,
		name          TEXT    NOT NULL,
		password_hash TEXT    NOT NULL,
		role          TEXT    NOT NULL DEFAULT 'user',
		status        TEXT    NOT NULL DEFAULT 'pending',
		created_at    INTEGER NOT NULL,
		updated_at    INTEGER NOT NULL,
		approved_at   INTEGER,
		approved_by   INTEGER REFERENCES users(id) ON DELETE SET NULL,
		last_login_at INTEGER
	);
	CREATE INDEX users_status ON users(status);
	CREATE TABLE sessions (
		token_hash   TEXT    PRIMARY KEY,
		user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at   INTEGER NOT NULL,
		expires_at   INTEGER NOT NULL,
		last_seen_at INTEGER NOT NULL,
		ip           TEXT    NOT NULL DEFAULT '',
		user_agent   TEXT    NOT NULL DEFAULT ''
	);
	CREATE INDEX sessions_user ON sessions(user_id);
	CREATE TABLE preferences (
		user_id    INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		data       TEXT    NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE TABLE audit_log (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		actor_id   INTEGER REFERENCES users(id) ON DELETE SET NULL,
		action     TEXT    NOT NULL,
		target_id  INTEGER REFERENCES users(id) ON DELETE SET NULL,
		details    TEXT    NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL
	);
	CREATE INDEX audit_created ON audit_log(created_at);`,
	// 2: game cover art cache
	`CREATE TABLE covers (
		key        TEXT    PRIMARY KEY,
		system     TEXT    NOT NULL,
		title      TEXT    NOT NULL,
		file       TEXT    NOT NULL DEFAULT '',
		source     TEXT    NOT NULL DEFAULT '',
		status     TEXT    NOT NULL,
		updated_at INTEGER NOT NULL
	);`,
	// 3: tournaments (single elimination), matches and replays
	`CREATE TABLE tournaments (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT    NOT NULL,
		game        TEXT    NOT NULL,
		system      TEXT    NOT NULL,
		rule        TEXT    NOT NULL,
		status      TEXT    NOT NULL,
		created_by  INTEGER REFERENCES users(id) ON DELETE SET NULL,
		winner_id   INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at  INTEGER NOT NULL,
		started_at  INTEGER,
		finished_at INTEGER
	);
	CREATE TABLE tournament_players (
		tournament_id INTEGER NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
		user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		joined_at     INTEGER NOT NULL,
		PRIMARY KEY (tournament_id, user_id)
	);
	CREATE TABLE matches (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		tournament_id INTEGER NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
		round         INTEGER NOT NULL,
		position      INTEGER NOT NULL,
		p1_id         INTEGER REFERENCES users(id) ON DELETE SET NULL,
		p2_id         INTEGER REFERENCES users(id) ON DELETE SET NULL,
		status        TEXT    NOT NULL,
		winner_id     INTEGER REFERENCES users(id) ON DELETE SET NULL,
		source        TEXT    NOT NULL DEFAULT '',
		score         TEXT    NOT NULL DEFAULT '',
		room_id       TEXT    NOT NULL DEFAULT '',
		replay_id     INTEGER,
		attempts      INTEGER NOT NULL DEFAULT 0,
		ready_at      INTEGER,
		started_at    INTEGER,
		finished_at   INTEGER,
		UNIQUE (tournament_id, round, position)
	);
	CREATE INDEX matches_status ON matches(status);
	CREATE TABLE replays (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id   INTEGER REFERENCES matches(id) ON DELETE SET NULL,
		game       TEXT    NOT NULL,
		frames     INTEGER NOT NULL,
		size       INTEGER NOT NULL,
		file       TEXT    NOT NULL,
		created_at INTEGER NOT NULL
	);`,
	`CREATE TABLE game_meta (
		key         TEXT    PRIMARY KEY,
		system      TEXT    NOT NULL,
		title       TEXT    NOT NULL,
		genre       TEXT    NOT NULL DEFAULT '',
		genre_raw   TEXT    NOT NULL DEFAULT '',
		year        INTEGER NOT NULL DEFAULT 0,
		developer   TEXT    NOT NULL DEFAULT '',
		publisher   TEXT    NOT NULL DEFAULT '',
		players     INTEGER NOT NULL DEFAULT 0,
		description TEXT    NOT NULL DEFAULT '',
		source      TEXT    NOT NULL DEFAULT '',
		status      TEXT    NOT NULL,
		updated_at  INTEGER NOT NULL
	);`,
	`CREATE TABLE user_avatars (
		user_id    INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		data       BLOB    NOT NULL,
		updated_at INTEGER NOT NULL
	);`,
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)`); err != nil {
		return err
	}
	var current int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return err
	}
	for i := current; i < len(migrations); i++ {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES (?)`, i+1); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// time helpers: everything is stored as unix seconds

func unix(t time.Time) int64 { return t.Unix() }

func fromUnix(v int64) time.Time { return time.Unix(v, 0).UTC() }

func fromNullUnix(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := fromUnix(v.Int64)
	return &t
}

func fromNullInt(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	x := v.Int64
	return &x
}

func isUniqueErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// Users

const userCols = `id, email, name, password_hash, role, status, created_at, updated_at,
	approved_at, approved_by, last_login_at`

type scanner interface{ Scan(dest ...any) error }

func scanUser(r scanner) (*User, error) {
	var u User
	var created, updated int64
	var approvedAt, approvedBy, lastLogin sql.NullInt64
	err := r.Scan(&u.Id, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.Status,
		&created, &updated, &approvedAt, &approvedBy, &lastLogin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt, u.UpdatedAt = fromUnix(created), fromUnix(updated)
	u.ApprovedAt, u.ApprovedBy, u.LastLoginAt = fromNullUnix(approvedAt), fromNullInt(approvedBy), fromNullUnix(lastLogin)
	return &u, nil
}

func (s *Store) CreateUser(ctx context.Context, u *User) error {
	now := time.Now()
	u.CreatedAt, u.UpdatedAt = now.UTC().Truncate(time.Second), now.UTC().Truncate(time.Second)
	var approvedAt any
	if u.ApprovedAt != nil {
		approvedAt = unix(*u.ApprovedAt)
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users(email, name, password_hash, role, status, created_at, updated_at, approved_at, approved_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Email, u.Name, u.PasswordHash, u.Role, u.Status, unix(now), unix(now), approvedAt, u.ApprovedBy)
	if isUniqueErr(err) {
		return ErrDuplicate
	}
	if err != nil {
		return err
	}
	u.Id, err = res.LastInsertId()
	return err
}

func (s *Store) UserById(ctx context.Context, id int64) (*User, error) {
	return scanUser(s.db.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE id = ?`, id))
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(s.db.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE email = ?`, email))
}

// ListUsers returns users filtered by status (empty = all), newest first.
func (s *Store) ListUsers(ctx context.Context, status Status) ([]User, error) {
	q := `SELECT ` + userCols + ` FROM users`
	var args []any
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC, id DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (s *Store) CountUsers(ctx context.Context, status Status) (n int, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE status = ?`, status).Scan(&n)
	return
}

func (s *Store) exec1(ctx context.Context, q string, args ...any) error {
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetUserStatus(ctx context.Context, id int64, status Status, actor *int64) error {
	now := unix(time.Now())
	if status == StatusApproved {
		return s.exec1(ctx, `UPDATE users SET status = ?, updated_at = ?, approved_at = ?, approved_by = ? WHERE id = ?`,
			status, now, now, actor, id)
	}
	return s.exec1(ctx, `UPDATE users SET status = ?, updated_at = ? WHERE id = ?`, status, now, id)
}

func (s *Store) SetUserRole(ctx context.Context, id int64, role Role) error {
	return s.exec1(ctx, `UPDATE users SET role = ?, updated_at = ? WHERE id = ?`, role, unix(time.Now()), id)
}

func (s *Store) SetUserName(ctx context.Context, id int64, name string) error {
	return s.exec1(ctx, `UPDATE users SET name = ?, updated_at = ? WHERE id = ?`, name, unix(time.Now()), id)
}

func (s *Store) SetUserPassword(ctx context.Context, id int64, hash string) error {
	return s.exec1(ctx, `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, hash, unix(time.Now()), id)
}

func (s *Store) TouchLogin(ctx context.Context, id int64) error {
	return s.exec1(ctx, `UPDATE users SET last_login_at = ? WHERE id = ?`, unix(time.Now()), id)
}

// Sessions

func (s *Store) CreateSession(ctx context.Context, ss Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions(token_hash, user_id, created_at, expires_at, last_seen_at, ip, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ss.TokenHash, ss.UserId, unix(ss.CreatedAt), unix(ss.ExpiresAt), unix(ss.LastSeenAt), ss.Ip, ss.UserAgent)
	return err
}

func (s *Store) SessionByHash(ctx context.Context, hash string) (*Session, error) {
	var ss Session
	var created, expires, seen int64
	err := s.db.QueryRowContext(ctx,
		`SELECT token_hash, user_id, created_at, expires_at, last_seen_at, ip, user_agent FROM sessions WHERE token_hash = ?`,
		hash).Scan(&ss.TokenHash, &ss.UserId, &created, &expires, &seen, &ss.Ip, &ss.UserAgent)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	ss.CreatedAt, ss.ExpiresAt, ss.LastSeenAt = fromUnix(created), fromUnix(expires), fromUnix(seen)
	return &ss, nil
}

func (s *Store) TouchSession(ctx context.Context, hash string, seen, expires time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE token_hash = ?`,
		unix(seen), unix(expires), hash)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hash)
	return err
}

// DeleteUserSessions removes all sessions of a user except the one given (may be empty).
func (s *Store) DeleteUserSessions(ctx context.Context, userId int64, except string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ? AND token_hash != ?`, userId, except)
	return err
}

func (s *Store) PurgeExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, unix(time.Now()))
	return err
}

// Preferences

func (s *Store) Preferences(ctx context.Context, userId int64) (string, error) {
	var data string
	err := s.db.QueryRowContext(ctx, `SELECT data FROM preferences WHERE user_id = ?`, userId).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return "{}", nil
	}
	return data, err
}

func (s *Store) SetPreferences(ctx context.Context, userId int64, data string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO preferences(user_id, data, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		userId, data, unix(time.Now()))
	return err
}

// Audit

func (s *Store) Audit(ctx context.Context, actor *int64, action string, target *int64, details string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_log(actor_id, action, target_id, details, created_at) VALUES (?, ?, ?, ?, ?)`,
		actor, action, target, details, unix(time.Now()))
	return err
}

func (s *Store) AuditLog(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT a.id, a.actor_id, COALESCE(ua.name, ''), a.action, a.target_id, COALESCE(ut.name || ' <' || ut.email || '>', ''),
		        a.details, a.created_at
		   FROM audit_log a
		   LEFT JOIN users ua ON ua.id = a.actor_id
		   LEFT JOIN users ut ON ut.id = a.target_id
		  ORDER BY a.created_at DESC, a.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var actor, target sql.NullInt64
		var created int64
		if err := rows.Scan(&e.Id, &actor, &e.ActorName, &e.Action, &target, &e.Target, &e.Details, &created); err != nil {
			return nil, err
		}
		e.ActorId, e.TargetId, e.CreatedAt = fromNullInt(actor), fromNullInt(target), fromUnix(created)
		out = append(out, e)
	}
	return out, rows.Err()
}

// Covers

const (
	CoverFound   = "found"
	CoverMissing = "missing"
)

// Cover is a cached cover art lookup (found: File is set; missing: nothing upstream).
type Cover struct {
	Key       string
	System    string
	Title     string
	File      string
	Source    string
	Status    string
	UpdatedAt time.Time
}

func (s *Store) Cover(ctx context.Context, key string) (*Cover, error) {
	var c Cover
	var updated int64
	err := s.db.QueryRowContext(ctx,
		`SELECT key, system, title, file, source, status, updated_at FROM covers WHERE key = ?`, key).
		Scan(&c.Key, &c.System, &c.Title, &c.File, &c.Source, &c.Status, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.UpdatedAt = fromUnix(updated)
	return &c, nil
}

func (s *Store) SetCover(ctx context.Context, c Cover) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO covers(key, system, title, file, source, status, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET file = excluded.file, source = excluded.source,
		   status = excluded.status, updated_at = excluded.updated_at`,
		c.Key, c.System, c.Title, c.File, c.Source, c.Status, unix(time.Now()))
	return err
}

// CoverStats returns how many covers are stored and how many lookups found nothing.
func (s *Store) CoverStats(ctx context.Context) (found, missing int, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(status = 'found'), 0), COALESCE(SUM(status = 'missing'), 0) FROM covers`).Scan(&found, &missing)
	return
}

// ForgetMissingCovers drops negative results so they are looked up again.
func (s *Store) ForgetMissingCovers(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM covers WHERE status = 'missing'`)
	return err
}

// Game catalog metadata

const (
	MetaFound   = "found"
	MetaMissing = "missing"
)

// GameMeta is the catalog data of a game (genre, year, developer, players).
type GameMeta struct {
	Key         string    `json:"-"`
	System      string    `json:"system"`
	Title       string    `json:"title"`
	Genre       string    `json:"genre"`
	GenreRaw    string    `json:"genreRaw,omitempty"`
	Year        int       `json:"year,omitempty"`
	Developer   string    `json:"developer,omitempty"`
	Publisher   string    `json:"publisher,omitempty"`
	Players     int       `json:"players,omitempty"`
	Description string    `json:"description,omitempty"`
	Source      string    `json:"source,omitempty"`
	Status      string    `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

const metaCols = `key, system, title, genre, genre_raw, year, developer, publisher, players, description, source, status, updated_at`

func scanMeta(sc interface{ Scan(...any) error }) (*GameMeta, error) {
	var m GameMeta
	var updated int64
	if err := sc.Scan(&m.Key, &m.System, &m.Title, &m.Genre, &m.GenreRaw, &m.Year, &m.Developer,
		&m.Publisher, &m.Players, &m.Description, &m.Source, &m.Status, &updated); err != nil {
		return nil, err
	}
	m.UpdatedAt = fromUnix(updated)
	return &m, nil
}

func (s *Store) GameMeta(ctx context.Context, key string) (*GameMeta, error) {
	m, err := scanMeta(s.db.QueryRowContext(ctx, `SELECT `+metaCols+` FROM game_meta WHERE key = ?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return m, err
}

func (s *Store) SetGameMeta(ctx context.Context, m GameMeta) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO game_meta(`+metaCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET genre = excluded.genre, genre_raw = excluded.genre_raw,
		   year = excluded.year, developer = excluded.developer, publisher = excluded.publisher,
		   players = excluded.players, description = excluded.description, source = excluded.source,
		   status = excluded.status, updated_at = excluded.updated_at`,
		m.Key, m.System, m.Title, m.Genre, m.GenreRaw, m.Year, m.Developer, m.Publisher,
		m.Players, m.Description, m.Source, m.Status, unix(time.Now()))
	return err
}

// AllGameMeta lists every game that has catalog data.
func (s *Store) AllGameMeta(ctx context.Context) ([]GameMeta, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+metaCols+` FROM game_meta WHERE status = 'found' ORDER BY system, title`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []GameMeta
	for rows.Next() {
		m, err := scanMeta(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (s *Store) GameMetaStats(ctx context.Context) (found, missing int, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(status = 'found'), 0), COALESCE(SUM(status = 'missing'), 0) FROM game_meta`).Scan(&found, &missing)
	return
}

// ForgetGameMeta drops negative results (and, when all is set, everything) so they are looked up again.
func (s *Store) ForgetGameMeta(ctx context.Context, all bool) error {
	q := `DELETE FROM game_meta WHERE status = 'missing'`
	if all {
		q = `DELETE FROM game_meta`
	}
	_, err := s.db.ExecContext(ctx, q)
	return err
}

// Avatars

// SetAvatar stores the profile picture (a PNG already validated and resized) of a user.
func (s *Store) SetAvatar(ctx context.Context, userId int64, png []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_avatars(user_id, data, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		userId, png, unix(time.Now()))
	return err
}

// Avatar returns the profile picture of a user (ErrNotFound when there is none).
func (s *Store) Avatar(ctx context.Context, userId int64) ([]byte, time.Time, error) {
	var data []byte
	var updated int64
	err := s.db.QueryRowContext(ctx, `SELECT data, updated_at FROM user_avatars WHERE user_id = ?`, userId).Scan(&data, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, time.Time{}, ErrNotFound
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	return data, fromUnix(updated), nil
}

func (s *Store) DeleteAvatar(ctx context.Context, userId int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_avatars WHERE user_id = ?`, userId)
	return err
}

// DeleteCover forgets one cover, so the next request looks it up again.
func (s *Store) DeleteCover(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM covers WHERE key = ?`, key)
	return err
}
