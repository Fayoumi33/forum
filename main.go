package main

import (
    "database/sql"
    "fmt"
    "forum/database"
    "forum/handlers"
    "net/http"
)

var db *sql.DB

func main() {
    fmt.Println("Initializing database...")  
    db = database.InitDB()
    fmt.Println("Database initialized")  
    
    http.Handle("/styles/",
    http.StripPrefix("/styles/",
        http.FileServer(http.Dir("./styles"))))

    
    http.HandleFunc("/", homeHandler)
    
    http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("=== Register handler called ===")  
        handlers.Register(w, r,db)
    })
    
    http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("=== Login handler called - Method:", r.Method, "===")  
        handlers.Login(w, r,db)
    })
    
    http.HandleFunc("/home", handlers.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("=== Home handler called ===") 
        handlers.HomePage(w, r, db)
    }))
    
    http.HandleFunc("/create-post" , handlers.RequireAuth(db , func(w http.ResponseWriter , r *http.Request) {
        handlers.CreatePost(w , r , db)
    }))

    http.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
        handlers.PostDetails(w, r, db)
    })

    http.HandleFunc("/add-comment", handlers.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
        handlers.AddComment(w, r, db)
    }))

    http.HandleFunc("/like-post", handlers.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
        handlers.LikePost(w, r, db)
    }))

    http.HandleFunc("/dislike-post", handlers.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
        handlers.DisLikePost(w, r, db)
    }))

    http.HandleFunc("/like-comment", handlers.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
        handlers.LikeComment(w, r, db)
    }))

    http.HandleFunc("/dislike-comment", handlers.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
        handlers.DislikeComment(w, r, db)
    }))

    http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
        handlers.Logout(w, r, db)
    })

    http.HandleFunc("/profile", handlers.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
        handlers.ProfilePage(w, r, db)
    }))

    http.HandleFunc("/edit-profile", handlers.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
        handlers.EditProfile(w, r, db)
    }))

    http.HandleFunc("/edit-post", handlers.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
        handlers.EditPost(w, r, db)
    }))

    // هذا السطر كان ناقص! 👇
    http.HandleFunc("/delete-post", handlers.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
        handlers.DeletePost(w, r, db)
    }))

    fmt.Println("Server started at http://localhost:8000")
    http.ListenAndServe(":8000", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Println("=== Root handler called - Path:", r.URL.Path, "===") 
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }
    http.Redirect(w, r, "/login", http.StatusSeeOther)
}