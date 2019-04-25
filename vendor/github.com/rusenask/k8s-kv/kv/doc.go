/*
Package kv implements a low-level key/value store backed by Kubernetes config maps.
It supports main operations expected from key/value store such as Put, Get, Delete and List.
Operations are protected by an internal mutex and therefore can be safely used inside a single
node application.
Basics
There are only few things worth to know: key/value database is created based on bucket name so in order
to have multiple configMaps - use different bucket names. Teardown() function will remove configMap entry
completely destroying all entries.
Caveats
Since k8s-kv is based on configMaps which are in turn based on Etcd key/value store - all values have a limitation
of 1MB so each bucket in k8s-kv is limited to that size. To overcome it - create more buckets.
If you have multi-node application that is frequently reading/writing to the same buckets - be aware of race
conditions as it doesn't provide any cross-node locking capabilities.
*/
package kv
