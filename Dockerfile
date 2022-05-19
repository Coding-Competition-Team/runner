FROM golang:1.17-alpine
WORKDIR /app
ENV TZ=Asia/Singapore
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
COPY . .
RUN chmod +x runner
EXPOSE 10000
CMD [ "/app/runner" , "/app/config"]
