# Forum Web Application

A web forum that allows communication between users with posts, comments, likes and dislikes.

## Features

- User registration and authentication (email, username, password)
- Session management with cookies
- Create posts with categories
- Comment on posts
- Like/dislike posts and comments
- Filter posts by categories, created posts, and liked posts
- Password encryption with bcrypt

## Technologies

- Go
- SQLite
- HTML/CSS
- Docker

## Running Locally

```bash
go run main.go
```

Access at: http://localhost:8080

## Running with Docker

Build:
```bash
docker build -t forum .
```

Run:
```bash
docker run -p 8080:8080 forum
```

Access at: http://localhost:8080

## Project Structure

```
forum/
├── go.mod, go.sum          # Dependency management
├── main.go                 # Application entry point
├── database/               # Database operations
├── handlers/               # Authentication & HTTP handlers
├── styles/                 # CSS styling
└── templates/              # HTML templates
```

## Authors

- Mohamed Darwish (#mohamdarwish)
- Ebrahim Alfayoumi (#ealfayou)
- Saeed Alsayeg (#salsayeg)
- Osama Essa (#osessa)
