package globaldb

import (
	"database/sql"
	"fmt"
	"time"
)

// DefaultLockTimeout is the duration after which a session lock expires.
// Configurable for testing.
var DefaultLockTimeout = 30 * time.Minute

// LockInfo contains details about a session lock on a feature.
type LockInfo struct {
	FeatureID string `json:"feature_id"`
	ProjectID string `json:"project_id"`
	SessionID string `json:"session_id"`
	LockedAt  string `json:"locked_at"`
}

// AcquireLock atomically acquires a lock on a feature for a session.
// If the same session already holds the lock, the timestamp is refreshed.
// Returns an error if a different session holds the lock.
func AcquireLock(projectID, featureID, sessionID string) error {
	db, err := DB()
	if err != nil {
		return fmt.Errorf("globaldb: %w", err)
	}

	// Clean expired locks first.
	cleanExpired(db)

	// Check for existing lock.
	var existingSession string
	err = db.QueryRow(
		"SELECT session_id FROM session_locks WHERE project_id = ? AND feature_id = ?",
		projectID, featureID,
	).Scan(&existingSession)

	if err == nil {
		// Lock exists.
		if existingSession == sessionID {
			// Same session — refresh timestamp.
			_, err = db.Exec(
				"UPDATE session_locks SET locked_at = datetime('now') WHERE project_id = ? AND feature_id = ?",
				projectID, featureID,
			)
			return err
		}
		return fmt.Errorf("feature **%s** is locked by session %s", featureID, existingSession[:min(8, len(existingSession))])
	}

	// No existing lock — acquire.
	_, err = db.Exec(
		"INSERT INTO session_locks (project_id, feature_id, session_id, locked_at) VALUES (?, ?, ?, datetime('now'))",
		projectID, featureID, sessionID,
	)
	return err
}

// CheckLock verifies the calling session owns the lock on a feature.
// Returns nil if: (a) the session owns the lock, or (b) no lock exists.
// Returns an error if a different session holds the lock.
func CheckLock(projectID, featureID, sessionID string) error {
	db, err := DB()
	if err != nil {
		return fmt.Errorf("globaldb: %w", err)
	}

	cleanExpired(db)

	var existingSession string
	err = db.QueryRow(
		"SELECT session_id FROM session_locks WHERE project_id = ? AND feature_id = ?",
		projectID, featureID,
	).Scan(&existingSession)

	if err != nil {
		return nil // No lock — allow.
	}

	if existingSession == sessionID {
		return nil // Same session — allow.
	}

	return fmt.Errorf("feature **%s** is locked by session %s — only that session can advance it",
		featureID, existingSession[:min(8, len(existingSession))])
}

// RefreshLock updates the locked_at timestamp for an active lock.
func RefreshLock(projectID, featureID, sessionID string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	_, err = db.Exec(
		"UPDATE session_locks SET locked_at = datetime('now') WHERE project_id = ? AND feature_id = ? AND session_id = ?",
		projectID, featureID, sessionID,
	)
	return err
}

// ReleaseLock removes a specific feature lock.
func ReleaseLock(projectID, featureID string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	_, err = db.Exec(
		"DELETE FROM session_locks WHERE project_id = ? AND feature_id = ?",
		projectID, featureID,
	)
	return err
}

// ReleaseSessionLocks removes ALL locks held by a session (disconnect cleanup).
func ReleaseSessionLocks(sessionID string) (int64, error) {
	db, err := DB()
	if err != nil {
		return 0, err
	}
	result, err := db.Exec("DELETE FROM session_locks WHERE session_id = ?", sessionID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CleanExpiredLocks removes locks older than the given duration.
func CleanExpiredLocks(maxAge time.Duration) (int64, error) {
	db, err := DB()
	if err != nil {
		return 0, err
	}
	minutes := int(maxAge.Minutes())
	result, err := db.Exec(
		fmt.Sprintf("DELETE FROM session_locks WHERE locked_at < datetime('now', '-%d minutes')", minutes),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetLockInfo returns the lock holder and timestamp for a feature, or nil if unlocked.
func GetLockInfo(projectID, featureID string) (*LockInfo, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}

	cleanExpired(db)

	var info LockInfo
	err = db.QueryRow(
		"SELECT feature_id, project_id, session_id, locked_at FROM session_locks WHERE project_id = ? AND feature_id = ?",
		projectID, featureID,
	).Scan(&info.FeatureID, &info.ProjectID, &info.SessionID, &info.LockedAt)

	if err != nil {
		return nil, nil // No lock.
	}
	return &info, nil
}

// cleanExpired removes locks older than DefaultLockTimeout.
func cleanExpired(db *sql.DB) {
	if db == nil {
		return
	}
	minutes := int(DefaultLockTimeout.Minutes())
	db.Exec(fmt.Sprintf("DELETE FROM session_locks WHERE locked_at < datetime('now', '-%d minutes')", minutes))
}
