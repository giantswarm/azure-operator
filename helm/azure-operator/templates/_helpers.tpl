{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "azure-operator.name" -}}
{{- default .Chart.Name .Values.project.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "azure-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "azure-operator.labels" -}}
helm.sh/chart: {{ include "azure-operator.chart" . }}
{{ include "azure-operator.selectorLabels" . }}
{{ include "azure-operator.name" . }}.giantswarm.io/branch: {{ .Values.project.branch }}
{{ include "azure-operator.name" . }}.giantswarm.io/commit: {{ .Values.project.commit }}
app.kubernetes.io/name: {{ include "azure-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "azure-operator.selectorLabels" -}}
app: {{ include "azure-operator.name" . }}
{{ include "azure-operator.name" . }}.giantswarm.io/version: {{ .Chart.AppVersion }}
{{- end -}}
