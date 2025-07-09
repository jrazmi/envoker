package userrepo

type User struct {
	UserID string `db:"user_id"`
	Email  string `db:"email"`
}
