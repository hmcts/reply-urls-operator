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

The Operator needs the permissions below to work properly.

| resources     | verbs            |
|---------------|------------------|
| replyurlsyncs | get, list, watch |
| ingresses     | get, list, watch |

All the RBAC files can be found in the `config/rbac` folder. They are created using markers in the Operators Go code, markers for RBAC can be found in `controllers/ingress_controller.go` and look similar to below.

```go
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
```

[More information on RBAC markers](https://book.kubebuilder.io/reference/markers/rbac.html) 

## Running the Operator

Before deploying anything you will need an Azure App Registration that will have its Reply URLs updated by the Operator and either the same App Registration or a separate one that has the right permissions to update the App Registration's Reply URLs.

[Instructions on creating an Azure App Registration](https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app)

You will need to take note of the Object ID of the App Registration that will be managed by the Operator and the Client/Application ID, Client Secret and Tenant ID of the App Registration that will be used to Authenticate.

Once you have created the app registration you will need to give it the correct permissions. This can be done by clicking on the API permissions tab in the Azure portal whilst viewing the app registration. You then need to click on Add a permission and then Microsoft Graph, click on Application permissions and search for Application. Add the Application.ReadWrite.All permission and click Add permissions, then click the Grant admin consent button.

#### Configuring the ReplyURLSync config
To configure the sync config so the Operator knows how to Authenticate with Azure, which App Registration to update and what Ingresses and URLs it should be managing, you will need to configure a `ReplyURLSync` custom resource. Currently, there are 6 fields available to configure the sync:

#### Deploying the Operator to a cluster

The commands below will deploy the Custom Resource Definitions (CRDs), RBAC, the Operator, the replyURLSync and the example Ingress. The Operator should be fully operational after these commands have been executed.

**Note:** The Makefile has the ability to build and push the container image manually, but there are GitHub Workflows in the `.github/workflows` folder that automate the process.

1. Build and push the controller images to a container registry (this step can be skipped if you already have an image built and pushed):
   
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

3. Create reply-urls-operator secret
   Currently the App Registration's Client Secret for the Operator is set as an environment variable. You will need to create a secret called `reply-urls-operator` with a data object called `azure-client-secret`.

   Command to create the secret manually:
   ```sh
    kubectl create secret -n admin generic reply-urls-operator --from-literal azure-client-secret=<client_secret>
   ```

4. Update the ReplyURLSync config:

   There is a sample ReplyURLSync config in `config/samples/reply-url-sync-example.yaml` which can be update if needs be.

   * ingressClassFilter: Name of the Ingress Class that you want to watch e.g. "traefik"
   * domainFilter (optional): Regex of the domain of the Ingress Hosts you want to manage e.g. ".*.sandbox.platform.hmcts.net". Defaults to match all ".*"
   * replyURLFilter (optional): Regex of the reply URLs you want to manage e.g. ".*.sandbox.platform.hmcts.net". This can be set to something different to the domainFilter if you would only like delete certain reply URLS from the app registration. Defaults to ".*"
   * clientID: Client ID of the app registration you are authenticating with.
   * objectID: Client ID of the app registration you want to sync ReplyURLS with.
   * tenantID: Tenant ID of the app registration you are authenticating with.

   Example yaml file configuration for the ReplyURLSync:

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

5. Install CRDs, RBAC and the Operator:

   ```sh
   kustomize build config/default | kubectl apply -f -
   ```

6. Install ReplyURLSync custom resource and example Ingress:

   ```sh
   kustomize build config/samples | kubectl apply -f -
   ```

The Reply URLs operator should now be running and managing your app registrations Reply URLs.

Move onto [testing the functionality of the operator](#Testing-the-functionality-of-the-Operator)

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

###### Delete Secret
```sh
kubectl create secret -n admin generic reply-urls-operator --from-literal azure-client-secret=<client_secret>
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

Move onto the next section to test that the operator is working correctly.

## Testing the functionality of the Operator

### Viewing the Operator logs
If you are running the operator on a cluster and not locally follow the steps below to view the logs:
<details>
    <summary>Running on a cluster</summary>
   
   Get the name of the operator pod: 
   ```shell
   kubectl get pods -n admin -l control-plane=reply-urls-operator
   ```

   View the logs of the pod (replace <pod-name> with the name of the pod from the previous step):
   ```shell
   kubectl logs -n admin -f <pod-name> 
   ```

</details>

If you can running locally use the steps below:
<details>
    <summary>Running locally</summary>

   If you have already followed the steps in [Test out the operator locally](#Test-out-the-operator-locally) and ran `main.go`, you should be viewing the logs in your terminal already.

</details>

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

To test that the operator is actually working and updating the App Registration's Reply URLs, you can use the Azure Portal to check that the App Registration's list of Reply URLs contains `https://reply-urls-example-1.local.platform.hmcts.net/oauth-proxy/callback` and `https://reply-urls-example-2.local.platform.hmcts.net/oauth-proxy/callback`.

You can also run the az command below to view the URLs (replace <object-id> with the object id of the app registration you are updating):
```sh
az ad app show --id <object-id> --query 'web.redirectUris'
```

### Testing the operator works
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

