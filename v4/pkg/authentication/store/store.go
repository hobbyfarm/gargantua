package store

type UserStore interface {
	GetUser(username string) User
}
