package user

type UserRepository interface {
	Create(u *User) error
	GetByUsername(username string) (*User, error)
	GetByID(id string) (*User, error)
}
