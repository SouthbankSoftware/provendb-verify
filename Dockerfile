FROM alpine:3.10.3

RUN wget https://storage.googleapis.com/provendb-dev/provendb-verify/provendb-verify_linux_amd64 -O /bin/provendb-verify && \
chmod a+x /bin/provendb-verify && \
apk add --no-cache libc6-compat ca-certificates 

CMD ["/bin/ash", "-c", "provendb-verify -h"]