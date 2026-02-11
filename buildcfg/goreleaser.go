package buildcfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var goreleaserTmpl = template.Must(template.New("goreleaser").Parse(`# Auto-generated from app-config.json
version: 2

project_name: {{.ProjectName}}

before:
  hooks:
    - go mod tidy

builds:
  - id: windows
    env:
      - CGO_ENABLED=1
    goos:
      - windows
    goarch:
      - amd64
    main: .
    binary: {{.AppName}}
    ldflags:
      - -s -w
      - -X main.Mode=release
      - -H windowsgui

  - id: linux
    env:
      - CGO_ENABLED=0
    goos:
      - linux
    goarch:
      - amd64
      - arm64
    main: .
    binary: {{.AppName}}
    ldflags:
      - -s -w
      - -X main.Mode=release

  - id: macos
    env:
      - CGO_ENABLED=1
    goos:
      - darwin
    goarch:
      - amd64
      - arm64
    main: .
    binary: {{.AppName}}
    ldflags:
      - -s -w
      - -X main.Mode=release
    hooks:
      post:
        - rcodesign sign --p12-file {{ "{{ .Env.MAC_CERT_P12_FILE }}" }} --p12-password {{ "{{ .Env.MAC_CERT_PASSWORD }}" }} --code-signature-flags runtime --entitlements-xml-path {{ "{{ .Env.MAC_ENTITLEMENTS_PATH }}" }} "{{ "{{ .Path }}" }}"

archives:
  - id: default
    format: tar.gz
    name_template: "{{ "{{ .ProjectName }}" }}_{{ "{{ .Os }}" }}_{{ "{{ .Arch }}" }}"
    files:
{{- range .ConfigFiles}}
      - src: {{.Src}}
        dst: {{.Dst}}
{{- end}}
{{- range .ExcludeFiles}}
      - src: "{{.}}"
{{- end}}
    format_overrides:
      - goos: windows
        format: zip
      - goos: darwin
        format: binary

upx:
  - enabled: true
    ids:
      - linux
    compress: best
    lzma: true

checksum:
  name_template: "{{ "{{ .ProjectName }}" }}_{{ "{{ .Version }}" }}_checksums.txt"
  algorithm: sha256

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"

release:
  draft: true
  footer: >-
    {{.Footer}}
  extra_files:
    - glob: "{{ "{{ .ProjectName }}" }}_{{ "{{ .Version }}" }}_checksums.txt"
`))

type goreleaserData struct {
	ProjectName  string
	AppName      string
	ConfigFiles  []ConfigFile
	ExcludeFiles []string
	Footer       string
}

func GenerateGoreleaser(cfg *Config, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	data := goreleaserData{
		ProjectName:  cfg.ProjectName(),
		AppName:      cfg.App.Name,
		ConfigFiles:  cfg.Build.ConfigFiles,
		ExcludeFiles: cfg.Build.ExcludeFiles,
		Footer:       strings.TrimSpace(cfg.Release.Footer),
	}

	f, err := os.Create(filepath.Join(outDir, ".goreleaser.yaml"))
	if err != nil {
		return fmt.Errorf("creating .goreleaser.yaml: %w", err)
	}
	defer f.Close()

	return goreleaserTmpl.Execute(f, data)
}
