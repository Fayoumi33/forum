package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"
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
}

type Comment struct {
    ID int 
    Content string
    CreatedAt string 
    UserID int 
    Username string
    LikesCount int
    DislikesCount int
    ParentCommentID *int
    Replies []Comment
    RepliesCount int
}

type PostDetailsData struct{
    Post Post
    Comments []Comment
}

type ProfileData struct{
	Username string
	Email string
	PostsCount int
	CommentsCount int
	Posts []Post
	CurrentTab string
	CurrentUserID int
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
		http.Error(w, "Please login first", http.StatusUnauthorized)
		return
	}

	// تبسيط الـ query - نجلب البوستات أولاً ثم نحسب الـ likes والـ comments بشكل منفصل
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
		http.Error(w, "server internal error", http.StatusInternalServerError)
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
		
		// حساب الـ likes والـ dislikes
		var likesCount, dislikesCount int
		err = db.QueryRow(`SELECT 
			COUNT(CASE WHEN type = 'like' THEN 1 END) as likes,
			COUNT(CASE WHEN type = 'dislike' THEN 1 END) as dislikes
			FROM post_likes WHERE post_id = ?`, p.ID).Scan(&likesCount, &dislikesCount)
		if err == nil {
			p.LikesCount = likesCount
			p.DislikesCount = dislikesCount
		}
		
		// حساب عدد الـ comments
		var commentsCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM comments WHERE post_id = ?`, p.ID).Scan(&commentsCount)
		if err == nil {
			p.CommentsCount = commentsCount
		}
		
		// جلب الـ comments
		commentsQuery := `SELECT c.id, c.content, c.created_at, c.user_id, u.username,
			c.parent_comment_id
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ? AND c.parent_comment_id IS NULL
		ORDER BY c.created_at ASC`
		
		commentRows, err := db.Query(commentsQuery, p.ID)
		if err == nil {
			for commentRows.Next() {
				var comment Comment
				err := commentRows.Scan(&comment.ID, &comment.Content, &comment.CreatedAt,
					&comment.UserID, &comment.Username, &comment.ParentCommentID)
				if err == nil {
					// حساب likes/dislikes للـ comment
					db.QueryRow(`SELECT 
						COUNT(CASE WHEN type = 'like' THEN 1 END),
						COUNT(CASE WHEN type = 'dislike' THEN 1 END)
						FROM comment_likes WHERE comment_id = ?`, comment.ID).Scan(&comment.LikesCount, &comment.DislikesCount)
					
					// حساب عدد الردود
					db.QueryRow(`SELECT COUNT(*) FROM comments WHERE parent_comment_id = ?`, comment.ID).Scan(&comment.RepliesCount)
					
					// جلب الردود
					repliesQuery := `SELECT c.id, c.content, c.created_at, c.user_id, u.username
					FROM comments c
					JOIN users u ON c.user_id = u.id
					WHERE c.parent_comment_id = ?
					ORDER BY c.created_at ASC`
					
					replyRows, err := db.Query(repliesQuery, comment.ID)
					if err == nil {
						for replyRows.Next() {
							var reply Comment
							err := replyRows.Scan(&reply.ID, &reply.Content, &reply.CreatedAt,
								&reply.UserID, &reply.Username)
							if err == nil {
								// حساب likes/dislikes للرد
								db.QueryRow(`SELECT 
									COUNT(CASE WHEN type = 'like' THEN 1 END),
									COUNT(CASE WHEN type = 'dislike' THEN 1 END)
									FROM comment_likes WHERE comment_id = ?`, reply.ID).Scan(&reply.LikesCount, &reply.DislikesCount)
								
								comment.Replies = append(comment.Replies, reply)
							}
						}
						replyRows.Close()
					}
					
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
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	fmt.Println("Executing template...") 
	err = tmpl.Execute(w, posts)
	if err != nil {
		fmt.Println("Execute error:", err) 
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}

	fmt.Println("HomePage completed successfully") 
}

func CreatePost(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("templates/create_post.html")
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
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
		title := r.FormValue("title")
		content := r.FormValue("content")
		categories := r.Form["categories"]
		
		fmt.Println("Categories received:", categories) // للتأكد
		if title == "" || content == "" || len(categories) < 1 {
			http.Error(w, "invalid inputs", http.StatusBadRequest)
			return
		}
		userID, err := GetUserFromSession(r, db)
		if err != nil {
			http.Error(w, "server internal error", http.StatusInternalServerError)
			return
		}
		createQuery := `INSERT INTO posts (title, content, user_id, created_at) VALUES (?, ?, ?, datetime('now'))`
		result, err := db.Exec(createQuery, title, content, userID)
		if err != nil {
			fmt.Println("Create post error:", err)
			http.Error(w, "server internal error", http.StatusInternalServerError)
			return
		}
		postID, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "server internal error ", http.StatusInternalServerError)
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

func PostDetails(w http.ResponseWriter , r *http.Request ,db *sql.DB){
    postIDStr:= r.URL.Query().Get("id")
    if postIDStr == "" {
        http.Error(w,"Missing post ID",http.StatusBadRequest)
        return
    }
    postID , err := strconv.Atoi(postIDStr)
    if err != nil {
        http.Error(w,"invalid post ID",http.StatusBadRequest)
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
    err = db.QueryRow(postQuery,postID).Scan( &post.ID,
    &post.Title,
    &post.Content,
    &post.CreatedAt,
    &post.UserID,
    &post.Username,
    &post.LikesCount,
    &post.DislikesCount,)
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w,"post not found",http.StatusNotFound)
        }else{
            http.Error(w,"iternal server error",http.StatusInternalServerError)
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
rows,err := db.Query(commentsQuery,postID)
if err != nil  {
    http.Error(w,"internal server error",http.StatusInternalServerError)
    return
}
defer rows.Close()
for rows.Next(){
    var comment Comment
    err := rows.Scan(&comment.ID, &comment.Content, &comment.CreatedAt,
                    &comment.UserID, &comment.Username,
                    &comment.LikesCount, &comment.DislikesCount)
                    if err != nil  {
                        continue
                    }
                    comments = append(comments, comment)
}
    data := PostDetailsData{
        Post: post,
        Comments: comments,
    }
    tmpl,err := template.ParseFiles("templates/post_details.html")
    if err != nil{
        http.Error(w,"Error loading templates",http.StatusInternalServerError)
        return
    }
    tmpl.Execute(w, data)
} 


func AddComment(w http.ResponseWriter , r *http.Request , db *sql.DB){
    if r.Method != "POST"{
        http.Error(w," mehtod not allowed" , http.StatusMethodNotAllowed)
        return
    }
    content := r.FormValue("content")
    postIDSrt := r.FormValue("post_id")
    parentCommentIDStr := r.FormValue("parent_comment_id")

    if content == "" || postIDSrt == ""{
        http.Error(w,"missing data" , http.StatusBadRequest)
        return
    }
    postID , err := strconv.Atoi(postIDSrt)
    if err != nil {
        http.Error(w,"invalid post id " , http.StatusBadRequest)
        return
    }
    
    var parentCommentID *int
    if parentCommentIDStr != "" {
        id, err := strconv.Atoi(parentCommentIDStr)
        if err == nil {
            parentCommentID = &id
        }
    }
    
    userID, err := GetUserFromSession(r ,db)
    if err != nil {
        http.Error(w,"Unothorized" , http.StatusUnauthorized)
        return
    }
    
    if parentCommentID != nil {
        _,err = db.Exec(`INSERT INTO comments(content, post_id, user_id, parent_comment_id, created_at) VALUES (?,?,?,?,datetime('now'))`,
            content, postID, userID, *parentCommentID)
    } else {
        _,err = db.Exec(`INSERT INTO comments(content, post_id, user_id, created_at) VALUES (?,?,?,datetime('now'))`,
            content, postID, userID)
    }
    
    if err != nil {
        http.Error(w,"internal server error" , http.StatusInternalServerError)
        return
    }
    
    referer := r.Header.Get("Referer")
    if referer == "" {
        referer = "/home"
    }
    http.Redirect(w,r,referer,http.StatusSeeOther)
}


func LikePost(w http.ResponseWriter , r *http.Request , db *sql.DB){
	if r.Method != "POST" {
		http.Error(w,"method are not allowed" , http.StatusMethodNotAllowed)
		return
	}
	postIDstr:= r.FormValue("post_id")
	postID , err := strconv.Atoi(postIDstr)
	if err != nil {
		http.Error(w,  "invalid post ID" , http.StatusBadRequest)
		return
	}
	userID,err := GetUserFromSession(r,db)
	if err != nil {
		http.Error(w,"invalid user id " , http.StatusInternalServerError)
		return
	}
	var existingType string 
	query := `SELECT type FROM post_likes WHERE post_id = ? AND user_id = ?`
	err = db.QueryRow(query,postID,userID).Scan(&existingType)
	if err == sql.ErrNoRows{
		db.Exec(`INSERT INTO post_likes(post_id, user_id, type) VALUES(?,?,'like')`,postID,userID)
	}else if err != nil {
		http.Error(w,"internal server error",http.StatusInternalServerError)
		return
	}else{
		if existingType == "like"{
			db.Exec(`DELETE FROM post_likes WHERE post_id = ? AND user_id = ?` ,postID,userID)
		}else{
			db.Exec(`UPDATE post_likes SET type = 'like' WHERE post_id = ? AND user_id = ?` , postID , userID)
		}
	}
	
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/home"
	}
	http.Redirect(w,r,referer,http.StatusSeeOther)
}

func DisLikePost(w http.ResponseWriter , r *http.Request , db *sql.DB){
	if r.Method != "POST" {
		http.Error(w,"method not allowed",http.StatusMethodNotAllowed)
		return
	}
	postIDStr := r.FormValue("post_id")
	postID,err:= strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w,"invalid post id" , http.StatusBadRequest)
		return
	}
	userID,err := GetUserFromSession(r,db)
	if err != nil {
		http.Error(w,"internal server error" , http.StatusInternalServerError)
		return
	}
	var existingType string 
	query := `SELECT type FROM post_likes WHERE post_id = ? AND user_id = ?`
	err = db.QueryRow(query,postID,userID).Scan(&existingType)
	if err == sql.ErrNoRows{
		db.Exec(`INSERT INTO post_likes(post_id, user_id, type) VALUES(?,?,'dislike')`,postID,userID)
	}else if err != nil {
		http.Error(w,"internal server error",http.StatusInternalServerError)
		return
	}else{
		if existingType == "dislike" {
			db.Exec(`DELETE FROM post_likes WHERE post_id = ? AND user_id = ?`,postID,userID)
		}else{
			db.Exec(`UPDATE post_likes SET type = 'dislike' WHERE post_id = ? AND user_id = ?`,postID,userID)
		}
	}
	
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/home"
	}
	http.Redirect(w,r,referer,http.StatusSeeOther)
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
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	
	var existingType string
	query := `SELECT type FROM comment_likes WHERE comment_id = ? AND user_id = ?`
	err = db.QueryRow(query, commentID, userID).Scan(&existingType)
	
	if err == sql.ErrNoRows {
		db.Exec(`INSERT INTO comment_likes(comment_id, user_id, type) VALUES(?,?,'like')`, commentID, userID)
	} else if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
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
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	
	var existingType string
	query := `SELECT type FROM comment_likes WHERE comment_id = ? AND user_id = ?`
	err = db.QueryRow(query, commentID, userID).Scan(&existingType)
	
	if err == sql.ErrNoRows {
		db.Exec(`INSERT INTO comment_likes(comment_id, user_id, type) VALUES(?,?,'dislike')`, commentID, userID)
	} else if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
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

func ProfilePage(w http.ResponseWriter , r *http.Request , db *sql.DB){
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
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }
    
    fmt.Println("Username:", username, "Email:", email)
	
	var postsCount int 
	err = db.QueryRow(`SELECT COUNT(*) FROM posts WHERE user_id = ?`,userID).Scan(&postsCount)
	if err != nil {
		postsCount = 0
	}
	var commentsCount int 
	err = db.QueryRow(`SELECT COUNT(*) FROM comments WHERE user_id = ?`,userID).Scan(&commentsCount)
	if err != nil  {
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
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		Username: username,
		Email: email,
		PostsCount: postsCount,
		CommentsCount: commentsCount,
		Posts: posts,
		CurrentTab: currentTab,
		CurrentUserID: userID,
	}
	tmpl,err := template.ParseFiles("templates/profile.html")
	if err != nil {
		http.Error(w,"internal server error",http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w,data)
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
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}

		data := struct {
			Username string
			Email    string
		}{username, email}

		tmpl, err := template.ParseFiles("templates/edit_profile.html")
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
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
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(currentPassword))
		if err != nil {
			http.Error(w, "current password incorrect", http.StatusUnauthorized)
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
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			_, err = db.Exec(`UPDATE users SET username = ?, email = ?, password = ? WHERE id = ?`,
				username, email, string(hashedPassword), userID)
			if err != nil {
				http.Error(w, "update failed", http.StatusInternalServerError)
				return
			}
		} else {
			_, err = db.Exec(`UPDATE users SET username = ?, email = ? WHERE id = ?`,
				username, email, userID)
			if err != nil {
				http.Error(w, "update failed", http.StatusInternalServerError)
				return
			}
		}

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}  

func DeletePost(w http.ResponseWriter , r *http.Request , db *sql.DB){
	if r.Method != http.MethodPost {
		http.Error(w , "method not allowed " , http.StatusMethodNotAllowed)
		return
	}
	postIDStr:= r.FormValue("post_id")
	postID , err := strconv.Atoi(postIDStr)
	if err != nil  {
		http.Redirect(w , r , "/login" , http.StatusSeeOther)
		return
	}
	userID , err := GetUserFromSession(r , db)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	query := `SELECT user_id FROM posts WHERE id = ?`
	var postOwnerID int
	err = db.QueryRow(query , postID).Scan(&postOwnerID)
	if err != nil  {
		http.Error(w , "post not found ", http.StatusNotFound)
		return
	}
	if userID != postOwnerID {
		http.Error(w , "you are not the owner of the post" , http.StatusForbidden)
		return
	}
	query = `DELETE FROM posts WHERE id = ?`
	_,err = db.Exec(query , postID)
	if err != nil {
		http.Error(w , "internal server error " , http.StatusInternalServerError)
		return
	}
	http.Redirect(w , r ,"/profile" , http.StatusSeeOther)
}

func EditPost(w http.ResponseWriter, r *http.Request , db *sql.DB){
	if r.Method == http.MethodGet {
		postIDStr := r.URL.Query().Get("id")
		postID ,err := strconv.Atoi(postIDStr)
		if err != nil {
			http.Error(w , "wrong post ID", http.StatusBadRequest)
			return
		}
		userID, err  := GetUserFromSession(r,db)
		if err != nil  {
			http.Redirect(w , r , "/login" , http.StatusSeeOther)
			return
		}
		var postOwnerID int 
		err = db.QueryRow(`SELECT user_id FROM posts WHERE id = ?` , postID).Scan(&postOwnerID)
		if err != nil  {
			http.Error(w , " user not found " , http.StatusNotFound)
			return
		}
		if postOwnerID != userID {
			http.Error(w , " you are not the owner of this post " , http.StatusForbidden)
			return
		}
		var title , content string
		err = db.QueryRow(`SELECT title, content FROM posts WHERE id = ?`,postID).Scan(&title , &content)
		if err != nil  {
			http.Error(w,"server internal error " , http.StatusInternalServerError)
			return
		}
		tmpl ,err := template.ParseFiles("templates/edit_post.html")
		if err != nil  {
			http.Error(w , "server internal error " , http.StatusInternalServerError)
			return
		}
		data := Post{
			ID: postID,
			Title: title,
			Content: content,
		}
		tmpl.Execute(w,data)
		return
	}
	if r.Method == http.MethodPost {
		postIDStr := r.FormValue("post_id")
		postID , err := strconv.Atoi(postIDStr)
		if err != nil {
			http.Error(w , "wrong post ID " , http.StatusBadRequest)
			return
		}
		title := r.FormValue("title")
		content := r.FormValue("content")
		categories := r.Form["categories"]
		if title == "" || content == "" || len(categories) == 0 {
			http.Error(w , "please fill all the fields" , http.StatusBadRequest)
			return
		}
		userID , err := GetUserFromSession(r,db)
		if err != nil {
			http.Redirect(w , r , "/login" , http.StatusSeeOther)
			return
		}
		var postOwnerID int 
		err = db.QueryRow(`SELECT user_id FROM posts WHERE id = ?` , postID).Scan(&postOwnerID)
		if err != nil  {
			http.Error(w , "user not found ",http.StatusNotFound)
			return
		}
		if postOwnerID != userID {
			http.Error(w," you are not the owner of this post",http.StatusForbidden)
			return
		}
		_ , err = db.Exec(`UPDATE posts SET title = ?, content = ? WHERE id = ?` , title , content , postID)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		_,err = db.Exec(`DELETE FROM post_categories WHERE post_id = ?`,postID)
		for _, categoryName := range categories{
			var categoryID int 
			err := db.QueryRow(`SELECT id FROM categories WHERE name = ?` , categoryName).Scan(&categoryID)
			if err == nil  {
				db.Exec(`INSERT INTO post_categories(post_id, category_id) VALUES (?,?)` , postID , categoryID)
			}
		}
	}
	http.Redirect(w , r , "/profile" , http.StatusSeeOther)
}