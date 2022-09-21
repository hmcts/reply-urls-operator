# reply-urls-operator
A k8s Operator that keeps Ingress hosts in sync with the Redirect URLs associated with an Azure App Registration.

## Table of Contents
**[Description](#Description)**<br>
**[How the Operator works](#How-the-Operator-works)**<br>
**[Deploying the Operator to a cluster](#Deploying-the-Operator-to-a-cluster)**<br>
**[Test out the operator locally](#Test-out-the-operator-locally)**<br>
**[GitHub Workflows](#GitHub-Workflows)**<br>
**[Modifying the API definitions](#Modifying-the-API-definitions)**<br>
**[License](#License)**<br>

## Description
The Reply URLs Operator was created to automate the manual process of updating and removing Redirect URLs when applications are created and removed from AKS clusters. This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

## Getting Started

### How the Operator works
1. Once running, the operator will watch for any Create, Update or Delete events associated with Ingress resources on the cluster it's running on. If you're running the controller locally it will be whichever cluster your kubectl config is pointing to.
2. When an event occurs on an Ingress on the cluster the operator will act upon that event, depending on the type of event.
   * **Create/Update:** The Ingress the event is created, will be filtered according the to configuration set in the ReplyURLSync config and will be synced, if it matches the filter and doesn't exist in the list of Reply URLs it will be added.
   * **Delete:** The list of Reply URLs on the app registration will be checked and if there are any URLs that do not have an Ingress associated with it, the operator will remove the URL from the App Registration. You can change this behaviour by setting `replyURLFilter` to a regex of the URLs the operator should manage, ignoring anything that doesn't match.
3. The operator also reconciles every 5 minutes against all Ingresses on the cluster.

### Azure permissions and RBAC
Permissions needed for the operator to run properly are as follows.

#### Azure permissions
The Operator needs to be able to read and write to the App Registrations and can be added via the `API Permissions` tab on the App Registration itself.

* API Permissions: `Application.ReadWrite.All` (Type: Application)

**Note:** If you are running the cluster locally you can use the az cli to authenticate with Azure as long as your user is able to Read and Write to the App Registration that the Operator is configured to manage.

#### Cluster RBAC
All the RBAC files can be found in the `config/rbac` folder. They are created using markers in the Operators Go code, markers for RBAC can be found in `controllers/ingress_controller.go` and look similar to below.

```go
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
```

[More information on RBAC markers](https://book.kubebuilder.io/reference/markers/rbac.html) 

The Operator needs the permissions below to work properly.

| resources     | verbs            |
|---------------|------------------|
| replyurlsyncs | get, list, watch |
| ingresses     | get, list, watch |

## Running the Operator

Before deploying anything you will need an Azure App Registration that will have its Reply URLs updated by the Operator and either the same App Registration or a separate one that has the right permissions to update the App Registration's Reply URLs.

[Instructions on creating an Azure App Registration](https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app)

You will need to take note of the Object ID of the App Registration that will be managed by the Operator and the Client/Application ID, Client Secret and Tenant ID of the App Registration that will be used to Authenticate.

#### Configuring the sync config
To configure the Sync config so the Operator knows how to Authenticate with Azure, which App Registration to update and what Ingresses and URLs it should be managing, you will need to configure a `ReplyURLSync` custom resource. Currently, there are 6 fields available to configure the sync:

1. ingressClassFilter: Name of the Ingress Class that you want to watch e.g. "traefik"
2. domainFilter (optional): Regex of the domain of the Ingress Hosts you want to manage e.g. ".*.sandbox.platform.hmcts.net". Defaults to match all ".*"
3. replyURLFilter (optional): Regex of the reply URLs you want to manage e.g. ".*.sandbox.platform.hmcts.net". This can be set to something different to the domainFilter if you would only like delete certain reply URLS from the app registration. Defaults to ".*"
4. clientID: Client ID of the app registration you are authenticating with.
5. objectID: Client ID of the app registration you want to sync ReplyURLS with.
6. tenantID: Tenant ID of the app registration you are authenticating with.

Example yaml file configuration for a ReplyURLSync:

```yaml
apiVersion: appregistrations.azure.hmcts.net/v1alpha1
kind: ReplyURLSync
metadata:
  name: replyurlsync-sample
spec:
  ingressClassFilter: traefik
  domainFilter: .*.sandbox.platform.hmcts.net
  clientID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  objectID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  tenantID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```
The above yaml will watch for any events on Ingresses that have the Ingress Class Name of `traefik` and a host that has a suffix of `.sandbox.platform.hmcts.net`.
**Note:** We're not setting a value for the replyURLFilter so the operator will manage the entire list of Reply URLs it finds associated with the App Registration. This means that if someone or something has added a Reply URL manually or another operator is also adding to the list, no matter what the domain, this operator will delete any URL it doesn't find associated to an Ingress on the cluster it is deployed to.

#### Deploying the Operator to a cluster

The commands below will deploy the Custom Resource Definitions (CRDs), RBAC, the Operator, the replyURLSync and the example Ingress. The Operator should be fully operational after these commands have been executed.

**Note:** The Makefile has the ability to build and push the container image manually, but there are GitHub Workflows in the `.github/workflows` folder that automate the process.

1. Deploy the controller to the cluster with the image specified by `IMG` (this step can be skipped if you already have an image built and pushed):
   
   Replace `<some-registry>` with the container registry you would like to push the image to and `<tag>` with the tag to identify the image.

   ```sh
   make docker-build docker-push IMG=<some-registry>/reply-urls-operator:<tag>
   ```

2. Update the container image for the manager deployment:

   Before deploying the Reply URLs Operator you will need to update the image being declared in the deployment file `config/manager/manager.yaml`. Update the `manager` container in the containers section of the file with the image you have built. 
   
   That section should look similar to the snippet below.

   ```yaml
         containers:
         - name: manager
           command:
           - /manager
           image: sdshmctspublic.azurecr.io/reply-urls-operator:prod-c4620b7-20220905093200
           imagePullPolicy: Always
   ```

3. Install CRDs, RBAC and the Operator:

   ```sh
   kustomize build config/default | kubectl apply -f -
   ```

4. Install ReplyURLSync custom resource and example Ingress:

   ```sh
   kustomize build config/samples | kubectl apply -f -
   ```

The Reply URLs operator should now be running and managing your app registrations Reply URLs.

##### Cleanup

###### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

###### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Test out the operator locally

First of all we need to deploy the CRDs and example resources so the Operator knows which Ingresses to watch for and which Reply URLs to manage.

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.

**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows), so make sure you're pointing to the right cluster before going any further.

Once you're happy that your kubectl context is correct you can install the CRDs and resources.

Install CRDs
```shell
 kustomize build config/crd | kubectl apply -f -
```

Create ReplyURLSync and Ingress resources
```shell
kustomize build config/samples | kubectl apply -f -
```

Now you have the necessary resources in place, you should be able to run the Operator.
```shell
go run main.go
```

You should see something similar to below:

```json lines
1.663325436660029e+09   INFO    controller-runtime.metrics      Metrics server is starting to listen    {"addr": ":8080"}
1.6633254366609678e+09  INFO    setup   starting manager
1.663325436661598e+09   INFO    Starting server {"path": "/metrics", "kind": "metrics", "addr": "[::]:8080"}
1.663325436661598e+09   INFO    Starting server {"kind": "health probe", "addr": "[::]:8081"}
1.663325436863172e+09   INFO    Starting EventSource    {"controller": "ingress", "controllerGroup": "networking.k8s.io", "controllerKind": "Ingress", "source": "kind source: *v1.Ingress"}
1.663325436863437e+09   INFO    Starting Controller     {"controller": "ingress", "controllerGroup": "networking.k8s.io", "controllerKind": "Ingress"}
1.663325436863749e+09   INFO    Starting workers        {"controller": "ingress", "controllerGroup": "networking.k8s.io", "controllerKind": "Ingress", "worker count": 1}
1.663325444372884e+09   INFO    Reply URL added {"URL": "https://reply-urls-example-2.local.platform.hmcts.net/oauth-proxy/callback", "object id": "b40e709c-24e0-4e1f-8e79-65268a4c24fe", "ingressClassName": "traefik"}
1.6633254472403562e+09  INFO    Reply URL added {"URL": "https://reply-urls-example-1.local.platform.hmcts.net/oauth-proxy/callback", "object id": "b40e709c-24e0-4e1f-8e79-65268a4c24fe", "ingressClassName": "traefik"}

```

You'll notice that in the logs it states that 2 URLs have been added to the list of Reply URLs. The Operator has picked up the hosts from the Ingresses we created and as they both meet the IngressClassName and Domain filters it has added them to the list. If you're using an already existing Dev cluster there will already be Ingresses on that cluster, but they won't match the filters and therefore will not be added to the App Registration's Reply URLs list.

Open up another terminal at the root of the reply-url-operator repo and delete the Ingresses from the cluster.
```shell
kubectl delete -f 'config/samples/ingress-*'
```

In your original terminal, where you are running the operator, You should now see two more lines in the log detailing the removal of the URls as the Ingresses no longer exist on the cluster, similar to below:

```json lines
1.6633259693645282e+09  INFO    Reply URLs removed      {"URLs": ["https://reply-urls-example-2.local.platform.hmcts.net/oauth-proxy/callback"], "object id": "b40e709c-24e0-4e1f-8e79-65268a4c24fe", "ingressClassName": "traefik"}
1.663325972135824e+09   INFO    Reply URLs removed      {"URLs": ["https://reply-urls-example-1.local.platform.hmcts.net/oauth-proxy/callback"], "object id": "b40e709c-24e0-4e1f-8e79-65268a4c24fe", "ingressClassName": "traefik"}
```

### Clean up

Delete the ReplyURL resource
```sh
kubectl delete -f config/samples/reply-url-sync-example.yaml
```

Delete the ReplyURLSync CRD 
```sh
kustomize build config/crd | kubectl apply -f -
```

Press `CTRL+C` to Stop the Operator.

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## GitHub Workflows

### Build and test workflow
This workflow builds and tests the Operator code. When the tests are successful, the workflow builds a container image and pushes the image to a container registry.

### Promote workflow
Promotes the built image when the PR is closed and approved.

### Tag code workflow
Tags the code base when the config directory gets updated.

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

