#!/bin/bash

set -e

cfssl gencert -initca ca-csr.json | cfssljson -bare ca
cfssl gencert \
  -ca=ca.pem \
  -ca-key=ca-key.pem \
  -config=ca-config.json \
  -hostname=127.0.0.1,container-analysis-webhook,container-analysis-webhook.kube-system,container-analysis-webhook.default,container-analysis-webhook.default.svc \
  -profile=default \
  container-analysis-webhook-csr.json | cfssljson -bare container-analysis-webhook

kubectl create secret tls tls-container-analysis-webhook \
  --cert=container-analysis-webhook.pem \
  --key=container-analysis-webhook-key.pem
