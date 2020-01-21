# Debugging e2etests
Sometimes e2e tests fail and we need to figure out why.
These tests will normally create a local kubernetes cluster where our operators will be installed and Custom Resources
will be created to spin up tenant clusters on the tested provider.

If we want to debug what happened, the first thing we would need to do would be to re-run the failed job in CircleCI
enabling SSH so we can later connect to the CircleCI worker running the job. Click on the following button to do that:

![](rerun.png)

The job will be re executed, but this time there is a new step that let's you connect to the worker using SSH.

![](ssh.png)

## Inspecting control plane cluster
Let's _ssh_ into the CircleCI worker to install `kubectl` so we can talk to our test cluster.
Copy and run the command from this step:

![](kubectl.png)

For example:
```bash
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.10.0/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
```

The `kubeconfig` file needed to connect to the test cluster was created during the environment preparation.
We can inspect the control plane cluster using `kubectl` and that `kubeconfig`:

```bash
export KUBECONFIG="/home/circleci/.kube/config
kubectl get nodes
```

Now we can see the content of our Custom Resources created on the control plane and check if they look correct.
For example:
```bash
kubectl get azureconfigs
```
