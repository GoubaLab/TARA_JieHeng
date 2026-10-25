package assurance

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestReportsStaticHTML(t *testing.T) {
	c := libraryCatalog(t)
	ws := reportWorkspace(t)
	before := Digest(ws)
	data, err := Printable(ws, nil, c)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "<script>") || strings.Contains(text, "animation") || strings.Contains(text, "@keyframes") {
		t.Fatal("report must be safe static HTML")
	}
	if !strings.Contains(text, "&lt;script&gt;") || strings.Count(text, "<section>") != 17 {
		t.Fatal("escaped title or sections missing")
	}
	if Digest(ws) != before {
		t.Fatal("report mutated aggregate")
	}
}
func TestExcelRealRenderer(t *testing.T) {
	if !ExcelAvailable() {
		t.Skip("optional local artifact-tool renderer unavailable")
	}
	c := libraryCatalog(t)
	ws := reportWorkspace(t)
	before := Digest(ws)
	data, e := Excel(ws, nil, c)
	if e != nil {
		t.Fatal(e)
	}
	book, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		t.Fatal(e)
	}
	count := 0
	hasTitle, hasSource, hasFormulaText := false, false, false
	for _, file := range book.File {
		if strings.HasPrefix(file.Name, "xl/worksheets/") && strings.HasSuffix(file.Name, ".xml") {
			count++
		}
		if strings.HasPrefix(file.Name, "xl/worksheets/") || file.Name == "xl/sharedStrings.xml" {
			r, e := file.Open()
			if e != nil {
				t.Fatal(e)
			}
			raw, e := io.ReadAll(r)
			_ = r.Close()
			if e != nil {
				t.Fatal(e)
			}
			text := string(raw)
			hasTitle = hasTitle || strings.Contains(text, "核对项目")
			hasSource = hasSource || strings.Contains(text, "通信介质")
			hasFormulaText = hasFormulaText || strings.Contains(text, "=1+1")
			if strings.Contains(text, "<f>") {
				t.Fatal("user text became a spreadsheet formula")
			}
		}
	}
	if count != 17 || !hasTitle || !hasSource || !hasFormulaText {
		t.Fatalf("invalid exported report: sheets=%d title=%v source=%v text=%v", count, hasTitle, hasSource, hasFormulaText)
	}
	if before != Digest(ws) {
		t.Fatal("Excel mutated aggregate")
	}
}
func reportTestPackage() map[string]string {
	return map[string]string{
		"[Content_Types].xml":        `<Types xmlns="` + reportContentNS + `"><Default Extension="xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		"_rels/.rels":                `<Relationships xmlns="` + reportPackageNS + `"><Relationship Id="book" Type="` + reportOfficeNS + `/officeDocument" Target="/xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<workbook xmlns="` + reportMainNS + `" xmlns:r="` + reportOfficeNS + `"><sheets><sheet name="报告" sheetId="1" r:id="sheet1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships xmlns="` + reportPackageNS + `"><Relationship Id="sheet1" Type="` + reportOfficeNS + `/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   `<worksheet xmlns="` + reportMainNS + `"><sheetData/></worksheet>`,
	}
}
func reportTestZIP(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for name, text := range files {
		file, e := writer.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = file.Write([]byte(text)); e != nil {
			t.Fatal(e)
		}
	}
	if e := writer.Close(); e != nil {
		t.Fatal(e)
	}
	return out.Bytes()
}
func TestExcelPackageValidation(t *testing.T) {
	if e := validateExcel(reportTestZIP(t, reportTestPackage()), []string{"报告"}); e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"not_zip", "empty_zip", "missing_workbook", "missing_sheet", "malformed_sheet", "external_sheet", "unexpected_sheet", "wrong_type", "wrong_root", "missing_sheet_data", "path_escape"} {
		t.Run(mode, func(t *testing.T) {
			files := reportTestPackage()
			switch mode {
			case "empty_zip":
				files = map[string]string{}
			case "missing_workbook":
				delete(files, "xl/workbook.xml")
			case "missing_sheet":
				delete(files, "xl/worksheets/sheet1.xml")
			case "malformed_sheet":
				files["xl/worksheets/sheet1.xml"] = "<invalid"
			case "external_sheet":
				files["xl/_rels/workbook.xml.rels"] = strings.ReplaceAll(files["xl/_rels/workbook.xml.rels"], `Target="worksheets/sheet1.xml"`, `Target="https://invalid.example/sheet.xml" TargetMode="External"`)
			case "unexpected_sheet":
				files["xl/workbook.xml"] = strings.ReplaceAll(files["xl/workbook.xml"], "报告", "错误报告")
			case "wrong_type":
				files["[Content_Types].xml"] = strings.ReplaceAll(files["[Content_Types].xml"], "worksheet+xml", "invalid+xml")
			case "wrong_root":
				files["xl/worksheets/sheet1.xml"] = strings.ReplaceAll(files["xl/worksheets/sheet1.xml"], "worksheet", "notworksheet")
			case "missing_sheet_data":
				files["xl/worksheets/sheet1.xml"] = strings.ReplaceAll(files["xl/worksheets/sheet1.xml"], "<sheetData/>", "")
			case "path_escape":
				files["xl/_rels/workbook.xml.rels"] = strings.ReplaceAll(files["xl/_rels/workbook.xml.rels"], "worksheets/sheet1.xml", "../../sheet1.xml")
			}
			data := reportTestZIP(t, files)
			if mode == "not_zip" {
				data = []byte("not a workbook")
			}
			if e := validateExcel(data, []string{"报告"}); e == nil || !strings.Contains(e.Error(), "不完整或已损坏") {
				t.Fatalf("invalid workbook accepted: %v", e)
			}
		})
	}
}
