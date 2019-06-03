FROM golang:1.12-alpine

RUN apk add git

RUN mkdir -p /app
WORKDIR /app

COPY . .

RUN go build -o app-api .

CMD /app/app-api
