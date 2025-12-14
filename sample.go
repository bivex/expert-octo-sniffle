package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"time"
)

// Sample Go code with various constructs for AST analysis

type UserService struct {
	repo   UserRepository
	logger Logger
}

type UserRepository interface {
	FindByID(id int) (*User, error)
	Save(user *User) error
	Delete(id int) error
}

type Logger interface {
	Log(message string)
	Error(err error)
}

type User struct {
	ID   int
	Name string
	Email string
}

// Complex function demonstrating high complexity
func ProcessUsersWithComplexLogic(ctx context.Context, users []*User, workers int) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)

	for _, user := range users {
		user := user // capture loop variable
		g.Go(func() error {
			// Complex nested logic
			if user.ID > 0 {
				if user.Name != "" {
					if user.Email != "" {
						if err := validateUser(user); err != nil {
							return fmt.Errorf("validation failed: %w", err)
						}

						// More nested conditions
						switch user.ID % 3 {
						case 0:
							if len(user.Name) > 10 {
								return processSpecialUser(ctx, user)
							}
						case 1:
							for i := 0; i < 5; i++ {
								if i == 3 {
									break
								}
								fmt.Printf("Processing %s iteration %d\n", user.Name, i)
							}
						case 2:
							select {
							case <-ctx.Done():
								return ctx.Err()
							case <-time.After(time.Second):
								return updateUserStatus(user)
							}
						}
					} else {
						return fmt.Errorf("email is required")
					}
				} else if user.Email != "" {
					// Alternative path
					return sendNotification(user)
				}
			}

			return nil
		})
	}

	return g.Wait()
}

// High complexity function with multiple branches
func validateAndProcessUser(user *User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	if user.ID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	if user.Name == "" || len(user.Name) > 100 {
		if user.Name == "" {
			return fmt.Errorf("name is required")
		} else {
			return fmt.Errorf("name too long")
		}
	}

	if user.Email == "" {
		return fmt.Errorf("email is required")
	} else if !contains(user.Email, "@") {
		return fmt.Errorf("invalid email format")
	} else if len(user.Email) > 255 {
		return fmt.Errorf("email too long")
	}

	// Complex conditional with logical operators
	if user.ID > 1000 && (user.Name == "admin" || user.Name == "root") && user.Email != "test@example.com" {
		return fmt.Errorf("special users not allowed")
	}

	return nil
}

func validateUser(user *User) error {
	if user.ID <= 0 {
		return fmt.Errorf("invalid ID")
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func processSpecialUser(ctx context.Context, user *User) error {
	// Implementation
	return nil
}

func updateUserStatus(user *User) error {
	// Implementation
	return nil
}

func sendNotification(user *User) error {
	// Implementation
	return nil
}

