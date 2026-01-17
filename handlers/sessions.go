package handlers

import (
	"database/sql"
	"net/http"
	"fmt"
	"time"
	_ "github.com/mattn/go-sqlite3"
)

func GetUserFromSession(r *http.Request, db *sql.DB) (int, error) {
    fmt.Println("=== GetUserFromSession called ===")
    
    cookie, err := r.Cookie("session_token")
    if err != nil {
        fmt.Println("ERROR: No cookie found:", err)
        return 0, err
    }
    
    fmt.Println("Cookie found:", cookie.Value)
    sessionID := cookie.Value
    
    var userID int
    var expiresAt string  // ← تأكد أنه string مش time.Time
    
    query := `SELECT user_id, expires_at FROM sessions WHERE id = ?`
    err = db.QueryRow(query, sessionID).Scan(&userID, &expiresAt)
    if err != nil {
        fmt.Println("ERROR: Session not found in DB:", err)
        return 0, err
    }
    
    fmt.Println("Session found! UserID:", userID, "Expires:", expiresAt)
    
    // Parse the time
    expTime, err := time.Parse("2006-01-02 15:04:05", expiresAt)
    if err != nil {
        fmt.Println("ERROR: Time parse failed:", err)
        return 0, err
    }
    
    if time.Now().After(expTime) {
        fmt.Println("Session expired!")
        db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
        return 0, fmt.Errorf("session expired")
    }
    
    fmt.Println("Session valid! Returning userID:", userID)
    return userID, nil
}

func RequireAuth(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        _, err := GetUserFromSession(r, db)
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }
        next(w, r)
    }
}
