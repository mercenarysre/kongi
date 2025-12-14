FROM golang:latest AS build

WORKDIR /app

COPY go.mod .

COPY go.sum .

RUN go mod download 

COPY . . 

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o kongi .


FROM gcr.io/distroless/static

COPY --from=build /app/kongi /kongi 

EXPOSE 8080

ENTRYPOINT ["/kongi"]