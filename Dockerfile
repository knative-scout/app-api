FROM golang:1.12-alpine

RUN apk add git
RUN adduser -D app

USER app
WORKDIR /home/app

COPY . .

RUN go build -o serverless-registry-api .

CMD /home/app/serverless-registry-api
