package assurance

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Excel export is built into the Go binary and needs no local helper process.
func ExcelAvailable() bool { return true }

func reportCheckReferences(ws Object, catalog *Catalog) error {
	if catalog == nil {
		return nil
	}
	for _, key := range libList(ws["regulation_refs"]) {
		if _, e := catalog.Get(libStr(key)); e != nil {
			return e
		}
	}
	for _, req := range libRows(ws["requirements"]) {
		for _, key := range libList(req["standard_refs"]) {
			if _, e := catalog.Get(libStr(key)); e != nil {
				return e
			}
		}
	}
	return nil
}

func Excel(ws Object, journal []Object, catalog *Catalog) ([]byte, error) {
	if err := reportCheckReferences(ws, catalog); err != nil {
		return nil, err
	}
	return renderExcelTables(Tables(ws, journal, catalog))
}

// renderExcelTables preserves Tables' sheet, column and row order. SetCellStr
// stores formula-like user text verbatim as text; it never evaluates formulas.
func renderExcelTables(tables []Object) ([]byte, error) {
	if len(tables) == 0 {
		return nil, errors.New("Excel 报告没有工作表")
	}
	book := excelize.NewFile()
	defer book.Close()
	if err := book.SetDefaultFont("Microsoft YaHei"); err != nil {
		return nil, err
	}
	if err := book.SetDocProps(&excelize.DocProperties{
		Title: "界衡工作产品报告", Creator: "界衡", Language: "zh-CN",
		Description: "静态工作产品快照；所有单元格均为文本，不含公式或宏。",
	}); err != nil {
		return nil, err
	}
	styles, err := reportExcelStyles(book)
	if err != nil {
		return nil, err
	}
	names, seen := make([]string, 0, len(tables)), map[string]bool{}
	for index, table := range tables {
		name := libStr(table["name"])
		if seen[strings.ToLower(name)] {
			return nil, errors.New("Excel 工作表名称重复")
		}
		seen[strings.ToLower(name)] = true
		if index == 0 {
			err = book.SetSheetName("Sheet1", name)
		} else {
			_, err = book.NewSheet(name)
		}
		if err != nil {
			return nil, err
		}
		if err = reportExcelSheet(book, table, styles); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	book.SetActiveSheet(0)
	buffer, err := book.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	if err = validateExcel(buffer.Bytes(), names); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type reportStyles struct{ title, note, header, body, band int }

func reportExcelStyles(book *excelize.File) (reportStyles, error) {
	result := reportStyles{}
	font := func(size float64, bold bool, color string) *excelize.Font {
		return &excelize.Font{Family: "Microsoft YaHei", Size: size, Bold: bold, Color: color}
	}
	for _, entry := range []struct {
		id    *int
		style *excelize.Style
	}{
		{&result.title, &excelize.Style{Font: font(16, true, "193C35"), NumFmt: 49}},
		{&result.note, &excelize.Style{Font: font(10, false, "6B7866"), NumFmt: 49, Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}}},
		{&result.header, &excelize.Style{Font: font(11, true, "284335"), NumFmt: 49, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E4EAD9"}}, Alignment: &excelize.Alignment{WrapText: true, Vertical: "center", Horizontal: "center"}}},
		{&result.body, &excelize.Style{Font: font(11, false, "243B31"), NumFmt: 49, Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}, Border: []excelize.Border{{Type: "bottom", Color: "E6EADD", Style: 1}}}},
		{&result.band, &excelize.Style{Font: font(11, false, "243B31"), NumFmt: 49, Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}, Border: []excelize.Border{{Type: "bottom", Color: "E6EADD", Style: 1}}, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F7F9F3"}}}},
	} {
		id, err := book.NewStyle(entry.style)
		if err != nil {
			return result, err
		}
		*entry.id = id
	}
	return result, nil
}
func reportTextUnits(text string) int {
	units := 0
	for _, r := range text {
		if r > 0xffff {
			units += 2
		} else {
			units++
		}
	}
	return units
}
func reportSetText(book *excelize.File, sheet, cell string, value any) error {
	text := libStr(value)
	if reportTextUnits(text) > 32767 {
		return fmt.Errorf("工作表“%s”的 %s 超过 Excel 单元格 32767 字符限制；请缩短内容或导出 HTML / JSON", sheet, cell)
	}
	return book.SetCellStr(sheet, cell, text)
}
func reportWrappedHeight(text string, width, minimum float64) float64 {
	lines := 0
	for _, line := range strings.Split(text, "\n") {
		units := 0
		for _, r := range line {
			if r > 255 {
				units += 2
			} else {
				units++
			}
		}
		lines += max(1, int(math.Ceil(float64(units)/max(1, width-2))))
	}
	return min(409, max(minimum, float64(lines)*16+12))
}
func reportExcelSheet(book *excelize.File, table Object, styles reportStyles) error {
	name, columns, rows := libStr(table["name"]), libList(table["columns"]), libList(table["rows"])
	if len(columns) < 1 || len(columns) > 16384 || len(rows) > 1048571 {
		return fmt.Errorf("工作表“%s”的行列数量超出 Excel 限制", name)
	}
	widths := libList(table["widths"])
	if len(widths) != len(columns) {
		return fmt.Errorf("工作表“%s”的列宽与列数不一致", name)
	}
	last, err := excelize.ColumnNumberToName(len(columns))
	if err != nil {
		return err
	}
	showGridLines := false
	if err = book.SetSheetView(name, 0, &excelize.ViewOptions{ShowGridLines: &showGridLines}); err != nil {
		return err
	}
	for _, row := range []int{2, 3} {
		if len(columns) > 1 {
			if err = book.MergeCell(name, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", last, row)); err != nil {
				return err
			}
		}
	}
	if err = reportSetText(book, name, "A2", name); err != nil {
		return err
	}
	note := libStr(table["note"])
	if note == "" {
		note = "工作产品快照；修改后请从工作台重新导出。"
	}
	if err = reportSetText(book, name, "A3", note); err != nil {
		return err
	}
	comment := note
	if reportTextUnits(comment) > 32512 {
		comment = "完整表说明保存在 A3 单元格中。"
	}
	if err = book.AddComment(name, excelize.Comment{Cell: "A3", Author: "界衡", Text: comment, Width: 480, Height: 200}); err != nil {
		return err
	}
	for _, entry := range []struct {
		start, end string
		style      int
	}{
		{"A2", last + "2", styles.title}, {"A3", last + "3", styles.note}, {"A5", last + "5", styles.header},
	} {
		if err = book.SetCellStyle(name, entry.start, entry.end, entry.style); err != nil {
			return err
		}
	}
	totalWidth := 0.0
	for column, label := range columns {
		col, _ := excelize.ColumnNumberToName(column + 1)
		width := float64(libInt(widths[column]))
		if width < 1 || width > 255 {
			return fmt.Errorf("工作表“%s”的列宽超出 Excel 限制", name)
		}
		totalWidth += width
		if err = book.SetColWidth(name, col, col, width); err != nil {
			return err
		}
		if err = reportSetText(book, name, col+"5", label); err != nil {
			return err
		}
	}
	for _, entry := range []struct {
		row    int
		height float64
	}{{1, 12}, {2, 30}, {3, reportWrappedHeight(note, totalWidth, 26)}, {4, 12}, {5, 34}} {
		if err = book.SetRowHeight(name, entry.row, entry.height); err != nil {
			return err
		}
	}
	for index, value := range rows {
		row, height := libList(value), 42.0
		if len(row) > len(columns) {
			return fmt.Errorf("工作表“%s”的数据列数超过表头", name)
		}
		for column := range columns {
			var content any
			if column < len(row) {
				content = row[column]
			}
			cell, _ := excelize.CoordinatesToCellName(column+1, index+6)
			if err = reportSetText(book, name, cell, content); err != nil {
				return err
			}
			height = max(height, reportWrappedHeight(libStr(content), float64(libInt(widths[column])), 42))
		}
		style := styles.body
		if index%2 == 1 {
			style = styles.band
		}
		if err = book.SetCellStyle(name, fmt.Sprintf("A%d", index+6), fmt.Sprintf("%s%d", last, index+6), style); err != nil {
			return err
		}
		if err = book.SetRowHeight(name, index+6, height); err != nil {
			return err
		}
	}
	panes := &excelize.Panes{Freeze: true, YSplit: 5, TopLeftCell: "A6", ActivePane: "bottomLeft"}
	if len(columns) > 6 {
		panes.XSplit, panes.TopLeftCell, panes.ActivePane = 1, "B6", "bottomRight"
	}
	return book.SetPanes(name, panes)
}

const reportMainNS = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
const reportOfficeNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
const reportPackageNS = "http://schemas.openxmlformats.org/package/2006/relationships"
const reportContentNS = "http://schemas.openxmlformats.org/package/2006/content-types"

type reportXML struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Children []*reportXML
}

func (x *reportXML) attr(space, local string) string {
	for _, a := range x.Attrs {
		if a.Name.Space == space && a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}
func (x *reportXML) children(space, local string) []*reportXML {
	rows := []*reportXML{}
	for _, c := range x.Children {
		if c.Name.Space == space && c.Name.Local == local {
			rows = append(rows, c)
		}
	}
	return rows
}
func reportParseXML(data []byte) (*reportXML, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root *reportXML
	stack := []*reportXML{}
	for {
		token, e := decoder.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		switch t := token.(type) {
		case xml.StartElement:
			n := &reportXML{Name: t.Name, Attrs: t.Attr}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, n)
			} else {
				if root != nil {
					return nil, errors.New("multiple XML roots")
				}
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		}
	}
	if root == nil || len(stack) > 0 {
		return nil, errors.New("incomplete XML")
	}
	return root, nil
}
func validateExcel(data []byte, expectedSheets []string) (err error) {
	defer func() {
		if err != nil {
			err = errors.New("本地报表文件不完整或已损坏，请重新生成或检查渲染组件")
		}
	}()
	archive, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return e
	}
	files := map[string][]byte{}
	for _, file := range archive.File {
		if _, ok := files[file.Name]; ok {
			return errors.New("duplicate ZIP member")
		}
		reader, e := file.Open()
		if e != nil {
			return e
		}
		body, e := io.ReadAll(reader)
		closeErr := reader.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
		files[file.Name] = body
	}
	// Reports are deliberately inert: no macros, embedded programs, external
	// relationships, or executable spreadsheet formulas are expected here.
	for name, body := range files {
		if path.Clean(name) != name || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "../") || strings.HasSuffix(strings.ToLower(name), ".bin") {
			return errors.New("unexpected executable or unsafe workbook part")
		}
		if strings.HasSuffix(name, ".rels") {
			root, err := reportParseXML(body)
			if err != nil || root.Name.Space != reportPackageNS || root.Name.Local != "Relationships" {
				return errors.New("invalid package relationships")
			}
			for _, link := range root.children(reportPackageNS, "Relationship") {
				typ := strings.ToLower(link.attr("", "Type"))
				if strings.EqualFold(link.attr("", "TargetMode"), "External") || strings.Contains(typ, "vbaproject") || strings.Contains(typ, "oleobject") || strings.Contains(typ, "externallink") || strings.HasSuffix(typ, "/control") {
					return errors.New("active or external workbook relationship")
				}
			}
		}
	}
	read := func(name, space, local string) (*reportXML, error) {
		body, ok := files[name]
		if !ok {
			return nil, fmt.Errorf("missing part %s", name)
		}
		root, e := reportParseXML(body)
		if e != nil {
			return nil, e
		}
		if root.Name.Space != space || root.Name.Local != local {
			return nil, errors.New("invalid XML root")
		}
		return root, nil
	}
	types, e := read("[Content_Types].xml", reportContentNS, "Types")
	if e != nil {
		return e
	}
	declared, defaults := map[string]string{}, map[string]string{}
	for _, v := range types.children(reportContentNS, "Override") {
		declared[v.attr("", "PartName")] = v.attr("", "ContentType")
	}
	for _, v := range types.children(reportContentNS, "Default") {
		defaults[v.attr("", "Extension")] = v.attr("", "ContentType")
	}
	contentType := func(name string) string {
		if value, ok := declared["/"+name]; ok {
			return value
		}
		return defaults[strings.TrimPrefix(path.Ext(name), ".")]
	}
	if contentType("xl/workbook.xml") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml" {
		return errors.New("missing workbook content type")
	}
	rootLinks, e := read("_rels/.rels", reportPackageNS, "Relationships")
	if e != nil {
		return e
	}
	found := false
	for _, link := range rootLinks.children(reportPackageNS, "Relationship") {
		if link.attr("", "Type") == reportOfficeNS+"/officeDocument" && strings.TrimLeft(link.attr("", "Target"), "/") == "xl/workbook.xml" && link.attr("", "TargetMode") != "External" {
			found = true
		}
	}
	if !found {
		return errors.New("missing workbook relationship")
	}
	workbook, e := read("xl/workbook.xml", reportMainNS, "workbook")
	if e != nil {
		return e
	}
	sheets := []*reportXML{}
	for _, group := range workbook.children(reportMainNS, "sheets") {
		sheets = append(sheets, group.children(reportMainNS, "sheet")...)
	}
	if len(sheets) == 0 || len(sheets) != len(expectedSheets) {
		return errors.New("incomplete report worksheets")
	}
	for i, sheet := range sheets {
		if sheet.attr("", "name") != expectedSheets[i] {
			return errors.New("unexpected report worksheet")
		}
	}
	links, e := read("xl/_rels/workbook.xml.rels", reportPackageNS, "Relationships")
	if e != nil {
		return e
	}
	byID := map[string]*reportXML{}
	for _, link := range links.children(reportPackageNS, "Relationship") {
		id := link.attr("", "Id")
		if _, exists := byID[id]; exists {
			return errors.New("duplicate relationship")
		}
		byID[id] = link
	}
	for _, sheet := range sheets {
		link := byID[sheet.attr(reportOfficeNS, "id")]
		if link == nil || link.attr("", "Type") != reportOfficeNS+"/worksheet" || link.attr("", "TargetMode") == "External" {
			return errors.New("invalid worksheet relationship")
		}
		target := link.attr("", "Target")
		name := ""
		if strings.HasPrefix(target, "/") {
			name = path.Clean(strings.TrimLeft(target, "/"))
		} else {
			name = path.Clean("xl/" + target)
		}
		if !strings.HasPrefix(name, "xl/worksheets/") || contentType(name) != "application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml" {
			return errors.New("missing worksheet content type")
		}
		worksheet, e := read(name, reportMainNS, "worksheet")
		if e != nil {
			return e
		}
		if len(worksheet.children(reportMainNS, "sheetData")) != 1 {
			return errors.New("missing worksheet data")
		}
		if e = reportValidateStaticCells(worksheet); e != nil {
			return e
		}
	}
	return nil
}

func reportValidateStaticCells(node *reportXML) error {
	if node.Name.Space == reportMainNS {
		switch node.Name.Local {
		case "f", "oleObjects", "controls", "externalReferences":
			return errors.New("active spreadsheet content")
		case "c":
			// Styled empty cells need no type; every populated value is text.
			if len(node.children(reportMainNS, "v"))+len(node.children(reportMainNS, "is")) > 0 && node.attr("", "t") != "s" && node.attr("", "t") != "inlineStr" {
				return errors.New("report cell must be a string")
			}
		}
	}
	for _, child := range node.Children {
		if err := reportValidateStaticCells(child); err != nil {
			return err
		}
	}
	return nil
}
