package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Tournament struct {
	Id         int64
	Name       string
	Game       string
	System     string
	Rule       string
	Status     string
	CreatedBy  *int64
	WinnerId   *int64
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

type TournamentPlayer struct {
	UserId   int64
	Name     string
	JoinedAt time.Time
}

type Match struct {
	Id           int64
	TournamentId int64
	Round        int
	Position     int
	P1, P2       *int64
	Status       string
	WinnerId     *int64
	Source       string
	Score        string
	RoomId       string
	ReplayId     *int64
	Attempts     int
	ReadyAt      *time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

type ReplayInfo struct {
	Id        int64
	MatchId   *int64
	Game      string
	Frames    int64
	Size      int
	File      string
	CreatedAt time.Time
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return unix(*t)
}

// Tournaments

const tournamentCols = `id, name, game, system, rule, status, created_by, winner_id, created_at, started_at, finished_at`

func scanTournament(r scanner) (*Tournament, error) {
	var t Tournament
	var by, winner, started, finished sql.NullInt64
	var created int64
	err := r.Scan(&t.Id, &t.Name, &t.Game, &t.System, &t.Rule, &t.Status, &by, &winner, &created, &started, &finished)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t.CreatedBy, t.WinnerId = fromNullInt(by), fromNullInt(winner)
	t.CreatedAt, t.StartedAt, t.FinishedAt = fromUnix(created), fromNullUnix(started), fromNullUnix(finished)
	return &t, nil
}

func (s *Store) CreateTournament(ctx context.Context, t *Tournament) error {
	t.CreatedAt = time.Now().UTC().Truncate(time.Second)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tournaments(name, game, system, rule, status, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.Name, t.Game, t.System, t.Rule, t.Status, t.CreatedBy, unix(t.CreatedAt))
	if err != nil {
		return err
	}
	t.Id, err = res.LastInsertId()
	return err
}

func (s *Store) Tournament(ctx context.Context, id int64) (*Tournament, error) {
	return scanTournament(s.db.QueryRowContext(ctx, `SELECT `+tournamentCols+` FROM tournaments WHERE id = ?`, id))
}

func (s *Store) Tournaments(ctx context.Context) ([]Tournament, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+tournamentCols+` FROM tournaments ORDER BY created_at DESC, id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Tournament
	for rows.Next() {
		t, err := scanTournament(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Store) SetTournamentStatus(ctx context.Context, id int64, status string, winner *int64) error {
	now := unix(time.Now())
	switch status {
	case "running":
		return s.exec1(ctx, `UPDATE tournaments SET status = ?, started_at = ? WHERE id = ?`, status, now, id)
	case "finished", "canceled":
		return s.exec1(ctx, `UPDATE tournaments SET status = ?, winner_id = ?, finished_at = ? WHERE id = ?`, status, winner, now, id)
	}
	return s.exec1(ctx, `UPDATE tournaments SET status = ? WHERE id = ?`, status, id)
}

// Players

func (s *Store) AddTournamentPlayer(ctx context.Context, tid, uid int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tournament_players(tournament_id, user_id, joined_at) VALUES (?, ?, ?)`, tid, uid, unix(time.Now()))
	if err != nil && strings.Contains(err.Error(), "constraint failed") {
		return ErrDuplicate
	}
	return err
}

func (s *Store) RemoveTournamentPlayer(ctx context.Context, tid, uid int64) error {
	return s.exec1(ctx, `DELETE FROM tournament_players WHERE tournament_id = ? AND user_id = ?`, tid, uid)
}

func (s *Store) TournamentPlayers(ctx context.Context, tid int64) ([]TournamentPlayer, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT p.user_id, u.name, p.joined_at FROM tournament_players p JOIN users u ON u.id = p.user_id
		  WHERE p.tournament_id = ? ORDER BY p.joined_at, p.user_id`, tid)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []TournamentPlayer
	for rows.Next() {
		var p TournamentPlayer
		var joined int64
		if err := rows.Scan(&p.UserId, &p.Name, &joined); err != nil {
			return nil, err
		}
		p.JoinedAt = fromUnix(joined)
		out = append(out, p)
	}
	return out, rows.Err()
}

// Matches

const matchCols = `id, tournament_id, round, position, p1_id, p2_id, status, winner_id, source, score, room_id,
	replay_id, attempts, ready_at, started_at, finished_at`

func scanMatch(r scanner) (*Match, error) {
	var m Match
	var p1, p2, winner, replay, ready, started, finished sql.NullInt64
	err := r.Scan(&m.Id, &m.TournamentId, &m.Round, &m.Position, &p1, &p2, &m.Status, &winner, &m.Source, &m.Score,
		&m.RoomId, &replay, &m.Attempts, &ready, &started, &finished)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	m.P1, m.P2, m.WinnerId, m.ReplayId = fromNullInt(p1), fromNullInt(p2), fromNullInt(winner), fromNullInt(replay)
	m.ReadyAt, m.StartedAt, m.FinishedAt = fromNullUnix(ready), fromNullUnix(started), fromNullUnix(finished)
	return &m, nil
}

// CreateBracket stores all matches of a tournament and marks it running, atomically.
func (s *Store) CreateBracket(ctx context.Context, tid int64, matches []*Match) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, m := range matches {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO matches(tournament_id, round, position, p1_id, p2_id, status, winner_id, source, ready_at, finished_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			tid, m.Round, m.Position, m.P1, m.P2, m.Status, m.WinnerId, m.Source, nullTime(m.ReadyAt), nullTime(m.FinishedAt))
		if err != nil {
			return err
		}
		if m.Id, err = res.LastInsertId(); err != nil {
			return err
		}
		m.TournamentId = tid
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tournaments SET status = 'running', started_at = ? WHERE id = ?`,
		unix(time.Now()), tid); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Matches(ctx context.Context, tid int64) ([]Match, error) {
	return s.queryMatches(ctx, `SELECT `+matchCols+` FROM matches WHERE tournament_id = ? ORDER BY round, position`, tid)
}

// OpenMatches returns matches that still need to be played (any tournament), oldest first.
func (s *Store) OpenMatches(ctx context.Context) ([]Match, error) {
	return s.queryMatches(ctx, `SELECT `+matchCols+` FROM matches WHERE status IN ('ready', 'starting', 'running')
		ORDER BY tournament_id, round, position`)
}

func (s *Store) queryMatches(ctx context.Context, q string, args ...any) ([]Match, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (s *Store) Match(ctx context.Context, id int64) (*Match, error) {
	return scanMatch(s.db.QueryRowContext(ctx, `SELECT `+matchCols+` FROM matches WHERE id = ?`, id))
}

func (s *Store) MatchAt(ctx context.Context, tid int64, round, position int) (*Match, error) {
	return scanMatch(s.db.QueryRowContext(ctx,
		`SELECT `+matchCols+` FROM matches WHERE tournament_id = ? AND round = ? AND position = ?`, tid, round, position))
}

// UpdateMatch writes the mutable fields of a match.
func (s *Store) UpdateMatch(ctx context.Context, m *Match) error {
	return s.exec1(ctx,
		`UPDATE matches SET p1_id = ?, p2_id = ?, status = ?, winner_id = ?, source = ?, score = ?, room_id = ?,
		   replay_id = ?, attempts = ?, ready_at = ?, started_at = ?, finished_at = ? WHERE id = ?`,
		m.P1, m.P2, m.Status, m.WinnerId, m.Source, m.Score, m.RoomId, m.ReplayId, m.Attempts,
		nullTime(m.ReadyAt), nullTime(m.StartedAt), nullTime(m.FinishedAt), m.Id)
}

// Replays

func (s *Store) CreateReplay(ctx context.Context, r *ReplayInfo) error {
	r.CreatedAt = time.Now().UTC().Truncate(time.Second)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO replays(match_id, game, frames, size, file, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		r.MatchId, r.Game, r.Frames, r.Size, r.File, unix(r.CreatedAt))
	if err != nil {
		return err
	}
	r.Id, err = res.LastInsertId()
	return err
}

func (s *Store) Replay(ctx context.Context, id int64) (*ReplayInfo, error) {
	var r ReplayInfo
	var match sql.NullInt64
	var created int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id, match_id, game, frames, size, file, created_at FROM replays WHERE id = ?`, id).
		Scan(&r.Id, &match, &r.Game, &r.Frames, &r.Size, &r.File, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	r.MatchId, r.CreatedAt = fromNullInt(match), fromUnix(created)
	return &r, nil
}
