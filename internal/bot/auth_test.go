package bot

import "testing"

func TestAuthorizer_IsAllowed_permitted(t *testing.T) {
	auth := NewAuthorizer([]int64{100, 200, 300})

	for _, id := range []int64{100, 200, 300} {
		if !auth.IsAllowed(id) {
			t.Errorf("expected user %d to be allowed", id)
		}
	}
}

func TestAuthorizer_IsAllowed_denied(t *testing.T) {
	auth := NewAuthorizer([]int64{100, 200})

	for _, id := range []int64{0, 999, -1} {
		if auth.IsAllowed(id) {
			t.Errorf("expected user %d to be denied", id)
		}
	}
}

func TestAuthorizer_EmptyList_deniesEveryone(t *testing.T) {
	auth := NewAuthorizer(nil)

	for _, id := range []int64{1, 42, 100} {
		if auth.IsAllowed(id) {
			t.Errorf("expected user %d to be denied by empty authorizer", id)
		}
	}
}
