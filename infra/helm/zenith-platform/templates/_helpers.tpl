{{- define "zenith.namespace" -}}
{{- .Values.namespace | default "zenith-platform" }}
{{- end }}

{{- define "zenith.labels" -}}
app.kubernetes.io/part-of: zenith
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end }}

{{- define "zenith.dockerconfigjson" -}}
{{- $auth := printf "%s:%s" .Values.registry.username .Values.registry.password | b64enc -}}
{{- printf "{\"auths\":{\"%s\":{\"auth\":\"%s\"}}}" .Values.registry.host $auth | b64enc -}}
{{- end }}
