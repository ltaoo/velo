package windowing

import "testing"

func TestBuildOpenWindowSpecDefaultsToSettings(t *testing.T) {
	spec := BuildOpenWindowSpec(OpenWindowRequest{})
	if spec.Pathname != "/settings" || spec.EntryPage != "settings.html" || spec.Name != "settings" {
		t.Fatalf("spec = %#v, want settings window", spec)
	}
}

func TestBuildOpenWindowSpecDesktop(t *testing.T) {
	spec := BuildOpenWindowSpec(OpenWindowRequest{Pathname: "/desktop"})
	if spec.Pathname != "/desktop" || spec.EntryPage != "index.html" || spec.Name != "desktop" {
		t.Fatalf("spec = %#v, want desktop window", spec)
	}
	if spec.Title != "App-Main" || spec.Width != 1024 || spec.Height != 768 {
		t.Fatalf("spec = %#v, want main desktop dimensions", spec)
	}
}

func TestBuildOpenWindowSpecPreviewIncludesQueryAndStableName(t *testing.T) {
	spec := BuildOpenWindowSpec(OpenWindowRequest{
		ObjectPath:       "docs/report.pdf",
		ObjectPathSuffix: "docs-report-pdf",
		Pathname:         "/oss-preview",
		StorageID:        "memo-local",
	})
	wantPathname := "/oss-preview?objectPath=docs%2Freport.pdf&storageId=memo-local"
	if spec.Pathname != wantPathname {
		t.Fatalf("pathname = %q, want %q", spec.Pathname, wantPathname)
	}
	if spec.Name != "oss-preview-memo-local-docs-report-pdf" || spec.EntryPage != "oss-preview.html" || spec.Width != 860 || spec.Height != 680 {
		t.Fatalf("spec = %#v, want preview window", spec)
	}
}

func TestBuildOpenWindowSpecImagePreviewIncludesID(t *testing.T) {
	spec := BuildOpenWindowSpec(OpenWindowRequest{
		Pathname:  "/image-preview",
		PreviewID: "preview-123",
	})
	if spec.Pathname != "/image-preview?id=preview-123" {
		t.Fatalf("pathname = %q, want image preview id query", spec.Pathname)
	}
	if spec.Name != "image-preview-preview-123" || spec.EntryPage != "image-preview.html" || spec.Width != 980 || spec.Height != 760 {
		t.Fatalf("spec = %#v, want image preview window", spec)
	}
}

func TestBuildOpenWindowSpecGTDSlim(t *testing.T) {
	spec := BuildOpenWindowSpec(OpenWindowRequest{Pathname: "/gtd-slim"})
	if spec.Pathname != "/gtd-slim" {
		t.Fatalf("pathname = %q, want /gtd-slim", spec.Pathname)
	}
	if spec.Name != "gtd-slim" || spec.EntryPage != "gtd-slim.html" || spec.Title != "Todos" || spec.Width != 420 || spec.Height != 640 {
		t.Fatalf("spec = %#v, want gtd slim window", spec)
	}
}
