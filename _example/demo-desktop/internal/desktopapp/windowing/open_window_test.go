package windowing

import "testing"

func TestBuildOpenWindowSpecDefaultsToSettings(t *testing.T) {
	spec := BuildOpenWindowSpec(OpenWindowRequest{})
	if spec.Pathname != "/settings" || spec.EntryPage != "settings.html" || spec.Name != "settings" {
		t.Fatalf("spec = %#v, want settings window", spec)
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
