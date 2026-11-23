FROM 416670754337.dkr.ecr.eu-west-2.amazonaws.com/ci-golang-build-1.26:latest

WORKDIR /opt

COPY /app .
COPY docker_start.sh .

# Run as user `nobody` to avoid running as root
USER 65534:65534

CMD ["./docker_start.sh"]
