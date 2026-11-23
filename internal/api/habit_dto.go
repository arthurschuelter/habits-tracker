package api

import "time"

type habitResponseDTO struct {
	ID            string                  `json:"id" format:"uuid" example:"11111111-1111-1111-1111-111111111111"`
	Name          string                  `json:"name" example:"Read"`
	Description   string                  `json:"description" example:"Read for 20 minutes"`
	CreatedAt     *time.Time              `json:"created_at" format:"date-time" example:"2026-11-17T12:34:56Z" extensions:"x-nullable"`
	Entries       []habitEntryResponseDTO `json:"entries"`
	HighestStreak int                     `json:"highest_streak" minimum:"0" example:"7"`
	Streak        int                     `json:"streak" minimum:"0" example:"2"`
	Archived      bool                    `json:"archived" example:"false"`
}

type habitEntryResponseDTO struct {
	ID      string `json:"id" format:"uuid" example:"22222222-2222-2222-2222-222222222222"`
	HabitID string `json:"habit_id" format:"uuid" example:"11111111-1111-1111-1111-111111111111"`
	Date    string `json:"date" format:"date" example:"2026-11-16"`
	// Recorded minutes. Legacy NULL durations are returned as zero.
	Duration int `json:"duration" minimum:"0" maximum:"32767" example:"20"`
}

type CreateHabitRequestDTO struct {
	Name        string `json:"name" binding:"required" minLength:"1" example:"Read"`
	Description string `json:"description" example:"Read for 20 minutes"`
}

// UpdateHabitRequestDTO is a partial update; omitted fields are unchanged.
// At least one field is required.
type UpdateHabitRequestDTO struct {
	// New name; trimmed and must be nonblank when present.
	Name *string `json:"name" minLength:"1" example:"Read before bed" extensions:"x-nullable"`
	// true archives the habit; false restores it.
	Archived *bool `json:"archived" example:"true" extensions:"x-nullable"`
}

type CreateEntryRequestRequestDTO struct {
	HabitID string `json:"habit_id" binding:"required" format:"uuid" example:"11111111-1111-1111-1111-111111111111"`
	// Optional YYYY-MM-DD date. Omitted or blank values use the server's current date.
	Date     string `json:"date" example:"2026-11-16"`
	Duration int    `json:"duration" binding:"required" minimum:"1" maximum:"32767" example:"20"`
}

// errorResponseDTO documents Echo's default JSON error envelope.
type errorResponseDTO struct {
	Message string `json:"message" example:"invalid request body"`
}
