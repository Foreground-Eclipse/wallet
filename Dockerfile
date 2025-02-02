FROM golang:1.22

WORKDIR / 

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN go build -o /wallet ./cmd/server/main.go


EXPOSE 8088

CMD ["/wallet"]