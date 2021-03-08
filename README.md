# Google Cloud Endpoints in GKE with Container-native Load Balancing

In this experiment, we extend the [Getting started with Cloud Endpoints for GKE with ESP](https://cloud.google.com/endpoints/docs/openapi/get-started-kubernetes-engine) documentation guide to
provide an example of how to configure HTTPS between the LB and the ESP and also
how to use Container-native Load Balancing. Additionally, the Open API configuration provides
examples of how to configure different types of `securityDefinitions` using Cloud
Endpoints. These `securityDefinitions` can be tested with [this client](./client/main.go).

The code in this repo is a fork of [Google Cloud golang-samples/endpoints/getting-started](github.com/GoogleCloudPlatform/golang-samples/endpoints/getting-started).

## Prepare the environment

Export environment variables with your GCP Project ID, GCP Region and name of the
GKE cluster to be created.

```bash
export GCP_PROJECT=your_project_id
export GCP_REGION=us-central1
export GKE_CLUSTER=le-cluster
```

```bash
gcloud config set project $GCP_PROJECT
```

Create the GKE cluster.

```bash
gcloud beta container clusters create $GKE_CLUSTER \
  --release-channel regular \
  --enable-ip-alias \
  --region $GCP_REGION
```

Configure your kubectl to point to the newly created cluster.

```bash
gcloud container clusters get-credentials $GKE_CLUSTER --region $GCP_REGION
```

Create an External IP that will be used in the Ingress later.

```bash
gcloud compute addresses create esp-echo --global
```

Store the IP address in a variable

```bash
INGRESS_IP=$(gcloud compute addresses describe esp-echo --global --format json | jq -r .address)
```

## Let's deploy the Cloud Endpoints configuration

Inspect the [openapi.yaml](./openapi.yaml) file and update the attribute `host`
with the name of your `GCP_PROJECT`.

```yaml
host: "echo-api.endpoints.YOUR_PROJECT_ID.cloud.goog"
```

```bash
sed -i.original "s/YOUR_PROJECT_ID/${GCP_PROJECT}/g" openapi.yaml
```

The value of the attribute host will be the name of the Endpoints service.

Now upload the configuration and create a managed service.

```bash
gcloud endpoints services deploy openapi.yaml
```

Check the Google services enabled in your project and enable the necessary
services if they aren't enabled.

```bash
gcloud services list

gcloud services enable servicemanagement.googleapis.com
gcloud services enable servicecontrol.googleapis.com
gcloud services enable endpoints.googleapis.com
gcloud services enable echo-api.endpoints."${GCP_PROJECT}".cloud.goog

```

## Deploy the Kubernetes config

Deploy the Kubernetes config using ESP with HTTP.

Adjust the name of your Endpoints service in the ESP configuration.

```yaml
      - name: esp
        image: gcr.io/endpoints-release/endpoints-runtime:1
        args: [
            "--http_port", "8081",
          "--backend", "127.0.0.1:8080",
          "--service", "echo-api.endpoints.YOUR_PROJECT_ID.cloud.goog",
          "--rollout_strategy", "managed",
        ]
```

```bash
sed -i.original "s/YOUR_PROJECT_ID/${GCP_PROJECT}/g" .kube-http.yaml
```

In the ESP configuration above, the `--rollout_strategy=managed` option
configures ESP to use the latest deployed service configuration. When you
specify this option, up to 5 minutes after you deploy a new service
configuration, ESP detects the change and automatically begins using it.
Alternatively, you can use the `--version` / `-v` flag to have more control of
which configuration id / version of your Cloud Endpoints the ESP is using. [See more details about the ESP options](https://cloud.google.com/endpoints/docs/openapi/specify-proxy-startup-options).

Deploy the Kubernetes config

```bash
kubectl apply -f .kube-http.yaml
```

Check if the pod was properly initialized

```bash
kubectl get po
```

Check if the ingress has an external IP assigned. This IP is the same IP we
defined earlier.

It will take several minutes until the Ingress becomes available. Wait until the
backend service reports `HEALTHY`.

```bash
watch "kubectl get ingress esp-echo \
  -o jsonpath='{.metadata.annotations.ingress\.kubernetes\.io/backends}'"
```

Use the following commands to observe how the GCP Backend Service and Health Check
get configured based on your Ingress, Service and Pod configuration.

```bash
BACKEND_SERVICE=$(kubectl get ingress esp-echo \
  -o jsonpath='{.metadata.annotations.ingress\.kubernetes\.io/backends}' | jq -r keys[0]

gcloud compute backend-services describe $BACKEND_SERVICE --global

gcloud compute health-checks describe $BACKEND_SERVICE --global
```

Finally, check your service.

```bash
curl -v http://"${INGRESS_IP}"/healthz


curl --request POST \
   --header "content-type:application/json" \
   --data '{"message":"hello world"}' \
   "http://${EXTERNAL_IP}/echo"
```

Execute the same steps with the [.kube-https.yaml](.kube-https.yaml) configuration.
Notice that you test from the `EXTERNAL_IP` still using HTTP. This is because
when you configure the ESP container with HTTPS you are using TLS only for traffic
from `LB -> ESP`.

## Troubleshooting

The main problems you may have in this setup are related to the Ingress configuration. You can check the [Google Cloud Troubleshooting page](https://cloud.google.com/kubernetes-engine/docs/how-to/container-native-load-balancing#troubleshooting).

If the Ingress health check for the https example is consistently showing Unhealthy you may need to create a firewall rule to allow the Google LB to reach the backend.

```bash
gcloud compute firewall-rules create fw-allow-health-check-and-proxy \
  --network=NETWORK_NAME \
  --action=allow \
  --direction=ingress \
  --target-tags=GKE_NODE_NETWORK_TAGS \
  --source-ranges=130.211.0.0/22,35.191.0.0/16 \
  --rules=tcp:8443
 ```

## References

These are some resources that helped me during this experiment:

https://cloud.google.com/endpoints/docs/openapi/get-started-kubernetes-engine
https://cloud.google.com/endpoints/docs/openapi/specify-proxy-startup-options
https://cloud.google.com/endpoints/docs/openapi/configure-endpoints
https://cloud.google.com/kubernetes-engine/docs/concepts/ingress-xlb#https_tls_between_load_balancer_and_your_application
https://cloud.google.com/kubernetes-engine/docs/concepts/container-native-load-balancing
https://cloud.google.com/kubernetes-engine/docs/how-to/container-native-load-balancing#troubleshooting
