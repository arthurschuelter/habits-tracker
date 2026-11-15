package habit_entry

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MaxDurationMinutes: largest duration supported by the SMALLINT column.
const MaxDurationMinutes = 32767

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func Validate(entry HabitEntry) (HabitEntry, error) {
	id, err := uuid.Parse(strings.TrimSpace(entry.HabitID))
	if err != nil {
		return HabitEntry{}, &ValidationError{Field: "habit_id", Message: "must be a valid UUID"}
	}
	entry.HabitID = id.String()

	entry.Date = strings.TrimSpace(entry.Date)
	if entry.Date == "" {
		entry.Date = time.Now().Format(time.DateOnly)
	}

	date, err := time.Parse(time.DateOnly, entry.Date)
	if err != nil || date.Year() < 1 {
		return HabitEntry{}, &ValidationError{Field: "date", Message: "must be a valid date in YYYY-MM-DD format"}
	}
	if err := ValidateDuration(entry.Duration); err != nil {
		return HabitEntry{}, err
	}

	return entry, nil
}

func ValidateDuration(duration int) error {
	if duration < 1 || duration > MaxDurationMinutes {
		return &ValidationError{
			Field:   "duration",
			Message: fmt.Sprintf("must be a positive whole number of minutes between 1 and %d", MaxDurationMinutes),
		}
	}
	return nil
}
