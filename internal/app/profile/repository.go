package profile

type Repository interface {
	Create(p *Profile) error
	GetByName(name string) (*Profile, error)
	GetByAlias(alias string) (*Profile, error)
	GetByID(id string) (*Profile, error)
	List() ([]Profile, error)
}

