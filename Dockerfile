FROM golang:1.18 as builder
WORKDIR /workspace
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make all

FROM alpine
RUN apk add --no-cache ffmpeg p7zip
WORKDIR /
COPY --from=builder /workspace/razbox .
COPY --from=builder /workspace/mkfolder .
COPY --from=builder /workspace/mkfile .
ENTRYPOINT ["/razbox"]
