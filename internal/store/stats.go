package store

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/mvanhorn/agent-tincan/internal/envelope"
)

// AgentStat is one agent's record of asks that finished in a window.
// MedianReply is over answered asks only, and is zero when none has a
// known reply time.
type AgentStat struct {
	Answered    int
	Failed      int
	Expired     int
	MedianReply time.Duration
}

// AgentStats returns the record of every agent with an ask that finished at
// or after since: counts and durations, never text. Pings and notifies are
// not work asked of a teammate. A decline, a cancel and an ask still open
// are not outcomes that say whether it answers, a hold the owner never
// approved never reached it, and an ask that expired on an unanswered
// clarification was waiting on its asker: only Answer removes the question's
// reply row. A held ask's reply time starts at its approval,
// so the owner's delay is not charged to the teammate; with no approval
// event to read, it counts but has no reply time.
func (s *Store) AgentStats(ctx context.Context, since time.Time) (map[string]AgentStat, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT r.to_agent, r.status,
		CASE WHEN r.was_held = 1 THEN (SELECT MAX(a.at) FROM audit a WHERE a.request_id = r.id AND a.event = 'approved') ELSE r.created_at END,
		q.created_at
		FROM requests r LEFT JOIN replies q ON q.request_id = r.id
		WHERE r.status IN ('answered', 'failed', 'expired') AND r.kind NOT IN ('ping', 'notify')
		AND NOT (r.was_held = 1 AND r.approved = 0)
		AND NOT (r.status = 'expired' AND COALESCE(q.status, '') = 'needs_input')
		AND COALESCE(q.created_at, r.updated_at) >= ?`, since.UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]AgentStat)
	replies := make(map[string][]int64)
	for rows.Next() {
		var agent, status string
		var started, replied sql.NullInt64
		if err := rows.Scan(&agent, &status, &started, &replied); err != nil {
			return nil, err
		}
		stat := out[agent]
		switch envelope.Status(status) {
		case envelope.StatusAnswered:
			stat.Answered++
			if started.Valid && replied.Valid {
				replies[agent] = append(replies[agent], max(replied.Int64-started.Int64, 0))
			}
		case envelope.StatusFailed:
			stat.Failed++
		case envelope.StatusExpired:
			stat.Expired++
		}
		out[agent] = stat
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for agent, ms := range replies {
		stat := out[agent]
		stat.MedianReply = time.Duration(medianMillis(ms)) * time.Millisecond
		out[agent] = stat
	}
	return out, nil
}

// medianMillis is the median of ms, sorting it in place. An even count takes
// the midpoint of the middle two.
func medianMillis(ms []int64) int64 {
	slices.Sort(ms)
	n := len(ms)
	if n%2 == 1 {
		return ms[n/2]
	}
	return ms[n/2-1] + (ms[n/2]-ms[n/2-1])/2
}
