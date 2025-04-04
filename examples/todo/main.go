package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/vibe-go/vibe"
	"github.com/vibe-go/vibe/httpx"
	"github.com/vibe-go/vibe/middleware/cors"
)

// Todo represents a todo item.
type Todo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// TodoStore is a simple in-memory store for todos.
type TodoStore struct {
	sync.RWMutex
	todos  map[int]Todo
	nextID int
}

// NewTodoStore creates a new todo store with some sample data.
func NewTodoStore() *TodoStore {
	store := &TodoStore{
		todos:  make(map[int]Todo),
		nextID: 1,
	}

	// Add some sample todos
	store.Create(Todo{Title: "Learn Go", Completed: true})
	store.Create(Todo{Title: "Build a web framework", Completed: true})
	store.Create(Todo{Title: "Share it with the world", Completed: false})

	return store
}

// GetAll returns all todos.
func (s *TodoStore) GetAll() []Todo {
	s.RLock()
	defer s.RUnlock()

	todos := make([]Todo, 0, len(s.todos))
	for _, todo := range s.todos {
		todos = append(todos, todo)
	}
	return todos
}

// Get returns a todo by ID.
func (s *TodoStore) Get(id int) (Todo, bool) {
	s.RLock()
	defer s.RUnlock()

	todo, ok := s.todos[id]
	return todo, ok
}

// Create adds a new todo.
func (s *TodoStore) Create(todo Todo) Todo {
	s.Lock()
	defer s.Unlock()

	todo.ID = s.nextID
	s.nextID++
	s.todos[todo.ID] = todo
	return todo
}

// Update updates an existing todo.
func (s *TodoStore) Update(id int, todo Todo) (Todo, bool) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.todos[id]; !ok {
		return Todo{}, false
	}

	todo.ID = id
	s.todos[id] = todo
	return todo, true
}

// Delete removes a todo.
func (s *TodoStore) Delete(id int) bool {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.todos[id]; !ok {
		return false
	}

	delete(s.todos, id)
	return true
}

func main() {
	// Create a new router
	router := vibe.New()

	// Set up middleware
	logger := log.New(os.Stdout, "[todo-api] ", log.LstdFlags)
	router.Use(cors.New())

	// Create a todo store
	store := NewTodoStore()

	// Create a group for todo routes
	todoGroup := router.Group("/todos")

	// Define routes using the group
	todoGroup.Get("", func(w http.ResponseWriter, _ *http.Request) error {
		todos := store.GetAll()
		return httpx.JSON(w, todos, http.StatusOK)
	})

	todoGroup.Get("/{id}", func(w http.ResponseWriter, r *http.Request) error {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		todo, ok := store.Get(id)
		if !ok {
			return httpx.Error(w, errors.New("Todo not found"), http.StatusNotFound)
		}

		return httpx.JSON(w, todo, http.StatusOK)
	})

	todoGroup.Post("", func(w http.ResponseWriter, r *http.Request) error {
		var todo Todo
		if err := httpx.DecodeJSON(r, &todo); err != nil {
			return err
		}

		created := store.Create(todo)
		return httpx.JSON(w, created, http.StatusCreated)
	})

	todoGroup.Put("/{id}", func(w http.ResponseWriter, r *http.Request) error {
		// Extract ID from path
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		var todo Todo
		if decodeErr := httpx.DecodeJSON(r, &todo); decodeErr != nil {
			return decodeErr
		}

		updated, ok := store.Update(id, todo)
		if !ok {
			return httpx.Error(w, errors.New("Todo not found"), http.StatusNotFound)
		}

		return httpx.JSON(w, updated, http.StatusOK)
	})

	todoGroup.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) error {
		// Extract ID from path
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		ok := store.Delete(id)
		if !ok {
			return httpx.Error(w, errors.New("Todo not found"), http.StatusNotFound)
		}

		httpx.WithStatusCode(w, http.StatusNoContent)
		return nil
	})

	const (
		readTimeoutSeconds  = 15
		writeTimeoutSeconds = 15
		idleTimeoutSeconds  = 60
	)

	// Start the server
	port := "8080"
	logger.Printf("Server starting on port %s...", port)
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  time.Duration(readTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(writeTimeoutSeconds) * time.Second,
		IdleTimeout:  time.Duration(idleTimeoutSeconds) * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}
