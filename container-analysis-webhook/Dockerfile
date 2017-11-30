FROM ubuntu:17.04
RUN apt-get update && apt-get install -y ca-certificates
ADD container-analysis-webhook /usr/local/bin/container-analysis-webhook
ENTRYPOINT ["/usr/local/bin/container-analysis-webhook"]
