FROM golang:1.19

WORKDIR /app

COPY go.mod main.go ./

RUN go build -o chat .

# Expose the port on which the application listens
EXPOSE 8888 1234

CMD ["./chat"]