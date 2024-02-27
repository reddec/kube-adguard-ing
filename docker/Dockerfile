FROM --platform=$BUILDPLATFORM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY kube-adguard-ing /bin/kube-adguard-ing
ENTRYPOINT ["/bin/kube-adguard-ing"]