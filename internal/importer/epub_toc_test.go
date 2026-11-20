package importer

import "testing"

func TestNCX(t *testing.T) {
	toc := map[string]map[string]string{}
	readNCX([]byte(`<ncx><navMap><navPoint><navLabel><text>开端</text></navLabel><content src="a.xhtml#start"/></navPoint></navMap></ncx>`), "OEBPS/toc.ncx", toc)
	if toc["OEBPS/a.xhtml"]["start"] != "开端" {
		t.Fatal(toc)
	}
}
