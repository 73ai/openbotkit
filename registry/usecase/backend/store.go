package main

import (
	"github.com/73ai/openbotkit/store"
)

type Store struct {
	db *store.DB
}

func NewStore(db *store.DB) *Store {
	return &Store{db: db}
}
