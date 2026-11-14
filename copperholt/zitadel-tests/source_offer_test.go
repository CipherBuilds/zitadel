//go:build copperholt_zitadel_upstream

// This regression test is copied into the pinned upstream console package during the Copperholt image build
// (LIC-FIX01). It reads what the image will serve: the Management Console the build embedded and the login v1
// pages the build generated into statik. Every page of both must show the AGPL-3.0 section 13 source offer in
// Compliance's FINAL wording of 14 Nov 2026, with "Source code" linking to the published fork.
package console

import (
	"bytes"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"testing"

	statikfs "github.com/rakyll/statik/fs"

	_ "github.com/zitadel/zitadel/internal/api/ui/login/statik"
)

const (
	copperholtSourceOffer = "This sign-in service runs a modified version of ZITADEL, free software under the GNU " +
		"Affero General Public License v3.0. Get the source code of the version running here, free of charge:"
	copperholtSourceURL = "https://github.com/cipherbuilds/zitadel"
)

func copperholtLoginFile(t *testing.T, files http.FileSystem, name string) string {
	t.Helper()
	file, err := files.Open(name)
	if err != nil {
		t.Fatalf("login v1 file %s is missing: %v", name, err)
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("login v1 file %s is unreadable: %v", name, err)
	}
	return string(raw)
}

func TestCopperholtSourceOfferOnEveryCorePage(t *testing.T) {
	t.Run("console", func(t *testing.T) {
		if _, err := fs.Stat(static, "static/index.html"); err != nil {
			t.Fatalf("the built Management Console is not embedded: %v", err)
		}
		found := false
		err := fs.WalkDir(static, "static", func(name string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || path.Ext(name) != ".js" {
				return err
			}
			raw, err := fs.ReadFile(static, name)
			if err != nil {
				return err
			}
			text := string(raw)
			if strings.Contains(text, copperholtSourceOffer) && strings.Contains(text, copperholtSourceURL) &&
				strings.Contains(text, "noopener noreferrer") && strings.Contains(text, "Source code") {
				found = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("the embedded console could not be read: %v", err)
		}
		if !found {
			t.Fatal("the console footer (shown on every console page) does not carry the source offer and its link")
		}
	})

	t.Run("login v1", func(t *testing.T) {
		files, err := statikfs.NewWithNamespace("login")
		if err != nil {
			t.Fatalf("the login v1 statik files are missing: %v", err)
		}
		// Rendered with every optional footer link and the watermark off: the offer is unconditional.
		footer := copperholtLoginFile(t, files, "/templates/footer.html")
		parsed, err := template.New("footer.html").Funcs(template.FuncMap{
			"hasWatermark": func(any) bool { return false },
			"t":            func(string, ...any) string { return "" },
		}).Parse(footer)
		if err != nil {
			t.Fatalf("the login v1 footer does not parse: %v", err)
		}
		var page bytes.Buffer
		if err := parsed.ExecuteTemplate(&page, "footer", map[string]any{"LabelPolicy": nil, "TOSLink": "",
			"PrivacyLink": "", "HelpLink": "", "SupportEmail": ""}); err != nil {
			t.Fatalf("the login v1 footer does not render: %v", err)
		}
		rendered := strings.Join(strings.Fields(page.String()), " ")
		link := `<a href="` + copperholtSourceURL + `" rel="noopener noreferrer" target="_blank" ` +
			`data-testid="copperholt-source-link">Source code</a>`
		if !strings.Contains(rendered, copperholtSourceOffer+" "+link+" </p>") {
			t.Fatal("the login v1 footer does not render the source offer with its Source code link")
		}
		main := copperholtLoginFile(t, files, "/templates/main.html")
		_, bottom, ok := strings.Cut(main, `{{define "main-bottom"}}`)
		if !ok || !strings.Contains(bottom, `{{template "footer" .}}`) {
			t.Fatal("the login v1 page frame no longer renders the footer")
		}
		if css := copperholtLoginFile(t, files, "/resources/themes/zitadel/css/zitadel.css"); !strings.Contains(css,
			".copperholt-source-offer") {
			t.Fatal("the login v1 stylesheet was built without the source offer style")
		}
		// Every page goes through the frame: a template that opens it also closes it (and so renders the footer),
		// and no template other than the frame renders its own <html>.
		directory, err := files.Open("/templates")
		if err != nil {
			t.Fatalf("the login v1 templates are missing: %v", err)
		}
		defer directory.Close()
		entries, err := directory.Readdir(0)
		if err != nil || len(entries) == 0 {
			t.Fatalf("the login v1 templates cannot be listed: %v", err)
		}
		pages := 0
		for _, entry := range entries {
			name := entry.Name()
			text := copperholtLoginFile(t, files, "/templates/"+name)
			if name != "main.html" && strings.Contains(text, "<html") {
				t.Errorf("%s renders its own <html> outside the page frame and its footer", name)
			}
			if strings.Contains(text, `{{template "main-top"`) {
				pages++
				if !strings.Contains(text, `{{template "main-bottom"`) {
					t.Errorf("%s opens the page frame but never renders its footer", name)
				}
			}
		}
		if pages < 30 {
			t.Fatalf("only %d login v1 pages use the page frame; the template layout changed", pages)
		}
	})
}
