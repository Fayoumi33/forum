# Forum Web Application

A full-stack web forum built with Go that enables users to communicate through posts, comments, and reactions.

---

## Features

| Feature | Description |
|---|---|
| Authentication | Register and log in with email, username, and password |
| Session Management | Cookie-based sessions with UUID tokens |
| Posts | Create, edit, delete, and view posts with category tags |
| Comments | Add comments to any post |
| Reactions | Like or dislike posts and comments |
| Filtering | Filter posts by category, your own posts, or posts you liked |
| Profile | View and edit your profile information |
| Security | Passwords hashed with bcrypt; protected routes require authentication |
| Error Pages | Custom error pages for 400, 401, 403, 404, and 500 responses |

---

## Tech Stack

- **Backend:** Go (standard library `net/http`)
- **Database:** SQLite (via `modernc.org/sqlite` — no CGO required)
- **Frontend:** HTML templates, CSS
- **Containerization:** Docker

---

## Project Structure

```
forum/
├── main.go               # Entry point — route registration & server startup
├── go.mod / go.sum       # Go module dependencies
├── dockerfile            # Docker build configuration
├── database/
│   └── db.go             # Database initialization and schema
├── handlers/
│   ├── auth.go           # Register, login, logout
│   ├── sessions.go       # Session creation and validation
│   ├── post.go           # Create, edit, delete, view posts
│   ├── comment.go        # Add comments
│   ├── profile.go        # Profile view and edit
│   └── errors.go         # Error page rendering
├── templates/            # HTML templates (home, post, profile, errors…)
└── styles/               # CSS stylesheets
```

---

## Routes

| Method | Path | Auth Required | Description |
|---|---|---|---|
| GET/POST | `/register` | No | User registration |
| GET/POST | `/login` | No | User login |
| GET | `/logout` | No | Log out and clear session |
| GET | `/home` | No | Home page with post feed |
| GET/POST | `/create-post` | Yes | Create a new post |
| GET | `/post` | No | View a post and its comments |
| GET/POST | `/edit-post` | Yes | Edit an existing post |
| POST | `/delete-post` | Yes | Delete a post |
| POST | `/add-comment` | Yes | Comment on a post |
| POST | `/like-post` | Yes | Like a post |
| POST | `/dislike-post` | Yes | Dislike a post |
| POST | `/like-comment` | Yes | Like a comment |
| POST | `/dislike-comment` | Yes | Dislike a comment |
| GET | `/profile` | Yes | View own profile |
| GET/POST | `/edit-profile` | Yes | Edit profile information |

---

## Running Locally

**Prerequisites:** Go 1.21+

```bash
git clone <repo-url>
cd forum
go run main.go
```

Open your browser at: [http://localhost:8000](http://localhost:8000)

---

## Running with Docker

**Build the image:**
```bash
docker build -t forum .
```

**Run the container:**
```bash
docker run -p 8080:8080 forum
```

Open your browser at: [http://localhost:8080](http://localhost:8080)

---

## Dependencies

| Package | Purpose |
|---|---|
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGO) |
| `github.com/google/uuid` | Session token generation |
| `golang.org/x/crypto` | bcrypt password hashing |

---

## Authors

- Mohamed Darwish — [@mohamdarwish](https://github.com/mohamdarwish)
- Ebrahim Alfayoumi — [@ealfayou](https://github.com/ealfayou)
- Saeed Alsayeg — [@salsayeg](https://github.com/salsayeg)
- Osama Essa — [@osessa](https://github.com/osessa)
