package bot

// Authorizer holds the set of Telegram user IDs that are permitted to interact
// with the bot. All other users are rejected with "Access denied."
type Authorizer struct {
	allowed map[int64]struct{}
}

// NewAuthorizer returns an Authorizer that permits only the given user IDs.
// An empty slice causes every user to be denied.
func NewAuthorizer(userIDs []int64) *Authorizer {
	m := make(map[int64]struct{}, len(userIDs))
	for _, id := range userIDs {
		m[id] = struct{}{}
	}
	return &Authorizer{allowed: m}
}

// IsAllowed reports whether userID is in the whitelist.
func (a *Authorizer) IsAllowed(userID int64) bool {
	_, ok := a.allowed[userID]
	return ok
}
