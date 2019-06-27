{{/*
Specialized .Values.host default value logic
*/}}
{{- define "serverless-registry-api.host" -}}
{{- if .Values.host -}}
{{ .Values.host }}
{{- else if eq .Values.env "prod" -}}
api.kscout.io
{{- else -}}
{{ .Values.env }}-api.kscout.io
{{- end -}}
{{- end -}}
