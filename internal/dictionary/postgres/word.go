package postgres

import "database/sql"

type WordData struct {
	word sql.NullString
	dddd sql.NullString
	///
}

type Store struct {
	db *sql.DB
}

func NewWord(db *sql.DB) *Store {
	return &Store{
		db: db,
	}
}

func (s *Store) Chinese_words() ([]words.Chinese_words, error) {
	return nil, nil
}

func (s *Store) Book(word string) (*words.Words, error) {
	return &words.Words(), nil
}
