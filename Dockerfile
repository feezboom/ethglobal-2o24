FROM golang:1.18
LABEL authors="akaryamin"

WORKDIR /app

COPY . .

RUN go build -o main .

EXPOSE 8080

CMD ["./main"]
