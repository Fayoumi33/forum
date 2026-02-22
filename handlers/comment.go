package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
)

type Comment struct {
	ID            int
	Content       string
	CreatedAt     string
	UserID        int
	Username      string
	LikesCount    int
	DislikesCount int
}

func AddComment(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != "POST" {
		http.Error(w, " mehtod not allowed", http.StatusMethodNotAllowed)
		return
	}
	content := strings.TrimSpace(r.FormValue("content"))
	postIDSrt := r.FormValue("post_id")

	if content == "" || postIDSrt == "" {
		RenderError(w, http.StatusBadRequest)
		return
	}
	postID, err := strconv.Atoi(postIDSrt)
	if err != nil {
		RenderError(w, http.StatusBadRequest)
		return
	}

	userID, err := GetUserFromSession(r, db)
	if err != nil {
		RenderError(w, http.StatusUnauthorized)
		return
	}

	_, err = db.Exec(`INSERT INTO comments(content, post_id, user_id, created_at) VALUES (?,?,?,datetime('now'))`,
		content, postID, userID)

	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/home"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func LikeComment(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	commentIDStr := r.FormValue("comment_id")
	commentID, err := strconv.Atoi(commentIDStr)
	if err != nil {
		http.Error(w, "invalid comment ID", http.StatusBadRequest)
		return
	}

	userID, err := GetUserFromSession(r, db)
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}

	var existingType string
	query := `SELECT type FROM comment_likes WHERE comment_id = ? AND user_id = ?`
	err = db.QueryRow(query, commentID, userID).Scan(&existingType)

	if err == sql.ErrNoRows {
		db.Exec(`INSERT INTO comment_likes(comment_id, user_id, type) VALUES(?,?,'like')`, commentID, userID)
	} else if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	} else {
		if existingType == "like" {
			db.Exec(`DELETE FROM comment_likes WHERE comment_id = ? AND user_id = ?`, commentID, userID)
		} else {
			db.Exec(`UPDATE comment_likes SET type = 'like' WHERE comment_id = ? AND user_id = ?`, commentID, userID)
		}
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/home"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func DislikeComment(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	commentIDStr := r.FormValue("comment_id")
	commentID, err := strconv.Atoi(commentIDStr)
	if err != nil {
		http.Error(w, "invalid comment ID", http.StatusBadRequest)
		return
	}

	userID, err := GetUserFromSession(r, db)
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}

	var existingType string
	query := `SELECT type FROM comment_likes WHERE comment_id = ? AND user_id = ?`
	err = db.QueryRow(query, commentID, userID).Scan(&existingType)

	if err == sql.ErrNoRows {
		db.Exec(`INSERT INTO comment_likes(comment_id, user_id, type) VALUES(?,?,'dislike')`, commentID, userID)
	} else if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	} else {
		if existingType == "dislike" {
			db.Exec(`DELETE FROM comment_likes WHERE comment_id = ? AND user_id = ?`, commentID, userID)
		} else {
			db.Exec(`UPDATE comment_likes SET type = 'dislike' WHERE comment_id = ? AND user_id = ?`, commentID, userID)
		}
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/home"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}
