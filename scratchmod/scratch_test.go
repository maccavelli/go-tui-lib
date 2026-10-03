package scratchmod

import "testing"

func TestBullet(t *testing.T) {
	if Bullet() == "" {
		t.Fatal("no bullet")
	}
}
