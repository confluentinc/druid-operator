# Build the manager binary
FROM golang:1.20.6 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY apis/ apis/
COPY controllers/ controllers/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN if [ "$TARGETOS" = "linux" ] && [ "$TARGETARCH" = "amd64" -o "$TARGETARCH" = "x86_64" ]; then \
		echo "Turning on boringcrypto" && \
		CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=amd64 GOEXPERIMENT=boringcrypto go build -a -o manager main.go; \
	else \
		CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go; \
	fi


# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM 519856050701.dkr.ecr.us-west-2.amazonaws.com/docker/prod/confluentinc/lobo-dynamic-fips:latest-202405310458
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
