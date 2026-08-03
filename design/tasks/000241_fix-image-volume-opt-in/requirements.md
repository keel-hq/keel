# Requirements: Align Image-Volume Opt-In Updates and Compatibility Documentation

## User Stories

- As a Keel user, I want image-volume updates to honor the same label and annotation opt-ins used during discovery so discovered image volumes can be rewritten.
- As a cluster operator, I want accurate Kubernetes compatibility guidance so I know when the feature gate and runtime support are required.

## Acceptance Criteria

- `checkForUpdate` uses `getImageVolumeTrackingFromMeta` with labels first and annotations second.
- The existing helper behavior is preserved: keys are matched case-insensitively, labels take precedence over annotations, and only the value `"true"` enables tracking.
- Image volumes are rewritten for a canonical label-only opt-in and for a case-variant opt-in key.
- Canonical annotation opt-in remains covered, while resources without a true opt-in remain unchanged.
- Existing `monitorContainers` filtering, workload-kind support, standard container updates, and init-container updates are unchanged.
- User documentation states that Kubernetes 1.31-1.34 requires explicitly enabling `ImageVolume`, Kubernetes 1.35 enables the beta feature by default, Kubernetes 1.36+ provides it as stable/GA, and the container runtime must support image volumes.
- Documentation retains links to the official Kubernetes feature-gate table and image-volume guide, and adjacent compatibility comments do not claim that the gate is always required.
- The change remains limited to the image-volume opt-in mismatch, its regression tests, and related compatibility wording.

## Open Questions

None.
