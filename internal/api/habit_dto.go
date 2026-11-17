package api

import "time"

type habitResponseDTO struct {
	ID          string `json:"id" format:"uuid" example:"11111111-1111-1111-1111-111111111111"`
	Name        string `json:"name" example:"Read"`
	Description string `json:"description" example:"Read for 20 minutes"`
	// Creation timestamp in RFC3339 format; null for legacy habits with an unknown creation time.
	CreatedAt *time.Time              `json:"created_at" format:"date-time" example:"2026-11-17T12:34:56Z" extensions:"x-nullable"`
	Entries   []habitEntryResponseDTO `json:"entries"`
	Streak    int                     `json:"streak" minimum:"0" example:"2"`
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
	UserID      string `json:"user_id" binding:"required" format:"uuid" example:"0a483f13-1ffc-4f85-aa71-98a9b96fedfe"`
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
