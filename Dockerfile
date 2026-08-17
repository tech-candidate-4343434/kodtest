FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o guestbook

FROM scratch
COPY --from=builder /app/guestbook /guestbook
EXPOSE 8080
CMD [ "/guestbook" ]
