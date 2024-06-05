FROM golang:alpine


WORKDIR /app
RUN apk add nodejs make npm
RUN go install github.com/a-h/templ/cmd/templ@latest

COPY . .

ENV GOPATH=/root/go

RUN npm i

ENTRYPOINT make build && make run




