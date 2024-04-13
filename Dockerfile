FROM golang:1.22.2-alpine3.19 as build
RUN apk add --no-cache --update go gcc g++
WORKDIR /app
ENV CGO_ENABLED=1
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/build/file-server main.go


FROM golang:1.22.2-alpine3.19 as run
WORKDIR /app
COPY --from=build /app/build/file-server .
EXPOSE 8080
CMD ["/app/file-server"]
