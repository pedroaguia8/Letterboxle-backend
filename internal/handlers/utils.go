package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sqlc-dev/pqtype"
)

func nullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullInt32(ni sql.NullInt32) int32 {
	if ni.Valid {
		return ni.Int32
	}
	return 0
}

// genreNames extracts the names out of a genres JSONB column storing TMDB's
// [{"id":..,"name":..}, ...] genre objects, keeping the ids in the DB for
// future filtering while the API only needs the names.
func genreNames(genres pqtype.NullRawMessage) []string {
	if !genres.Valid {
		return nil
	}
	var parsed []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(genres.RawMessage, &parsed); err != nil {
		return nil
	}
	names := make([]string, len(parsed))
	for i, g := range parsed {
		names[i] = g.Name
	}
	return names
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("couldn't marshal data to json: %w", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		return fmt.Errorf("error writing data to response: %w", err)
	}
	return nil
}

func RespondWithError(w http.ResponseWriter, code int, msg string) error {
	return RespondWithJSON(w, code, map[string]string{"error": msg})
}
