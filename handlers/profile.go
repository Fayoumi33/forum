package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type ProfileData struct {
	Username      string
	Email         string
	PostsCount    int
	CommentsCount int
	Posts         []Post
	CurrentTab    string
	CurrentUserID int
}

func ProfilePage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID, err := GetUserFromSession(r, db)
	if err != nil {
		fmt.Println("ERROR: GetUserFromSession failed:", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	fmt.Println("UserID from session:", userID)

	var username, email string
	err = db.QueryRow(`SELECT username, email FROM users WHERE id = ?`, userID).Scan(&username, &email)
	if err != nil {
		fmt.Println("ERROR: Query failed:", err)
		RenderError(w, http.StatusNotFound)
		return
	}

	fmt.Println("Username:", username, "Email:", email)

	var postsCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM posts WHERE user_id = ?`, userID).Scan(&postsCount)
	if err != nil {
		postsCount = 0
	}
	var commentsCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM comments WHERE user_id = ?`, userID).Scan(&commentsCount)
	if err != nil {
		commentsCount = 0
	}

	currentTab := r.URL.Query().Get("tab")
	if currentTab == "" {
		currentTab = "created"
	}

	var posts []Post
	var query string
	if currentTab == "created" {
		query = `SELECT p.id, p.title, p.content, p.created_at, p.user_id, u.username,
        COUNT(CASE WHEN post_likes.type = 'like' THEN 1 END) as likes_count,
        COUNT(CASE WHEN post_likes.type = 'dislike' THEN 1 END) as dislikes_count,
        COUNT(DISTINCT c.id) as comments_count
    FROM posts p
    JOIN users u ON p.user_id = u.id
    LEFT JOIN post_likes ON p.id = post_likes.post_id
    LEFT JOIN comments c ON p.id = c.post_id
    WHERE p.user_id = ?
    GROUP BY p.id
    ORDER BY p.created_at DESC`
	} else if currentTab == "liked" {
		query = `SELECT p.id, p.title, p.content, p.created_at, p.user_id, u.username,
        COUNT(CASE WHEN post_likes.type = 'like' THEN 1 END) as likes_count,
        COUNT(CASE WHEN post_likes.type = 'dislike' THEN 1 END) as dislikes_count,
        COUNT(DISTINCT c.id) as comments_count
    FROM posts p
    JOIN users u ON p.user_id = u.id
    LEFT JOIN post_likes ON p.id = post_likes.post_id
    LEFT JOIN comments c ON p.id = c.post_id
    WHERE p.id IN (SELECT post_id FROM post_likes WHERE user_id = ? AND type = 'like')
    GROUP BY p.id
    ORDER BY p.created_at DESC`
	}

	rows, err := db.Query(query, userID)
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p Post
		err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.CreatedAt, &p.UserID,
			&p.Username, &p.LikesCount, &p.DislikesCount, &p.CommentsCount)
		if err != nil {
			continue
		}
		posts = append(posts, p)
	}

	data := ProfileData{
		Username:      username,
		Email:         email,
		PostsCount:    postsCount,
		CommentsCount: commentsCount,
		Posts:         posts,
		CurrentTab:    currentTab,
		CurrentUserID: userID,
	}
	tmpl, err := template.ParseFiles("templates/profile.html")
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func EditProfile(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID, err := GetUserFromSession(r, db)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		var username, email string
		err := db.QueryRow(`SELECT username, email FROM users WHERE id = ?`, userID).Scan(&username, &email)
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}

		data := struct {
			Username string
			Email    string
		}{username, email}

		tmpl, err := template.ParseFiles("templates/edit_profile.html")
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, data)
		return
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		email := r.FormValue("email")
		currentPassword := r.FormValue("current_password")
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")

		if username == "" || email == "" || currentPassword == "" {
			http.Error(w, "missing required fields", http.StatusBadRequest)
			return
		}

		if newPassword != "" && newPassword != confirmPassword {
			http.Error(w, "passwords do not match", http.StatusBadRequest)
			return
		}

		var storedPassword string
		err := db.QueryRow(`SELECT password FROM users WHERE id = ?`, userID).Scan(&storedPassword)
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(currentPassword))
		if err != nil {
			RenderError(w, http.StatusUnauthorized)
			return
		}

		var existingUserID int
		err = db.QueryRow(`SELECT id FROM users WHERE username = ? AND id != ?`, username, userID).Scan(&existingUserID)
		if err == nil {
			http.Error(w, "username already exists", http.StatusBadRequest)
			return
		}

		err = db.QueryRow(`SELECT id FROM users WHERE email = ? AND id != ?`, email, userID).Scan(&existingUserID)
		if err == nil {
			http.Error(w, "email already exists", http.StatusBadRequest)
			return
		}

		if newPassword != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				RenderError(w, http.StatusInternalServerError)
				return
			}

			_, err = db.Exec(`UPDATE users SET username = ?, email = ?, password = ? WHERE id = ?`,
				username, email, string(hashedPassword), userID)
			if err != nil {
				RenderError(w, http.StatusInternalServerError)
				return
			}
		} else {
			_, err = db.Exec(`UPDATE users SET username = ?, email = ? WHERE id = ?`,
				username, email, userID)
			if err != nil {
				RenderError(w, http.StatusInternalServerError)
				return
			}
		}

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}
