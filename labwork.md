# Labwork: Kubernetes Hello

## Step 1: Containerize an HTTP server

The `main.go` file contains code for an HTTP server. Compile this code and
package the server as a Docker container image:

```bash
docker build --tag=busser/k8s-hello:latest .
docker images
```

Display the server's usage:

```bash
docker run --rm busser/k8s-hello:latest --help
```

Run the HTTP server to try out some its features:

```bash
docker run --detach --publish=8080:80 --name=k8s-hello busser/k8s-hello:latest
```

Send an HTTP request to the server:

```bash
curl localhost:8080/
```

The server's response contains the requested URL, as well as some information
about the server's Kubernetes deployment. The server is not currently running
inside a Kubernetes cluster, so the information displayed now can be ignored.

The server exposes a simple healthcheck that returns an HTTP error code of 200
when it is healthy:

```bash
curl -I localhost:8080/healthz
```

Sometimes a web server can enter an unhealthy state: it can still serve
requests but should ideally be restarted. This server's health can be remotely
deteriorated; the server will then respond to a healthcheck with an error code
of 500:

```bash
curl localhost:8080/damage
curl -I localhost:8080/healthz
```

Fix the server so that it will once again report that it is healthy:

```bash
curl localhost:8080/heal
curl -I localhost:8080/healthz
```

Sometimes a web server can crash. Make this server crash:

```bash
curl localhost:8080/kill
```

The server is no longer running and will not accept connections. Notice that
its container has stopped:

```bash
docker ps --all
```

Delete the container and move on to the next part of the lab:

```bash
docker rm k8s-hello
```

## Step 2: Push the container image to a registry

Log into the registry:

```bash
docker login docker.io
```

Push the image to the registry:

```bash
docker push busser/k8s-hello:latest
```

If you are using a public registry - i.e one that does not require
authentication to pull images - you can move on to step 3 of this lab.

If you are using a private registry, you need to provide your credentials to
Kubernetes in order to pull iamges from the registry.

There are two ways of providing your credentials to Kubernetes. Both involve
creating a secret. The first option uses the credentials from earlier. The
second provides the option of providing different credentials. Pick an option
and, once you have completed it, move on to step 3.

### Option A: Use the same credentials as before

The credentials you provided during to the `docker login` command are
stored in `~/.docker/config.json`. Create a Kubernetes sercret with the
contents of that file:

```bash
kubectl create secret generic docker.io-credentials \
  --from-file=.dockerconfigjson=$HOME/.docker/config.json \
  --type=kubernetes.io/dockerconfigjson
```

Examine the secret:

```bash
kubectl get secret docker.io-credentials --output=yaml
```

Confirm that the contents of the secret match the contents of
`~/.docker/config.json`:

```bash
kubectl get secret docker.io-credentials \
  --output="jsonpath={.data.\.dockerconfigjson}" | base64 --decode
```

### Option B: Use any credentials

Store the username, password, and email address you wish to use inside shell
variables:

```bash
REGISTRY_USERNAME=busser
read -p "Your password: " -s REGISTRY_PASSWORD
EMAIL_ADDRESS=docker@busser.me
```

Create a Kubernetes secret containing the credentials:

```bash
kubectl create secret docker-registry docker.io-credentials \
  --docker-server=docker.io \
  --docker-username=$REGISTRY_USERNAME \
  --docker-password=$REGISTRY_PASSWORD \
  --docker-email=$EMAIL_ADDRESS
```

Examine the secret:

```bash
kubectl get secret docker.io-credentials --output=yaml
```

See what format the credentials were stored in:

```bash
kubectl get secret docker.io-credentials \
  --output="jsonpath={.data.\.dockerconfigjson}" | base64 --decode
```

## Step 3: Deploy an instance of the HTTP server

Write a `k8s-hello.yaml`file:

```yaml
---
apiVersion: v1
kind: Pod
metadata:
  name: hello-pod
spec:
  containers:
  - name: hello-container
    image: busser/k8s-hello:latest
    ports:
    - containerPort: 80
  imagePullSecrets:
  - name: docker.io-credentials
```

> If you are using a public registry to store the container image, do not
> write the ìmagePullSecrets` section of the pod spec.

Create your pod (this will fail):

```bash
kubectl apply --filename=k8s-hello.yaml
```

The reason Kubernetes refuses to create this pod is that its specifications
do not specify any resource requests or limits.

Resource requests specify the amount of resources the application absolutely
needs to function. The Kubernetes scheduler will use this information to decide
which node will run the pod.

Resource limits specify how many resources the container should be allowed
to consume. The container will not be able to consume more. If it tries to,
it will be killed.

Add resource information to `k8s-hello.yaml`:

```yaml
...
spec:
  containers:
  - ...
    resources:
      requests:
        cpu: 50m
        memory: 50Mi
      limits:
        cpu: 100m
        memory: 100Mi
  ...
```

Create your pod:

```bash
kubectl apply -f k8s-hello.yaml
kubectl get pods
```

Kubernetes has deployed the application. Get all the information Kubernetes
has about your pod:

```bash
kubectl get pod hello-pod --output=yaml
```

Map a port from your environment to the pod's:

```bash
kubectl port-forward hello-pod 8080:80 >/dev/null &
curl localhost:8080/
```

## Step 4: Create a service to send requests to the pod

> Above, `kubectl port-forward` was started in the background.
> If you want to get it's process ID, use `jobs -p`.

To avoid needing to forward a port, add a Kubernetes NodePort service in
`k8s-hello.yaml`:

```yaml
...

---
apiVersion: v1
kind: Service
metadata:
  name: hello-service
spec:
  type: NodePort
  selector:
    app: hello
  ports:
  - port: 80
```

Add a label to the pod the service should redirect to:

```yaml
...
metadata:
  ...
  labels:
    app: hello
...
```

Update everything:

```bash
kill %1 # Kills 'kubectl port-forward' command.
kubectl delete pod hello-pod
kubectl apply -f k8s-hello.yaml
```

Find which port is mapped to the application's port 80:

```bash
kubectl get services
# Note the port mapping, e.g. 80:30731/TCP
```

Look at your `kubectl` configuration file to know which cluster node you
can connect to:

```bash
kubectl config view | grep server
# This is the IP address of the Kubernetes master node, e.g. 192.168.99.101
```

Contact the application through the Kubernetes service:

```bash
curl 192.168.99.101:30731/
```

## Step 5: Create a deployment to manage the pod

To avoid needing to delete the pod with every update, edit `k8s-hello.yaml` to
replace the pod with a deployment:

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-deployment
spec:
  selector:
    matchLabels:
      app: hello
  template:
    # Put your pod metadata and spec here.
    # Remove the pod's name; Kubernetes will generate a name for it.
...
```

Delete the pod and replace it with the deployment:

```bash
kubectl delete pod hello-pod
kubectl apply -f k8s-hello.yaml
kubectl get pods
```

The service still listens on the same port:

```bash
curl 192.168.99.101:30731/
```

## Step 6: Configure the HTTP server

Update `k8s-hello.yaml` so the application knows where it is in the cluster:

```yaml
...
spec:
  template:
    spec:
      containers:
      - ...
        args: ["-namespace", "$(K8S_NAMESPACE)", "-node", "$(K8S_NODE)", "-pod", "$(K8S_POD)"]
        env:
        - name: K8S_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: K8S_NODE
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: K8S_POD
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
      ...
...
```

Update the pod with more features:

```bash
kubectl apply -f k8s-hello.yaml
curl 192.168.99.101:30731/
```

## Step 7: Add a liveness probe

If your application crashes, Kubernetes will restart it:

```bash
curl 192.168.99.101:30731/kill
kubectl get pods
```

Kubernetes will automatically restart the application. The more times
the application crashes, the longer Kubernetes will wait before
restarting it. This is called "backing off from a crash loop'.

However, if we "damage" our application, Kubernetes will not realise that the
pod is unhealthy. It will not restart it:

```bash
curl 192.168.99.101:30731/damage
kubectl get pods
```

Update `k8s-hello.yaml` to let Kubernetes know the application is not in a good
state and should be restarted. This is called a liveness probe:

```yaml
...
spec:
  template:
    spec:
      containers:
      - ...
        livenessProbe:
          httpGet:
            path: /healthz
            port: 80
          periodSeconds: 1
...
```

Update the deployment to perform a rolling update:

```bash
kubectl apply -f k8s-hello.yaml
kubectl get pods
```

Now, if you damage the application, Kubernetes will restart it. Notice that the
pod's "restarts" counter gets incremented:

```bash
curl 192.168.99.101:30731/damage
kubectl get pods
```

## Step 8: Add a readiness probe

If an application takes some time to start, and should not receive any requests
until it reports itself as ready, Kubernetes should know to wait.

Update `k8s-hello.yaml` to add a readiness probe:

```yaml
...
spec:
  template:
    spec:
      containers:
      - ...
        readinessProbe:
          httpGet:
            path: /healthz
            port: 80
          periodSeconds: 1
...
```

Configure the HTTP server to wait some time before starting:

```yaml
...
spec:
  template:
    spec:
      containers:
      - ...
        args: ["-init", "5s", ...]
        ...
        livenessProbe:
          ...
          initialDelaySeconds: 5
        ...
      ...
...
```

Now update the deployment:

```bash
kubectl apply -f k8s-hello.yaml
```

Notice that the pod may be running but not ready. Kubernetes will wait for the
new pod to be ready before it terminates the old one. This guarantees that
incoming requests can always be handled. If no pods are ready, individual pods
will still accept connections but the service will not.

```bash
kubectl get pods
curl 192.168.99.101:30731/
```

## Step 9: Replicate the HTTP server

Since the application may not always be ready and/or healthy, it is best to
deploy multiple replicas of the application. Update `k8s-hello.yaml` to specify
a number of pod replicas in the deployment:

```yaml
...
spec:
  replicas: 20
  ...
...
```

Update the deployment:

```bash
kubectl apply -f k8s-hello.yaml
```

Kubernetes will start any missing pods:

```bash
kubectl get pods
```

The service will automatically perform load-balancing on all instances of the
application. You can see this when you send requests:

```bash
curl 192.168.99.101:30731/
curl 192.168.99.101:30731/
curl 192.168.99.101:30731/
```

If you tell several instances to kill themselves, the service will not route
any requests to them until they are healthy again:

```bash
for i in {1..9}; do
  curl 192.168.99.101:30731/kill
done

kubectl get pods
curl 192.168.99.101:30731/
```

## Step 10: Add a termination grace period

When stopping a pod, the application may need time to finish what it is doing
before shutting down. Kubernetes will first signal to the application that it
should turn off. If the pod does not shut down on its own after a certain
delay, Kubernetes will forcefully kill it.

Update the deployment in `k8s-hello.yaml` so that Kubernetes waits for at most
10 seconds:

```yaml
...
spec:
  ...
  terminationGracePeriodSeconds: 10
...
```

This avoids applications that fail to shut down gracefully poluting the
Kubernetes cluster and blocking system resources.

## Conclusion

This lab is now completed. You have deployed a highly-available HTTP server
inside a Kubernetes cluster.

If you wish to delete your deployment and service:

```bash
kubectl delete --filename=k8s-hello.yaml
```

In its final state, the `k8s-hello.yaml` should look like this:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-deployment
spec:
  replicas: 20
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
      - name: hello-container
        image: busser/k8s-hello:latest
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 50m
            memory: 50Mi
          limits:
            cpu: 100m
            memory: 100Mi
        args: ["-init", "5s", "-namespace", "$(K8S_NAMESPACE)", "-node", "$(K8S_NODE)", "-pod", "$(K8S_POD)"]
        env:
        - name: K8S_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: K8S_NODE
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: K8S_POD
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        livenessProbe:
          httpGet:
            path: /healthz
            port: 80
          periodSeconds: 1
          initialDelaySeconds: 5
        readinessProbe:
          httpGet:
            path: /healthz
            port: 80
          periodSeconds: 1
      imagePullSecrets:
      - name: docker.io-credentials

---
apiVersion: v1
kind: Service
metadata:
  name: hello-service
spec:
  selector:
    app: hello
  ports:
  - port: 80
  type: NodePort
```
