# Container Analysis Demo



## Tutorial

### Infrastructure

A Kubernetes 1.8+ cluster is required with support for the [external admission
webhooks](https://kubernetes.io/docs/admin/extensible-admission-controllers/#external-admission-webhooks)
alpha feature enabled.

If you have access to [Google Container
Engine](https://cloud.google.com/container-engine/) use the gcloud command to
create a 1.8 Kubernetes cluster:

```
gcloud alpha container clusters create grafeas \
  --enable-kubernetes-alpha \
  --cluster-version 1.8.4-gke.0
```

> Any Kubernetes 1.8 cluster with support for external admission webhooks will work. 

### Deploy the Image Signature Webhook

Create the `tls-image-signature-webhook` secret and store the TLS certs:

```
kubectl create secret tls tls-image-signature-webhook \
  --key pki/image-signature-webhook-key.pem \
  --cert pki/image-signature-webhook.pem
```

Create the `image-signature-webhook` deployment:

```
kubectl apply -f kubernetes/image-signature-webhook.yaml 
```

Create the `image-signature-webook` [ExternalAdmissionHookConfiguration](https://kubernetes.io/docs/admin/extensible-admission-controllers/#how-are-external-admission-webhooks-triggered):

```
kubectl apply -f kubernetes/admission-hook-configuration.yaml
```

> After you create the external admission hook configuration, the system will take a few seconds to honor the new configuration.

### Testing the Admission Webhook



## Cleanup

Run the following commands to remove the Kubernetes resources created during this tutorial:

```
kubectl delete deployments grafeas image-signature-webhook
kubectl delete pods echod
kubectl delete svc grafeas image-signature-webhook
kubectl delete secrets tls-image-signature-webhook
kubectl delete configmap image-signature-webhook
```
