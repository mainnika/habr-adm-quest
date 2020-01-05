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
RUN go build -o task3 github.com/mainnika/habr-adm-quest/task3-backend

FROM alpine

ENV GOPATH /root/go
ENV PATH ${GOPATH}/bin:${PATH}

RUN mkdir -p /app /app/postbox
WORKDIR /app

COPY --from=builder /app/task3 .
COPY --from=builder /app/task3-backend/config config

EXPOSE 8081
EXPOSE 31337

VOLUME /app/postbox

CMD ["/app/task3"]