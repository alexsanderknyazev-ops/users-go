package users

import "testing"

func TestUsers_TableName(t *testing.T) {
	var u Users
	if u.TableName() != "users" {
		t.Fatalf("TableName() = %q", u.TableName())
	}
}
