# Builder: compile static binary (CGO disabled).
FROM golang:1.26.3-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

ENV CGO_ENABLED=0

RUN go build \
	-ldflags "-X github.com/mab-go/trello-mcp/internal/version.Version=${VERSION} -X github.com/mab-go/trello-mcp/internal/version.Commit=${COMMIT} -X github.com/mab-go/trello-mcp/internal/version.Date=${DATE}" \
	-o /trello-mcp \
	./cmd/trello-mcp

# Runtime: minimal image with only the binary (stdio MCP).
FROM gcr.io/distroless/static-debian12@sha256:20bc6c0bc4d625a22a8fde3e55f6515709b32055ef8fb9cfbddaa06d1760f838

LABEL io.modelcontextprotocol.server.name="io.github.mab-go/trello-mcp"

COPY --from=builder /trello-mcp /trello-mcp

# Config must be mounted at runtime, e.g.:
#   -v ~/.config/trello-mcp:/home/nonroot/.config/trello-mcp
ENV HOME=/home/nonroot
USER 65532:65532
ENTRYPOINT ["/trello-mcp"]
