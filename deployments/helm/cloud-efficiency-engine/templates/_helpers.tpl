{{/*
Expand the name of the chart.
*/}}
{{- define "cloud-efficiency-engine.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "cloud-efficiency-engine.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name (include "cloud-efficiency-engine.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Chart label.
*/}}
{{- define "cloud-efficiency-engine.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "cloud-efficiency-engine.labels" -}}
helm.sh/chart: {{ include "cloud-efficiency-engine.chart" . }}
{{ include "cloud-efficiency-engine.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "cloud-efficiency-engine.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cloud-efficiency-engine.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: cloud-efficiency-engine
{{- end }}

{{/*
Service name.
*/}}
{{- define "cloud-efficiency-engine.serviceName" -}}
{{ include "cloud-efficiency-engine.fullname" . }}
{{- end }}

{{/*
Namespace.
*/}}
{{- define "cloud-efficiency-engine.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride }}
{{- end }}