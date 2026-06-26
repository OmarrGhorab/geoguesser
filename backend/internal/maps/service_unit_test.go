package maps

import "testing"

func TestListFilterValidation(t *testing.T) {
	for _, tier := range []string{"", "free", "premium", "admin"} {
		if !validAccessTier(tier) {
			t.Fatalf("expected access tier %q to be valid", tier)
		}
	}
	for _, tier := range []string{"private", "paid", "free;drop table maps"} {
		if validAccessTier(tier) {
			t.Fatalf("expected access tier %q to be invalid", tier)
		}
	}

	for _, difficulty := range []string{"", "mixed", "easy", "medium", "hard"} {
		if !validDifficulty(difficulty) {
			t.Fatalf("expected difficulty %q to be valid", difficulty)
		}
	}
	for _, difficulty := range []string{"expert", "random", "hard desc"} {
		if validDifficulty(difficulty) {
			t.Fatalf("expected difficulty %q to be invalid", difficulty)
		}
	}
}
