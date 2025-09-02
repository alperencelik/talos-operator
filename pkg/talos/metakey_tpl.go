package talos

const metaKeyTemplate = `
addresses:
    - address: {{ .Endpoint }}/{{ .Subnet }}
      linkName: {{ .Interface }}
      family: inet4
      scope: global
      flags: permanent
      layer: platform
links:
    - name: {{ .Interface }}
      logical: false
      up: true
      mtu: 0
      kind: ""
      type: ether
      layer: platform
routes:
    - family: inet4
      dst: ""
      src: ""
      gateway: {{ .Gateway }}
      outLinkName: {{ .Interface }}
      table: main
      scope: global
      type: unicast
      flags: ""
      protocol: static
      layer: platform
{{- if .Hostname }}
hostnames:
  - hostname: {{ .Hostname }}
    layer: platform
{{- else }}
hostnames: []
{{- end }}
resolvers:
    - dnsServers:
      {{- range .DNSServers }}
        - {{ . }}
      {{- end }}
      layer: platform
timeServers: []
operators: []
externalIPs: []
`
