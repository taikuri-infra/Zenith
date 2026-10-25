{{- define "zenith.labels" -}}
app.kubernetes.io/part-of: zenith
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
