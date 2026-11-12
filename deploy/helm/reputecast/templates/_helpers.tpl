{{/*
Names. The full name is cut to 48 characters so the suffixes the chart adds
(-headless, -metrics, -migrate) stay within Kubernetes' 63.
*/}}
{{- define "reputecast.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "reputecast.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 48 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 48 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 48 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "reputecast.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "reputecast.selectorLabels" -}}
app.kubernetes.io/name: {{ include "reputecast.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "reputecast.labels" -}}
helm.sh/chart: {{ include "reputecast.chart" . }}
{{ include "reputecast.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: reputecast
{{- end }}

{{/* Labels of one component: (dict "root" $ "component" "web"). */}}
{{- define "reputecast.componentLabels" -}}
{{ include "reputecast.labels" .root }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{- define "reputecast.componentSelector" -}}
{{ include "reputecast.selectorLabels" .root }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/* The component that runs the web role: it serves HTTP and the SMTP relay. */}}
{{- define "reputecast.webComponent" -}}
{{- if eq .Values.mode "split" }}web{{ else }}app{{ end }}
{{- end }}

{{- define "reputecast.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "reputecast.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "reputecast.image" -}}
{{- if .Values.image.digest }}
{{- printf "%s@%s" .Values.image.repository .Values.image.digest }}
{{- else }}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}
{{- end }}

{{/* The ingress hosts: the values', or publicURL's host. */}}
{{- define "reputecast.ingressHosts" -}}
{{- if .Values.ingress.hosts }}
{{- toYaml .Values.ingress.hosts }}
{{- else }}
{{- list (urlParse .Values.publicURL).hostname | toYaml }}
{{- end }}
{{- end }}

{{/*
Checks that would otherwise fail late, each with what to do, the mode
against the storage first. Every template that renders a workload includes
it.
*/}}
{{- define "reputecast.validate" -}}
{{- $v := .Values }}
{{- /* The layout first: it decides what the other values must be. */}}
{{- if and (eq $v.mode "split") (ne $v.storage.backend "s3") }}
{{- fail "\n\nmode split needs storage.backend s3: the web and worker pods can't share a local folder, so uploaded files must live in an S3-compatible bucket. Set storage.backend=s3 and storage.s3.bucket, or use mode single." }}
{{- end }}
{{- if not $v.publicURL }}
{{- fail "\n\nSet publicURL to the address people and sending providers reach, such as --set publicURL=https://news.example.com." }}
{{- end }}
{{- $u := urlParse $v.publicURL }}
{{- if or (not (has $u.scheme (list "http" "https"))) (not $u.host) }}
{{- fail (printf "\n\npublicURL must be an absolute http(s) address such as https://news.example.com, not %q." $v.publicURL) }}
{{- end }}
{{- if and (not $v.database.url) (not $v.database.existingSecret) }}
{{- fail "\n\nSet database.url or database.existingSecret: the chart doesn't bundle Postgres. The chart README shows CloudNativePG and a managed database." }}
{{- end }}
{{- if and $v.database.url $v.database.existingSecret }}
{{- fail "\n\nSet database.url or database.existingSecret, not both." }}
{{- end }}
{{- if and (eq $v.storage.backend "s3") (not $v.storage.s3.bucket) }}
{{- fail "\n\nstorage.backend s3 needs storage.s3.bucket." }}
{{- end }}
{{- if ne (empty $v.storage.s3.accessKeyID) (empty $v.storage.s3.secretAccessKey) }}
{{- fail "\n\nstorage.s3.accessKeyID and storage.s3.secretAccessKey go together: set both, or neither for the default AWS credential chain." }}
{{- end }}
{{- if and $v.storage.s3.existingSecret $v.storage.s3.accessKeyID }}
{{- fail "\n\nSet storage.s3.existingSecret or the access keys, not both." }}
{{- end }}
{{- if and $v.storage.s3.kmsKeyID (ne $v.storage.s3.sse "aws:kms") }}
{{- fail "\n\nstorage.s3.kmsKeyID needs storage.s3.sse aws:kms." }}
{{- end }}
{{- if and $v.ingest.separate (ne $v.mode "split") }}
{{- fail "\n\ningest.separate needs mode split: in single mode one pod runs webhook ingest with everything else." }}
{{- end }}
{{- if and $v.hpa.enabled (ne $v.mode "split") }}
{{- fail "\n\nhpa needs mode split: single mode runs one pod." }}
{{- end }}
{{- end }}

{{/* Whether the chart's credentials Secret has the S3 keys. */}}
{{- define "reputecast.s3KeysInline" -}}
{{- if and (eq .Values.storage.backend "s3") .Values.storage.s3.accessKeyID (not .Values.storage.s3.existingSecret) }}true{{ end }}
{{- end }}

{{- define "reputecast.databaseSecretName" -}}
{{- default (include "reputecast.fullname" .) .Values.database.existingSecret }}
{{- end }}

{{- define "reputecast.databaseSecretKey" -}}
{{- if .Values.database.existingSecret }}{{ .Values.database.existingSecretKey }}{{ else }}database-url{{ end }}
{{- end }}

{{- define "reputecast.secretKeySecretName" -}}
{{- default (printf "%s-key" (include "reputecast.fullname" .)) .Values.secretKey.existingSecret }}
{{- end }}

{{- define "reputecast.secretKeySecretKey" -}}
{{- if .Values.secretKey.existingSecret }}{{ .Values.secretKey.existingSecretKey }}{{ else }}secret-key{{ end }}
{{- end }}

{{/*
The environment every ReputeCast container gets besides the ConfigMap: the
database URL, the secret key, the bucket's keys and extraEnv.
*/}}
{{- define "reputecast.secretEnv" -}}
- name: REPUTECAST_DATABASE__URL
  valueFrom:
    secretKeyRef:
      name: {{ include "reputecast.databaseSecretName" . }}
      key: {{ include "reputecast.databaseSecretKey" . }}
- name: REPUTECAST_SECRETS__KEY
  valueFrom:
    secretKeyRef:
      name: {{ include "reputecast.secretKeySecretName" . }}
      key: {{ include "reputecast.secretKeySecretKey" . }}
{{- if eq .Values.storage.backend "s3" }}
{{- $s3 := .Values.storage.s3 }}
{{- if or $s3.existingSecret $s3.accessKeyID }}
{{- $name := default (include "reputecast.fullname" .) $s3.existingSecret }}
- name: REPUTECAST_STORAGE__S3__ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      name: {{ $name }}
      key: {{ if $s3.existingSecret }}{{ $s3.accessKeyIDKey }}{{ else }}s3-access-key-id{{ end }}
- name: REPUTECAST_STORAGE__S3__SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $name }}
      key: {{ if $s3.existingSecret }}{{ $s3.secretAccessKeyKey }}{{ else }}s3-secret-access-key{{ end }}
{{- end }}
{{- end }}
{{- with .Values.extraEnv }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{- define "reputecast.envFrom" -}}
- configMapRef:
    name: {{ include "reputecast.fullname" . }}-env
{{- with .Values.extraEnvFrom }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/* The relay's ports reach the web pods below 1024 only with this sysctl. */}}
{{- define "reputecast.relayLowestPort" -}}
{{- min (int .Values.smtpRelay.starttlsPort) (int .Values.smtpRelay.tlsPort) }}
{{- end }}

{{/*
A ReputeCast pod: (dict "root" $ "component" "web" "args" (list …)
"probePort" "http" "web" true "data" <volume source>).
probePort is the listener the probes ask (a worker has only the metrics one);
web adds the SMTP relay's ports when its Service is on.
*/}}
{{- define "reputecast.podTemplate" -}}
{{- $root := .root }}
{{- $v := $root.Values }}
{{- $relay := and .web $v.smtpRelay.service.enabled -}}
metadata:
  labels:
    {{- include "reputecast.componentLabels" (dict "root" $root "component" .component) | nindent 4 }}
    {{- with $v.podLabels }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  annotations:
    checksum/config: {{ include (print $root.Template.BasePath "/configmap.yaml") $root | sha256sum }}
    checksum/secret: {{ include (print $root.Template.BasePath "/secret.yaml") $root | sha256sum }}
    {{- with $v.podAnnotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
spec:
  serviceAccountName: {{ include "reputecast.serviceAccountName" $root }}
  automountServiceAccountToken: {{ $v.serviceAccount.automount }}
  {{- with $v.imagePullSecrets }}
  imagePullSecrets:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with $v.priorityClassName }}
  priorityClassName: {{ . }}
  {{- end }}
  terminationGracePeriodSeconds: {{ $v.terminationGracePeriodSeconds }}
  securityContext:
    {{- toYaml $v.podSecurityContext | nindent 4 }}
    {{- if and $relay (lt (int (include "reputecast.relayLowestPort" $root)) 1024) }}
    sysctls:
      - name: net.ipv4.ip_unprivileged_port_start
        value: {{ include "reputecast.relayLowestPort" $root | quote }}
    {{- end }}
  containers:
    - name: reputecast
      image: {{ include "reputecast.image" $root }}
      imagePullPolicy: {{ $v.image.pullPolicy }}
      args:
        {{- toYaml .args | nindent 8 }}
      envFrom:
        {{- include "reputecast.envFrom" $root | nindent 8 }}
      env:
        {{- include "reputecast.secretEnv" $root | nindent 8 }}
      ports:
        {{- if ne .probePort "metrics" }}
        - name: http
          containerPort: 8080
          protocol: TCP
        {{- end }}
        - name: metrics
          containerPort: 9464
          protocol: TCP
        {{- if $relay }}
        - name: smtp-starttls
          containerPort: {{ $v.smtpRelay.starttlsPort }}
          protocol: TCP
        - name: smtp-tls
          containerPort: {{ $v.smtpRelay.tlsPort }}
          protocol: TCP
        {{- end }}
      startupProbe:
        httpGet:
          path: /healthz
          port: {{ .probePort }}
        periodSeconds: 5
        failureThreshold: 60
      readinessProbe:
        httpGet:
          path: /readyz
          port: {{ .probePort }}
        periodSeconds: 10
        timeoutSeconds: 3
        failureThreshold: 3
      livenessProbe:
        httpGet:
          path: /healthz
          port: {{ .probePort }}
        periodSeconds: 20
        timeoutSeconds: 3
        failureThreshold: 3
      resources:
        {{- toYaml (default $v.resources .resources) | nindent 8 }}
      securityContext:
        {{- toYaml $v.securityContext | nindent 8 }}
      volumeMounts:
        - name: data
          mountPath: /var/lib/reputecast
        - name: tmp
          mountPath: /tmp
        {{- if $v.geoip.volume }}
        - name: geoip
          mountPath: /var/lib/geoip
          readOnly: true
        {{- end }}
  volumes:
    {{- with .data }}
    - name: data
      {{- toYaml . | nindent 6 }}
    {{- end }}
    - name: tmp
      emptyDir: {}
    {{- with $v.geoip.volume }}
    - name: geoip
      {{- toYaml . | nindent 6 }}
    {{- end }}
  {{- with $v.nodeSelector }}
  nodeSelector:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with $v.affinity }}
  affinity:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with $v.tolerations }}
  tolerations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with $v.topologySpreadConstraints }}
  topologySpreadConstraints:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
