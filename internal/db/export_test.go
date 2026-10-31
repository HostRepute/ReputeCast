package db

import "time"

// SetMigrateLockPoll shortens how often a waiting migrator polls, for tests.
func SetMigrateLockPoll(d time.Duration) func() {
	old := migrateLockPoll
	migrateLockPoll = d
	return func() { migrateLockPoll = old }
}
