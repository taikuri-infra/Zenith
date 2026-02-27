{{- define "zenith.image" -}}
{{- if .registry -}}
{{ .registry }}/{{ .image }}
{{- else -}}
{{ .image }}
{{- end -}}
{{- end }}
