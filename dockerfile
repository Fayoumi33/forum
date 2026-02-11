FROM golang:1.21

LABEL version="1.0"
LABEL description="Forum is a website that allows users to communicate through posts,comments ,and likes."

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN go build -o forum ./main.go

COPY templates ./templates
COPY styles ./styles
COPY database ./database
COPY handlers ./handlers

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/home || exit 1

CMD ["./forum"]
