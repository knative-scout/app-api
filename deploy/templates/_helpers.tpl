{{/*
Specialized .Values.host default value logic
*/}}
{{- define "serverless-registry-api.host" -}}
{{- if .Values.host -}}
{{ .Values.host }}
{{- else if eq .Values.global.env "prod" -}}
api.kscout.io
{{- else -}}
{{ .Values.global.env }}-api.kscout.io
{{- end -}}
{{- end -}}
