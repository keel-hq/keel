{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "keel.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "serviceAccount.name" -}}
{{- if .Values.rbac.serviceAccount.name -}}
{{- .Values.rbac.serviceAccount.name -}}
{{- else -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "keel.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "keel.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Validate the explicit administrator authentication topology. */}}
{{- define "keel.validateAuth" -}}
{{- $mode := default "legacy" .Values.auth.mode -}}
{{- if not (has $mode (list "legacy" "basic" "external-proxy")) -}}
{{- fail (printf "auth.mode must be one of legacy, basic, or external-proxy, got %q" $mode) -}}
{{- end -}}
{{- if and (eq $mode "basic") (not .Values.basicauth.enabled) -}}
{{- fail "auth.mode=basic requires basicauth.enabled=true and non-empty credentials" -}}
{{- end -}}
{{- if and .Values.basicauth.enabled (or (empty .Values.basicauth.user) (empty .Values.basicauth.password)) -}}
{{- fail "basicauth.enabled=true requires non-empty basicauth.user and basicauth.password" -}}
{{- end -}}
{{- if eq $mode "external-proxy" -}}
{{- if .Values.basicauth.enabled -}}
{{- fail "auth.mode=external-proxy conflicts with basicauth.enabled=true" -}}
{{- end -}}
{{- if not .Values.oauth2Proxy.enabled -}}
{{- fail "auth.mode=external-proxy requires oauth2Proxy.enabled=true so the loopback-only Keel listener has an authenticated entrypoint" -}}
{{- end -}}
{{- if empty .Values.oauth2Proxy.existingSecret -}}
{{- fail "auth.mode=external-proxy requires oauth2Proxy.existingSecret" -}}
{{- end -}}
{{- if not .Values.service.enabled -}}
{{- fail "auth.mode=external-proxy requires service.enabled=true; the Service targets oauth2-proxy, never Keel directly" -}}
{{- end -}}
{{- else if .Values.oauth2Proxy.enabled -}}
{{- fail "oauth2Proxy.enabled=true requires auth.mode=external-proxy" -}}
{{- end -}}
{{- end -}}
