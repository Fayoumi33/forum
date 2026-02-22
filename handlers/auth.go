package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type login struct {
	hashedPassword string
	sessionToken   string
	CSRFToken      string
}

func renderRegisterWithError(w http.ResponseWriter, errorMsg string) {
	tmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, map[string]string{"Error": errorMsg})
}

func Register(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("templates/register.html")
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
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
		if username == "" || email == "" || password == "" {
			renderRegisterWithError(w, "Please fill all the fields")
			return
		}
		if password != confirmPassword {
			renderRegisterWithError(w, "Passwords do not match")
			return
		}
		if !(strings.Contains(email, "@")) {
			renderRegisterWithError(w, "Invalid email address")
			return
		}
		var userID int
		usernameQuery := `select id from users where username = ?`
		err := db.QueryRow(usernameQuery, username).Scan(&userID)
		if err == nil {
			renderRegisterWithError(w, "Username already exists")
			return
		}
		var emailID int
		emailQuery := `select id from users where email = ?`
		err = db.QueryRow(emailQuery, email).Scan(&emailID)
		if err == nil {
			renderRegisterWithError(w, "Email already registered")
			return
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}
		insertUser := `insert into users (username , email , password) values (?,?,?)`
		_, err = db.Exec(insertUser, username, email, string(hashedPassword))
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
}

func renderLoginWithError(w http.ResponseWriter, errorMsg string) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		RenderError(w, http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, map[string]string{"Error": errorMsg})
}

func Login(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			RenderError(w, http.StatusInternalServerError)
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
			renderLoginWithError(w, "Please fill all the fields")
			return
		}

		fmt.Println("Querying database for email:", email)
		var userID int
		var storedHashedPassword string
		query := `select id,password from users where email = ?`
		err := db.QueryRow(query, email).Scan(&userID, &storedHashedPassword)
		if err != nil {
			fmt.Println("ERROR: Database query failed:", err)
			renderLoginWithError(w, "Invalid email or password")
			return
		}

		fmt.Println("User found! ID:", userID)
		fmt.Println("Checking password...")

		err = bcrypt.CompareHashAndPassword([]byte(storedHashedPassword), []byte(password))
		if err != nil {
			fmt.Println("ERROR: Password wrong:", err)
			renderLoginWithError(w, "Invalid email or password")
			return
		}

		fmt.Println("Password correct!")

		fmt.Println("Creating session...")
		sessionID := uuid.New().String()

		fmt.Println("Session ID:", sessionID)

		insertSession := `insert into sessions (id, user_id, expires_at) values (?, ?, datetime('now', '+24 hours'))`
		_, err = db.Exec(insertSession, sessionID, userID)
		if err != nil {
			fmt.Println("ERROR: Session insert failed:", err)
			RenderError(w, http.StatusInternalServerError)
			return
		}

		fmt.Println("Session created successfully!")

		cookie := &http.Cookie{
			Name:     "session_token",
			Value:    sessionID,
			Expires:  time.Now().Add(24 * time.Hour), // ← هذا يبقى time.Time (للـ cookie)
			HttpOnly: true,
			Path:     "/",
		}
		http.SetCookie(w, cookie)

		fmt.Println("Cookie set! Redirecting to /home")
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	} // ← القوس هنا!
}

func Logout(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	cookie, err := r.Cookie("session_token")
	if err == nil {
		sessionID := cookie.Value
		deleteSession := `delete from sessions where id = ?`
		_, err = db.Exec(deleteSession, sessionID)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
