package data

import (
	"errors"
	"sync"
)

// singletwon pattern - only one instance of MoviesStore across the application

type MoviesStore struct {
	mu     sync.RWMutex
	movies map[int64]Movie
}

var singletown *MoviesStore
var once sync.Once

var ErrMovieNotFound = errors.New("movie not found")

// will enter the once.Do only on the first call of the function across the
// application
func GetMoviesStore() *MoviesStore {
	once.Do(func() {
		singletown = &MoviesStore{
			movies: make(map[int64]Movie),
		}
	})
	return singletown
}

// update the movie id, adds to the map, return the updated movie
// Blocks if a write/read lock is held by another goroutine
func (ms *MoviesStore) AddMovie(movie Movie) (Movie)   {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	movie.ID = ms.nextID()
	ms.movies[movie.ID] = movie

	return movie
}

// Allows multiple concurrent reads.
// Blocks if a write lock is held by another goroutine.
func (ms *MoviesStore) GetMovie(id int64) (Movie, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	movie, ok := ms.movies[id]
	if !ok {
		return Movie{}, ErrMovieNotFound
	}

	return movie, nil
}

// ------------------ helpers

func (ms *MoviesStore) nextID() int64 {
    return int64(len(ms.movies)) + 1
}
