{{/*
Add helper methods here for your chart
*/}}

{{- define "cray-hms-hmcollector.image-prefix" -}}
{{ $base := .Values }}
{{- if $base.imagesHost -}}
{{- printf "%s/" $base.imagesHost -}}
{{- else -}}
{{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
Helper function to get the proper image tag
*/}}
{{- define "cray-hms-hmcollector.imageTag" -}}
{{- default "latest" .Chart.AppVersion -}}
{{- end -}}
