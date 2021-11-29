# ---------------------------
# Use distroless as minimal base image to package the final binary
# Refer https://github.com/GoogleContainerTools/distroless
# ---------------------------
FROM gcr.io/distroless/static:nonroot
WORKDIR /

COPY ./bin/k8s-experiments /usr/bin/

USER nonroot:nonroot

CMD ["/usr/bin/k8s-experiments"]