# Deployment files

This directory contains example deployment manifests for `keel` that can
be used in place of the official Helm chart.

This is useful if you are deploying `keel` into an environment without
Helm, or want to inspect a 'bare minimum' deployment.

## Where do these come from?

The manifests in this are generated from the Helm chart automatically.
The `values.yaml` files used to configure `keel` can be found in
[`values`](../chart/keel/values.yaml).

<!-- Deprecated -->
They are automatically generated by running `./deployment/scripts/gen-deploy.sh`.
