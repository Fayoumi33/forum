package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type Post struct {
	ID            int
	Title         string
	Content       string
	CreatedAt     string
	UserID        int
	Username      string
	CommentsCount int
	LikesCount    int
	DislikesCount int
	Comments      []Comment
	Categories    []string
}

type PostDetailsData struct {
	Post     Post
	Comments []Comment
}

func HomePage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	fmt.Println("HomePage called")
	filter := r.URL.Query().Get("filter")
	category := r.URL.Query().Get("category")

	fmt.Println("=== DEBUG ===")
	fmt.Println("Filter:", filter)
	fmt.Println("Category:", category)
	fmt.Println("Full URL:", r.URL.String())
	fmt.Println("=============")

	userID, err := GetUserFromSession(r, db)
	if (filter == "created" || filter == "liked") && err != nil {
		RenderError(w, http.StatusUnauthorized)
		return
	}

	query := `SELECT p.id, p.title, p.content, p.created_at, p.user_id, u.username
	FROM posts p
	JOIN users u ON p.user_id = u.id`

	var args []interface{}

	if filter == "created" {
		query += ` WHERE p.user_id = ?`
		args = append(args, userID)
	} else if filter == "liked" {
		query += ` WHERE p.id IN (SELECT post_id FROM post_likes WHERE user_id = ? AND type = 'like')`
		args = append(args, userID)
	} else if category != "" && category != "all" {
		query += ` WHERE p.id IN (SELECT post_id FROM post_categories pc JOIN categories c ON pc.category_id = c.id WHERE c.name = ?)`
		args = append(args, category)
	}

	query += ` ORDER BY p.created_at DESC`

	fmt.Println("Final Query:", query)
	fmt.Println("Args:", args)

	var rows *sql.Rows
	if len(args) > 0 {
		rows, err = db.Query(query, args...)
	} else {
		rows, err = db.Query(query)
	}
	if err != nil {
		fmt.Println("Query error:", err)
		RenderError(w, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	fmt.Println("Query successful, reading rows...")

	var posts []Post
	for rows.Next() {
		var p Post
		err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.CreatedAt, &p.UserID, &p.Username)
		if err != nil {
			fmt.Println("Scan error:", err)
			continue
		}

		fmt.Println("Processing post ID:", p.ID, "Title:", p.Title)

		var likesCount, dislikesCount int
		err = db.QueryRow(`SELECT
			COUNT(CASE WHEN type = 'like' THEN 1 END) as likes,
			COUNT(CASE WHEN type = 'dislike' THEN 1 END) as dislikes
			FROM post_likes WHERE post_id = ?`, p.ID).Scan(&likesCount, &dislikesCount)
		if err == nil {
			p.LikesCount = likesCount
			p.DislikesCount = dislikesCount
		}

		var commentsCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM comments WHERE post_id = ?`, p.ID).Scan(&commentsCount)
		if err == nil {
			p.CommentsCount = commentsCount
		}

		catRows, err := db.Query(`SELECT c.name FROM categories c JOIN post_categories pc ON c.id = pc.category_id WHERE pc.post_id = ?`, p.ID)
		if err == nil {
			for catRows.Next() {
				var catName string
				if catRows.Scan(&catName) == nil {
					p.Categories = append(p.Categories, catName)
				}
			}
			catRows.Close()
		}

		commentsQuery := `SELECT c.id, c.content, c.created_at, c.user_id, u.username
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC`

		commentRows, err := db.Query(commentsQuery, p.ID)
		if err == nil {
			for commentRows.Next() {
				var comment Comment
				err := commentRows.Scan(&comment.ID, &comment.Content, &comment.CreatedAt,
					&comment.UserID, &comment.Username)
				if err == nil {
					db.QueryRow(`SELECT
						COUNT(CASE WHEN type = 'like' THEN 1 END),
						COUNT(CASE WHEN type = 'dislike' THEN 1 END)
						FROM comment_likes WHERE comment_id = ?`, comment.ID).Scan(&comment.LikesCount, &comment.DislikesCount)

					p.Comments = append(p.Comments, comment)
				}
			}
			commentRows.Close()
		}

		posts = append(posts, p)
	}

	fmt.Println("Total posts found:", len(posts))

	tmpl, err := template.ParseFiles("templates/home.html")
	if err != nil {
		fmt.Println("Template error:", err)
		RenderError(w, http.StatusInternalServerError)
		return
	}

	fmt.Println("Executing template...")
	err = tmpl.Execute(w, posts)
	if err != nil {
		fmt.Println("Execute error:", err)
		RenderError(w, http.StatusInternalServerError)
		return
	}

	fmt.Println("HomePage completed successfully")
}

func CreatePost(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("templates/create_post.html")
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}
		title := strings.TrimSpace(r.FormValue("title"))
		content := r.FormValue("content")
		categories := r.Form["categories"]

		fmt.Println("Categories received:", categories)
		if title == "" || content == "" || len(categories) < 1 {
			RenderError(w, http.StatusBadRequest)
			return
		}
		userID, err := GetUserFromSession(r, db)
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}
		createQuery := `INSERT INTO posts (title, content, user_id, created_at) VALUES (?, ?, ?, datetime('now'))`
		result, err := db.Exec(createQuery, title, content, userID)
		if err != nil {
			fmt.Println("Create post error:", err)
			RenderError(w, http.StatusInternalServerError)
			return
		}
		postID, err := result.LastInsertId()
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}
		for _, categoryName := range categories {
			var categoryID int
			err := db.QueryRow(`SELECT id FROM categories WHERE name = ?`, categoryName).Scan(&categoryID)
			if err != nil {
				continue
			}
			db.Exec(`INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)`, postID, categoryID)
		}
		http.Redirect(w, r, "/home", http.StatusSeeOther)
	}
}

func PostDetails(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	postIDStr := r.URL.Query().Get("id")
	if postIDStr == "" {
		http.Error(w, "Missing post ID", http.StatusBadRequest)
		return
	}
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "invalid post ID", http.StatusBadRequest)
		return
	}
	postQuery := `SELECT p.id, p.title, p.content, p.created_at, p.user_id, u.username,
    COUNT(CASE WHEN pl.type = 'like' THEN 1 END) as likes_count,
    COUNT(CASE WHEN pl.type = 'dislike' THEN 1 END) as dislikes_count
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN post_likes pl ON p.id = pl.post_id
WHERE p.id = ?
GROUP BY p.id`
	var post Post
	err = db.QueryRow(postQuery, postID).Scan(&post.ID,
		&post.Title,
		&post.Content,
		&post.CreatedAt,
		&post.UserID,
		&post.Username,
		&post.LikesCount,
		&post.DislikesCount)
	if err != nil {
		if err == sql.ErrNoRows {
			RenderError(w, http.StatusNotFound)
		} else {
			RenderError(w, http.StatusInternalServerError)
		}
		return
	}
	commentsQuery := `SELECT c.id, c.content, c.created_at, c.user_id, u.username,
    COUNT(CASE WHEN cl.type = 'like' THEN 1 END) as likes_count,
    COUNT(CASE WHEN cl.type = 'dislike' THEN 1 END) as dislikes_count
FROM comments c
JOIN users u ON c.user_id = u.id
LEFT JOIN comment_likes cl ON c.id = cl.comment_id
WHERE c.post_id = ?
GROUP BY c.id
ORDER BY c.created_at ASC`
	var comments []Comment
	rows, err := db.Query(commentsQuery, postID)
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.Content, &comment.CreatedAt,
			&comment.UserID, &comment.Username,
			&comment.LikesCount, &comment.DislikesCount)
		if err != nil {
			continue
		}
		comments = append(comments, comment)
	}
	data := PostDetailsData{
		Post:     post,
		Comments: comments,
	}
	tmpl, err := template.ParseFiles("templates/post_details.html")
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func LikePost(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != "POST" {
		http.Error(w, "method are not allowed", http.StatusMethodNotAllowed)
		return
	}
	postIDstr := r.FormValue("post_id")
	postID, err := strconv.Atoi(postIDstr)
	if err != nil {
		http.Error(w, "invalid post ID", http.StatusBadRequest)
		return
	}
	userID, err := GetUserFromSession(r, db)
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}
	var existingType string
	query := `SELECT type FROM post_likes WHERE post_id = ? AND user_id = ?`
	err = db.QueryRow(query, postID, userID).Scan(&existingType)
	if err == sql.ErrNoRows {
		db.Exec(`INSERT INTO post_likes(post_id, user_id, type) VALUES(?,?,'like')`, postID, userID)
	} else if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	} else {
		if existingType == "like" {
			db.Exec(`DELETE FROM post_likes WHERE post_id = ? AND user_id = ?`, postID, userID)
		} else {
			db.Exec(`UPDATE post_likes SET type = 'like' WHERE post_id = ? AND user_id = ?`, postID, userID)
		}
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/home"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func DisLikePost(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	postIDStr := r.FormValue("post_id")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}
	userID, err := GetUserFromSession(r, db)
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}
	var existingType string
	query := `SELECT type FROM post_likes WHERE post_id = ? AND user_id = ?`
	err = db.QueryRow(query, postID, userID).Scan(&existingType)
	if err == sql.ErrNoRows {
		db.Exec(`INSERT INTO post_likes(post_id, user_id, type) VALUES(?,?,'dislike')`, postID, userID)
	} else if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	} else {
		if existingType == "dislike" {
			db.Exec(`DELETE FROM post_likes WHERE post_id = ? AND user_id = ?`, postID, userID)
		} else {
			db.Exec(`UPDATE post_likes SET type = 'dislike' WHERE post_id = ? AND user_id = ?`, postID, userID)
		}
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/home"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func DeletePost(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed ", http.StatusMethodNotAllowed)
		return
	}
	postIDStr := r.FormValue("post_id")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	userID, err := GetUserFromSession(r, db)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	query := `SELECT user_id FROM posts WHERE id = ?`
	var postOwnerID int
	err = db.QueryRow(query, postID).Scan(&postOwnerID)
	if err != nil {
		RenderError(w, http.StatusNotFound)
		return
	}
	if userID != postOwnerID {
		RenderError(w, http.StatusForbidden)
		return
	}
	query = `DELETE FROM posts WHERE id = ?`
	_, err = db.Exec(query, postID)
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func EditPost(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == http.MethodGet {
		postIDStr := r.URL.Query().Get("id")
		postID, err := strconv.Atoi(postIDStr)
		if err != nil {
			http.Error(w, "wrong post ID", http.StatusBadRequest)
			return
		}
		userID, err := GetUserFromSession(r, db)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		var postOwnerID int
		err = db.QueryRow(`SELECT user_id FROM posts WHERE id = ?`, postID).Scan(&postOwnerID)
		if err != nil {
			RenderError(w, http.StatusNotFound)
			return
		}
		if postOwnerID != userID {
			RenderError(w, http.StatusForbidden)
			return
		}
		var title, content string
		err = db.QueryRow(`SELECT title, content FROM posts WHERE id = ?`, postID).Scan(&title, &content)
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}
		tmpl, err := template.ParseFiles("templates/edit_post.html")
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}
		data := Post{
			ID:      postID,
			Title:   title,
			Content: content,
		}
		tmpl.Execute(w, data)
		return
	}
	if r.Method == http.MethodPost {
		postIDStr := r.FormValue("post_id")
		postID, err := strconv.Atoi(postIDStr)
		if err != nil {
			http.Error(w, "wrong post ID ", http.StatusBadRequest)
			return
		}
		title := r.FormValue("title")
		content := r.FormValue("content")
		categories := r.Form["categories"]
		if title == "" || content == "" || len(categories) == 0 {
			http.Error(w, "please fill all the fields", http.StatusBadRequest)
			return
		}
		userID, err := GetUserFromSession(r, db)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		var postOwnerID int
		err = db.QueryRow(`SELECT user_id FROM posts WHERE id = ?`, postID).Scan(&postOwnerID)
		if err != nil {
			RenderError(w, http.StatusNotFound)
			return
		}
		if postOwnerID != userID {
			RenderError(w, http.StatusForbidden)
			return
		}
		_, err = db.Exec(`UPDATE posts SET title = ?, content = ? WHERE id = ?`, title, content, postID)
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}
		_, err = db.Exec(`DELETE FROM post_categories WHERE post_id = ?`, postID)
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}
		for _, categoryName := range categories {
			var categoryID int
			err := db.QueryRow(`SELECT id FROM categories WHERE name = ?`, categoryName).Scan(&categoryID)
			if err == nil {
				db.Exec(`INSERT INTO post_categories(post_id, category_id) VALUES (?,?)`, postID, categoryID)
			}
		}
	}
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
