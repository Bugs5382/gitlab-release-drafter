FROM golang:1.26 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/gitlab-release-drafter ./cmd/gitlab-release-drafter

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gitlab-release-drafter /usr/local/bin/gitlab-release-drafter
ENTRYPOINT ["/usr/local/bin/gitlab-release-drafter"]
