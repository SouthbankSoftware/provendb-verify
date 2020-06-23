FROM frolvlad/alpine-glibc:glibc-2.31

RUN wget https://storage.googleapis.com/provendb-dev/provendb-verify/provendb-verify_linux_amd64 -O /bin/provendb-verify && chmod a+x /bin/provendb-verify

CMD ["/bin/ash", "-c", "provendb-verify -h"]