package windowing

import (
	"net/url"
	"strings"
)

type OpenWindowRequest struct {
	ObjectPath       string
	ObjectPathSuffix string
	Pathname         string
	PreviewID        string
	PreviewSrc       string
	PreviewTitle     string
	Provider         string
	StorageID        string
}

type WindowSpec struct {
	EntryPage string
	Height    int
	Name      string
	Pathname  string
	Title     string
	Width     int
}

func BuildOpenWindowSpec(req OpenWindowRequest) WindowSpec {
	pathname := req.Pathname
	if pathname == "" {
		pathname = "/settings"
	}

	storageID := strings.TrimSpace(req.StorageID)
	objectPath := strings.TrimSpace(req.ObjectPath)
	previewID := strings.TrimSpace(req.PreviewID)
	previewSrc := strings.TrimSpace(req.PreviewSrc)
	previewTitle := strings.TrimSpace(req.PreviewTitle)
	provider := strings.ToLower(strings.TrimSpace(req.Provider))

	if pathname == "/oss-manager" && storageID != "" {
		pathname += "?storageId=" + url.QueryEscape(storageID)
	}
	if pathname == "/oss-storage-editor" {
		params := url.Values{}
		if storageID != "" {
			params.Set("storageId", storageID)
		}
		if provider != "" {
			params.Set("provider", provider)
		}
		if encoded := params.Encode(); encoded != "" {
			pathname += "?" + encoded
		}
	}
	if pathname == "/oss-preview" {
		params := url.Values{}
		if storageID != "" {
			params.Set("storageId", storageID)
		}
		if objectPath != "" {
			params.Set("objectPath", objectPath)
		}
		if encoded := params.Encode(); encoded != "" {
			pathname += "?" + encoded
		}
	}
	if pathname == "/image-preview" {
		params := url.Values{}
		if previewID != "" {
			params.Set("id", previewID)
		}
		if previewSrc != "" {
			params.Set("src", previewSrc)
		}
		if previewTitle != "" {
			params.Set("title", previewTitle)
		}
		if encoded := params.Encode(); encoded != "" {
			pathname += "?" + encoded
		}
	}

	pathBase := pathname
	if index := strings.Index(pathBase, "?"); index >= 0 {
		pathBase = pathBase[:index]
	}

	spec := WindowSpec{
		EntryPage: "index.html",
		Height:    640,
		Name:      "app-window",
		Pathname:  pathname,
		Title:     "App",
		Width:     760,
	}
	switch pathBase {
	case "/desktop":
		spec.EntryPage = "index.html"
		spec.Name = "desktop"
		spec.Title = "App-Main"
		spec.Width = 1024
		spec.Height = 768
	case "/settings":
		spec.EntryPage = "settings.html"
		spec.Name = "settings"
		spec.Title = "App-Settings"
	case "/oss-manager":
		spec.EntryPage = "oss-manager.html"
		spec.Name = "oss-manager"
		spec.Title = "OSS 文件管理"
		spec.Width = 1040
		spec.Height = 720
		if storageID != "" {
			spec.Name += "-" + storageID
		}
	case "/oss-storage-editor":
		spec.EntryPage = "oss-storage-editor.html"
		spec.Name = "oss-storage-editor"
		spec.Title = "OSS 存储编辑"
		spec.Width = 760
		spec.Height = 720
	case "/oss-preview":
		spec.EntryPage = "oss-preview.html"
		spec.Name = "oss-preview"
		spec.Title = "OSS 文件预览"
		spec.Width = 860
		spec.Height = 680
		if storageID != "" {
			spec.Name += "-" + storageID
		}
		if objectPath != "" && req.ObjectPathSuffix != "" {
			spec.Name += "-" + req.ObjectPathSuffix
		}
	case "/image-preview":
		spec.EntryPage = "image-preview.html"
		spec.Name = "image-preview"
		spec.Title = "图片预览"
		spec.Width = 980
		spec.Height = 760
		if previewID != "" {
			spec.Name += "-" + previewID
		}
	case "/memo-slim":
		spec.EntryPage = "memo-slim.html"
		spec.Name = "memo-slim"
		spec.Title = "Memos"
		spec.Width = 430
		spec.Height = 640
	case "/gtd-slim":
		spec.EntryPage = "gtd-slim.html"
		spec.Name = "gtd-slim"
		spec.Title = "Todos"
		spec.Width = 420
		spec.Height = 640
	case "/timeline":
		spec.EntryPage = "timeline-window.html"
		spec.Name = "timeline"
		spec.Title = "时间线"
		spec.Width = 420
		spec.Height = 640
	}
	return spec
}
