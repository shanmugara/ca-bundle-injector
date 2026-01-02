# ca-bundle-injector

A Kubernetes Mutating Admission Webhook to inject CA bundles into pods or resources that require trusted certificates.

## Features
- Mutates pod specs to inject CA bundles
- Runs as a Deployment in Kubernetes
- Exposes a webhook via a Service

## Project Structure
- `main.go`: Entry point for the webhook server
- `admission/`: Admission controller logic
- `mutation/`: Mutation logic for injecting CA bundles
- `deployment.yaml`: Kubernetes Deployment manifest
- `service.yaml`: Kubernetes Service manifest
- `mutatingwebhook.yaml`: MutatingWebhookConfiguration manifest

## Usage
1. Build and push the Docker image:
   ```sh
   docker build -t <your-repo>/ca-bundle-injector:<tag> .
   docker push <your-repo>/ca-bundle-injector:<tag>
   ```
2. Deploy to Kubernetes:
   ```sh
   kubectl apply -f deployment.yaml
   kubectl apply -f service.yaml
   kubectl apply -f mutatingwebhook.yaml
   ```
   Replace `<CA_BUNDLE_PLACEHOLDER>` in `mutatingwebhook.yaml` with your actual CA bundle (base64 encoded).

## Example
See `sample-pod.json` for a sample pod manifest.

## License
MIT
