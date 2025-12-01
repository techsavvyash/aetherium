{{/*
Expand the name of the chart.
*/}}
{{- define "aetherium.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "aetherium.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "aetherium.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "aetherium.labels" -}}
helm.sh/chart: {{ include "aetherium.chart" . }}
{{ include "aetherium.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: aetherium
environment: {{ .Values.global.environment }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "aetherium.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aetherium.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
API Gateway labels
*/}}
{{- define "aetherium.apiGateway.labels" -}}
{{ include "aetherium.labels" . }}
app.kubernetes.io/component: api-gateway
{{- end }}

{{/*
API Gateway selector labels
*/}}
{{- define "aetherium.apiGateway.selectorLabels" -}}
{{ include "aetherium.selectorLabels" . }}
app.kubernetes.io/component: api-gateway
{{- end }}

{{/*
Worker labels
*/}}
{{- define "aetherium.worker.labels" -}}
{{ include "aetherium.labels" . }}
app.kubernetes.io/component: worker
{{- end }}

{{/*
Worker selector labels
*/}}
{{- define "aetherium.worker.selectorLabels" -}}
{{ include "aetherium.selectorLabels" . }}
app.kubernetes.io/component: worker
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "aetherium.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "aetherium.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the config map
*/}}
{{- define "aetherium.configMapName" -}}
{{- if .Values.configMap.create }}
{{- default (printf "%s-config" (include "aetherium.fullname" .)) .Values.configMap.name }}
{{- else }}
{{- default "aetherium-config" .Values.configMap.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the secrets
*/}}
{{- define "aetherium.secretName" -}}
{{- printf "%s-secrets" (include "aetherium.fullname" .) }}
{{- end }}

{{/*
Get PostgreSQL host
*/}}
{{- define "aetherium.postgresql.host" -}}
{{- if .Values.postgresql.enabled }}
{{- printf "%s-postgresql" .Release.Name }}
{{- else }}
{{- .Values.postgresql.external.host }}
{{- end }}
{{- end }}

{{/*
Get PostgreSQL port
*/}}
{{- define "aetherium.postgresql.port" -}}
{{- if .Values.postgresql.enabled }}
{{- "5432" }}
{{- else }}
{{- .Values.postgresql.external.port | toString }}
{{- end }}
{{- end }}

{{/*
Get Redis host
*/}}
{{- define "aetherium.redis.host" -}}
{{- if .Values.redis.enabled }}
{{- printf "%s-redis-master" .Release.Name }}
{{- else }}
{{- .Values.redis.external.host }}
{{- end }}
{{- end }}

{{/*
Get Redis port
*/}}
{{- define "aetherium.redis.port" -}}
{{- if .Values.redis.enabled }}
{{- "6379" }}
{{- else }}
{{- .Values.redis.external.port | toString }}
{{- end }}
{{- end }}

{{/*
Image pull secrets
*/}}
{{- define "aetherium.imagePullSecrets" -}}
{{- if .Values.global.imagePullSecrets }}
imagePullSecrets:
{{- range .Values.global.imagePullSecrets }}
  - name: {{ . }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Full image name for API Gateway
*/}}
{{- define "aetherium.apiGateway.image" -}}
{{- if .Values.global.imageRegistry }}
{{- printf "%s/%s:%s" .Values.global.imageRegistry .Values.apiGateway.image.repository .Values.apiGateway.image.tag }}
{{- else }}
{{- printf "%s:%s" .Values.apiGateway.image.repository .Values.apiGateway.image.tag }}
{{- end }}
{{- end }}

{{/*
Full image name for Worker
*/}}
{{- define "aetherium.worker.image" -}}
{{- if .Values.global.imageRegistry }}
{{- printf "%s/%s:%s" .Values.global.imageRegistry .Values.worker.image.repository .Values.worker.image.tag }}
{{- else }}
{{- printf "%s:%s" .Values.worker.image.repository .Values.worker.image.tag }}
{{- end }}
{{- end }}
