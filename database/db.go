package database

import (
	"database/sql"
	"log"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB() *sql.DB {
	db, err := sql.Open("sqlite3" , "forum.db")
	if err != nil {
		log.Fatal(err)
	}
	createTables(db)
	insertDefaultCategories(db)
	return db
}

func createTables(db *sql.DB) {

	users := `create table if not exists users(
	id integer primary key autoincrement,
	username text unique,
	email text unique,
	password text
	)`;

	posts := `create table if not exists posts(
	id integer primary key autoincrement,
	title text not null,
	content text,
	created_at datetime default current_timestamp,
	user_id integer not null,
	foreign key(user_id) references users(id)
	)`;

	comments := `create table if not exists comments(
	id integer primary key autoincrement,
	content text,
	created_at datetime default current_timestamp,
	user_id integer not null,
	post_id integer not null,
	parent_comment_id integer,
	foreign key(user_id) references users(id),
	foreign key(post_id) references posts(id),
	foreign key(parent_comment_id) references comments(id)
	)`;

	post_likes := `create table if not exists post_likes(
	id integer primary key autoincrement,
	type text not null,
	post_id integer not null,
	user_id integer not null,
	foreign key(post_id) references posts(id),
	foreign key(user_id) references users(id)
	)`;

	comments_likes := `create table if not exists comment_likes(
	id integer primary key autoincrement,
	type text not null,
	comment_id integer not null,
	user_id integer not null,
	foreign key(comment_id) references comments(id),
	foreign key(user_id) references users(id)
	)`;
	categories := `create table if not exists categories(
	id integer primary key autoincrement,
	name text unique not null
	)`
	post_categories := `create table if not exists post_categories(
	id integer primary key autoincrement,
	post_id integer not null,
	category_id integer not null,
	foreign key(post_id) references posts(id),
	foreign key(category_id) references categories(id)
	)`
	sessions := `create table if not exists sessions(
	id text primary key,
	user_id integer not null,
	expires_at text not null,
	foreign key(user_id) references users(id)
	)`;

	queries := []string{users,posts,comments,post_likes,comments_likes,categories,post_categories,sessions}
	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func insertDefaultCategories(db *sql.DB) {
    categories := []string{"Technology", "Sport", "Gaming", "News", "Programming", "Quran", "Other"}
    
    for _, cat := range categories {
        var count int
        db.QueryRow(`SELECT COUNT(*) FROM categories WHERE name = ?`, cat).Scan(&count)
        
        if count == 0 {
            db.Exec(`INSERT INTO categories (name) VALUES (?)`, cat)
        }
    }
}