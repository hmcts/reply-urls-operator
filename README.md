# reply-urls-operator
An Operator which converts configured Kubernetes Ingress hosts to valid Azure App Registration Reply URLs, updates the App Registration and keeps them in sync.  

### Table of Contents
**[Description](#Description)**<br>
**[How the Operator works](#How-the-Operator-works)**<br>
**[Running on a cluster](#Running on a cluster)**<br>
**[Test out the operator locally](#Test out the operator locally)**<br>
**[GitHub Workflows](#GitHub Workflows)**<br>
**[Modifying the API definitions](#Modifying the API definitions)**<br>
**[License](#License)**<br>

## Description
This Operator was created to automate the manual process of updating and removing Redirect URLs when applications are deployed and removed. 

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

## Getting Started

### How the Operator works
1. Once running, the operator will watch for any Create, Update or Delete events associated with Ingress resources on the cluster it's running on. If you're running the controller locally it will be whichever cluster your kubectl config is pointing to.
2. When an event occurs on one of the Ingresses on the cluster the operator will act upon that event, depending on the type of event.
   * **Create/Update:** The ingress the event is targeting will be synced, if it doesn't exist in the list of Reply URLs it will be added.
   * **Delete:** The list of Reply URLs on the app registration will be checked and if there are any URLs that do not have an ingress associated with it, the operator will remove the URL for the App Registration. You can change this behaviour by setting `replyURLFilter` to a regex of the URLs the operator should manage, ignoring anything that doesn't match.
3. The operator will also reconcile every 5 minutes against all ingresses on the cluster.


### Permissions
Permissions needed for the operator to run properly are as follows.

#### Operator App Registration
* API Permissions: `Application.ReadWrite.All` (Type: Application)


### Running on a cluster

#### Configuring the sync config
Before deploying anything you will need an App Registration to monitor and an App Registration with the correct permissions to update the App Reg you are monitoring. We will also need to configure the Sync config for so the Operator knows how to Authenticate with Azure, which App Registration to update and what Ingresses and URLs it should be monitoring and managing.  

Currently, there are 6 fields available to configure the sync.

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

2. Install CRDs, RBAC and the Operator:

```sh
kustomize build config/default | kubectl apply -f -
```

3. Install ReplyURLSync custom resource and example Ingress:

```sh
kustomize build config/samples | kubectl apply -f -
```

The Reply URLs operator should now be running and managing your app registrations Reply URLs.

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```



### Test out the operator locally

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).


1. Install the CRDs onto the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)


## GitHub Workflows

TODO

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

