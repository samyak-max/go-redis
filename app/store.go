package main

import "time"

type Store struct {
	data     map[string]string
	timeouts map[string]time.Time
}

func NewStore() *Store {
	return &Store{
		data:     make(map[string]string),
		timeouts: make(map[string]time.Time),
	}
}
