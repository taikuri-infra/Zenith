{{- define "zenith.image" -}}
{{- if .registry -}}
{{ .registry }}/{{ .image }}
{{- else -}}
{{ .image }}
{{- end -}}
{{- end }}

{{/*
IngressRoute host match — call with a list of hosts
Returns: Host(`h1`) || Host(`h2`)
*/}}
{{- define "zenith.hostMatch" -}}
{{- $result := list -}}
{{- range . -}}
{{- $result = append $result (printf "Host(`%s`)" .) -}}
{{- end -}}
{{- join " || " $result -}}
{{- end }}
