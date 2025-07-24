FROM golang:1.24.4-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o url-shortener-go ./cmd

FROM alpine AS prod

WORKDIR /app

COPY --from=build /app/url-shortener-go /app/url-shortener-go

EXPOSE 8080

CMD ["./url-shortener-go"]