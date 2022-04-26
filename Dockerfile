FROM golang:1.17-alpine
WORKDIR /app
ENV TZ=Asia/Singapore
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
COPY . .
RUN go build -o /runner ./cmd/runner
EXPOSE 10000
CMD [ "/runner" , "/app/config"]
