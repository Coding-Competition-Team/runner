FROM golang:1.17-alpine
WORKDIR /app

COPY . .
RUN go build -o /runner ./cmd/runner
EXPOSE 10000
CMD [ "/runner" , "/app/config"]
