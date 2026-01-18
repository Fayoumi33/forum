package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strings"
	"time"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"fmt"
)

type login struct {
	hashedPassword string
	sesstionToken  string
	CSRFToken     string
}

func Register(w http.ResponseWriter, r *http.Request , db *sql.DB) {
	if r.Method == "GET" {
		tmpl,err := template.ParseFiles("templates/register.html")
		if err != nil {
			http.Error (w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}
	if r.Method == "POST" {
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")
		if password != confirmPassword {
			http.Error(w, "Passwords do not match" , http.StatusBadRequest)
			return
		}
		if username == "" || email == "" || password == "" {
			http.Error(w, "Please fill all the fields" , http.StatusBadRequest)
			return
		}
		if !(strings.Contains(email,"@")) {
			http.Error(w, "Invalid email address", http.StatusBadRequest)
			return
		}
		var userID int 
		usernameQuery := `select id from users where username = ?`
		err := db.QueryRow(usernameQuery , username).Scan(&userID)
		if(err == nil){
			http.Error(w, "Username already exists", http.StatusBadRequest)
			return
		}
		var emailID int
		emailQuery := `select id from users where email = ?`
		err  = db.QueryRow(emailQuery , email).Scan(&emailID)
		if(err == nil){
			http.Error(w, "Email already registered", http.StatusBadRequest)
			return
		}
		hashedPassword ,err := bcrypt.GenerateFromPassword([]byte(password) , bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Internal Server Error" , http.StatusInternalServerError)
			return
		}
		insertUser := `insert into users (username , email , password) values (?,?,?)`
		_, err = db.Exec(insertUser , username , email , string(hashedPassword))
		if err != nil {
			http.Error(w, "Internal Server Error" , http.StatusInternalServerError)
			return
		}
		http.Redirect(w , r , "/login" , http.StatusSeeOther)
	return
	}
}

func Login(w http.ResponseWriter, r *http.Request , db *sql.DB) {
	if r.Method == "GET" {
		tmpl,err := template.ParseFiles("templates/index.html")
		if err != nil {
			http.Error(w, " internal server error" , http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}
	if r.Method == "POST" {
    fmt.Println("=== LOGIN POST STARTED ===")
    
    email := r.FormValue("email")
    password := r.FormValue("password")
    
    fmt.Println("Email:", email)
    fmt.Println("Password:", password)
    
    if email == "" || password == "" {
        fmt.Println("ERROR: Empty fields")
        http.Error(w, "Please fill all the fields", http.StatusBadRequest)
        return
    }
    
    fmt.Println("Querying database for email:", email)
    var userID int
    var storedHashedPassword string
    query := `select id,password from users where email = ?`
    err := db.QueryRow(query, email).Scan(&userID, &storedHashedPassword)
    if err != nil {
        fmt.Println("ERROR: Database query failed:", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    
    fmt.Println("User found! ID:", userID)
    fmt.Println("Checking password...")
    
    err = bcrypt.CompareHashAndPassword([]byte(storedHashedPassword), []byte(password))
    if err != nil {
        fmt.Println("ERROR: Password wrong:", err)
        http.Error(w, "Invalid email or password", http.StatusBadRequest)
        return
    }
    
fmt.Println("Password correct!")

fmt.Println("Creating session...")
sessionID := uuid.New().String()
expiresAt := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")

fmt.Println("Session ID:", sessionID)
fmt.Println("Expires at:", expiresAt)

insertSession := `insert into sessions (id, user_id, expires_at) values (?, ?, datetime('now' , '+24 hours'))`
_, err = db.Exec(insertSession, sessionID, userID, expiresAt)
if err != nil {
    fmt.Println("ERROR: Session insert failed:", err)
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
}

fmt.Println("Session created successfully!")

cookie := &http.Cookie{
    Name:    "session_token",
    Value:   sessionID,
    Expires: time.Now().Add(24 * time.Hour),  // ← هذا يبقى time.Time (للـ cookie)
    HttpOnly: true,
    Path:    "/",
}
http.SetCookie(w, cookie)

fmt.Println("Cookie set! Redirecting to /home")
http.Redirect(w, r, "/home", http.StatusSeeOther)
return
}  // ← القوس هنا!
}


func Logout(w http.ResponseWriter, r *http.Request , db *sql.DB) {

	cookie , err := r.Cookie("session_token")
	if err == nil {
		sessionID := cookie.Value
	deleteSession := `delete from sessions where id = ?`
	_,err = db.Exec(deleteSession,sessionID)
	}
	http.SetCookie(w , &http.Cookie{
		Name: "session_token",
		Value: "",
		Expires: time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Path: "/",
	})
	http.Redirect(w,r,"/login" , http.StatusSeeOther)
}