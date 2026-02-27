{{- define "zenith.image" -}}
{{- if .registry -}}
{{ .registry }}/{{ .image }}
{{- else -}}
{{ .image }}
{{- end -}}
{{- end }}

{{- define "zenith.dockerconfigjson" -}}
{{- $auth := printf "%s:%s" .Values.registry.username .Values.registry.password | b64enc -}}
{{- printf "{\"auths\":{\"%s\":{\"auth\":\"%s\"}}}" .Values.registry.host $auth | b64enc -}}
{{- end }}
