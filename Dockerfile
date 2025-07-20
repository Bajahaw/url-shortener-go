FROM golang:1.24.4-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd

FROM scratch AS prod

WORKDIR /app

COPY --from=build /app/main /app/main

EXPOSE 8080

CMD ["./main"]