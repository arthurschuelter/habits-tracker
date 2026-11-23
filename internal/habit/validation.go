package habit

import (
	"strings"

	"github.com/google/uuid"
)

// ValidationError reports input that breaks a habit business rule.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + " " + e.Message
}

func Validate(habit Habit) (Habit, error) {
	name, err := NormalizeName(habit.Name)
	if err != nil {
		return Habit{}, err
	}
	habit.Name = name
	if habit.UserID == "" {
		return Habit{}, &ValidationError{Field: "user_id", Message: "is required"}
	}

	return habit, nil
}

// ValidateUpdate normalizes a habit ID and a partial update without mutating
// the caller's values.
func ValidateUpdate(habitID string, u Update) (string, Update, error) {
	id, err := uuid.Parse(strings.TrimSpace(habitID))
	if err != nil {
		return "", Update{}, &ValidationError{Field: "id", Message: "must be a valid UUID"}
	}
	if u.Name == nil && u.Archived == nil {
		return "", Update{}, &ValidationError{Field: "request", Message: "must include name or archived"}
	}

	out := Update{Archived: u.Archived}
	if u.Name != nil {
		name, err := NormalizeName(*u.Name)
		if err != nil {
			return "", Update{}, err
		}
		out.Name = &name
	}

	return id.String(), out, nil
}

// NormalizeName trims a habit name and rejects blank names.
func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", &ValidationError{Field: "name", Message: "is required"}
	}
	return name, nil
}
