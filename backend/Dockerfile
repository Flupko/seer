FROM golang:1.24-alpine AS base-build

WORKDIR /app 

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go mod download


FROM base-build AS dev

RUN apk add --no-cache git
RUN go install github.com/air-verse/air@v1.62.0

COPY . .

CMD ["air", "-c", ".air.toml"]