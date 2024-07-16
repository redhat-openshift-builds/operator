# operator
OpenShift Builds operator provides the API to manage Shipwright Build and Shared Resource CSI Driver.

## Description
OpenShift Builds operator deploys and manages the following components
- Shipwright Components (pending implementation)
- Shared Resource CSI Driver (pending implementation)

## Getting Started

### Prerequisites
- go version v1.20.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Deploy Operator (standalone)

#### Step 1: Build and push your operator image

Use the IMAGE_TAG_BASE variable to change the operator image's target repostiory.
This should be a proper image name and not end with trailing slashes or special characters.

```sh
$ make docker-build docker-push IMAGE_TAG_BASE=quay.io/myusername/rh-openshift-builds/operator
```

**NOTE:** You must have permission to push to the container registry referenced in `IMAGE_TAG_BASE`.
Your cluster must also have permission to pull images from the referenced container registry.

#### Step 2: Deploy CRDs and Operator**Install the CRDs into the cluster:**

For this step, you must have the equivalent of "cluster admin" privileges on the cluster.

First, deploy custom resource definitions (CRDs) for the operator by running:

```sh
$ make install
```
Next, deploy the operator using the same `IMAGE_TAG_BASE` variable as above.

```sh
make deploy IMAGE_TAG_BASE=quay.io/myusername/rh-openshift-builds/operator
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

### Deploy with OLM

Red Hat operators are designed to be managed by Operator Lifecycle Manager (OLM) and deployed
through the `OperatorHub` section of the OpenShift web console. To deploy with OpenShift and OLM:

1. Build your operator and push it to a container registry (step 1 above).
2. Build the operator bundle and push it to a container registry, by running the following `make
   commands:

   ```sh
   $ make bundle IMAGE_TAG_BASE=quay.io/myusername/rh-openshift-builds/operator
   $ make bundle-build bundle-push IMAGE_TAG_BASE=quay.io/myusername/rh-openshift-builds/operator
   ```

3. Build and push the operator catalog

   ```sh
   $ make catalog-fbc-build IMAGE_TAG_BASE=quay.io/myusername/rh-openshift-builds/operator
   $ make catalog-push IMAGE_TAG_BASE=quay.io/myusername/rh-openshift-builds/operator
   ```

4. Deploy the catalog as a `CatalogSource`

   ```sh
   $ make catalog-deploy IMAGE_TAG_BASE=quay.io/myusername/rh-openshift-builds/operator
   ```

5. In the OpenShift web console, navigate to "OperatorHub" in the Administrator view. You should be
   able to filter for operators in the "Test Candidate Operators" catalog and install the Builds for OpenShift operator from there.


## Contributing
TBD

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

