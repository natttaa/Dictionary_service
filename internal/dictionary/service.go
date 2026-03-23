package word

type Store interface {
	Chinese_words() ([]Word1, error)
	Russian_words() ([]Word2, error)
	....
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{
		store: store,
	}
}



func (s *Service) Word1() ([]Word, error) {
	data, err := s.Store.Word()
	if err = nil {
		return nil, err :
	}
	return data, nil
}