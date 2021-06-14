# Build the chat-roulette server binary
FROM golang:1.16 as builder

WORKDIR /workspace

# Copy the go source
COPY src/simple-chat/* ./

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on  go build -a -o chat-server server.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/chat-server .
COPY src/simple-chat/index.html .
USER nonroot:nonroot

VOLUME /data

EXPOSE 8080

ENTRYPOINT ["/chat-server"]
