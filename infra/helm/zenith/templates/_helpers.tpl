{{/*
Platform namespace
*/}}
{{- define "zenith.namespace" -}}
{{- .Values.platform.namespace | default "zenith-platform" }}
{{- end }}

{{/*
Common labels for all resources
*/}}
{{- define "zenith.labels" -}}
app.kubernetes.io/part-of: zenith
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end }}

{{/*
Component labels — call with dict "name" "zenith-api" "component" "api"
*/}}
{{- define "zenith.componentLabels" -}}
app: {{ .name }}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/part-of: zenith
app.kubernetes.io/component: {{ .component }}
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

{{/*
Full image reference — prepends registry prefix if set.
Usage: {{ include "zenith.image" (dict "global" .Values.global "image" .Values.api.image) }}
*/}}
{{- define "zenith.image" -}}
{{- if .global.imageRegistry -}}
{{ .global.imageRegistry }}/{{ .image }}
{{- else -}}
{{ .image }}
{{- end -}}
{{- end }}

{{/*
imagePullSecrets block — included in pod spec when registry is enabled
*/}}
{{- define "zenith.imagePullSecrets" -}}
{{- if .Values.registry.enabled }}
imagePullSecrets:
  - name: {{ .Values.registry.secretName }}
{{- end }}
{{- end }}

{{/*
Docker config JSON for registry authentication (base64 encoded)
*/}}
{{- define "zenith.dockerconfigjson" -}}
{{- $auth := printf "%s:%s" .Values.registry.username .Values.registry.password | b64enc -}}
{{- printf "{\"auths\":{\"%s\":{\"auth\":\"%s\"}}}" .Values.registry.host $auth | b64enc -}}
{{- end }}
