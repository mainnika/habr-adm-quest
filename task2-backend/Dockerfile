FROM alpine as builder

ENV GOPATH /root/go
ENV PATH ${GOPATH}/bin:${PATH}

RUN mkdir -p /app
WORKDIR /app

RUN apk add --no-cache go git curl gcc g++ make

COPY go.sum .
COPY go.mod .
RUN go mod download

COPY . .
RUN go build -o task2 github.com/mainnika/habr-adm-quest/task2-backend

FROM alpine

ENV GOPATH /root/go
ENV PATH ${GOPATH}/bin:${PATH}

RUN mkdir -p /app
WORKDIR /app

COPY --from=builder /app/task2 .
COPY --from=builder /app/task2-backend/config config

EXPOSE 8081

CMD ["/app/task2"]