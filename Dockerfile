# Build the manager binary
FROM golang:1.15.6 as builder

# Copy in the go src
COPY . /go/src/github.com/Ridecell/ridecell-operator
WORKDIR /go/src/github.com/Ridecell/ridecell-operator

# Build
RUN make dep generate manifests && \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager -tags release github.com/Ridecell/ridecell-operator/cmd/manager && \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o install_crds -tags release github.com/Ridecell/ridecell-operator/cmd/install_crds && \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o initcontainer -tags release github.com/Ridecell/ridecell-operator/cmd/initcontainer && \
  cd cmd/webui && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 packr2 build -a -tags release

# Copy the controller-manager into a thin image
FROM alpine:latest
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /go/src/github.com/Ridecell/ridecell-operator/manager /ridecell-operator
COPY --from=builder /go/src/github.com/Ridecell/ridecell-operator/install_crds /install_crds
COPY --from=builder /go/src/github.com/Ridecell/ridecell-operator/initcontainer /initcontainer
COPY --from=builder /go/src/github.com/Ridecell/ridecell-operator/cmd/webui/webui /webui
COPY --from=builder /go/src/github.com/Ridecell/ridecell-operator/config/crds /crds
CMD ["/ridecell-operator"]
