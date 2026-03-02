package utils

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type ReadExcelReq struct {
	FileName  string
	F         *excelize.File
	SheetName string
}
type ReadExcelResp struct {
	Err       error
	F         *excelize.File
	RowData   [][]string
	MergeCell []excelize.MergeCell
	SheetList []string
}

func OpenFile(fileName string) (f *excelize.File, err error) {
	f, err = excelize.OpenFile(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	return
}
func ReadExcel(req *ReadExcelReq) (resp *ReadExcelResp) {
	var err error
	resp = &ReadExcelResp{RowData: make([][]string, 0)}
	if req.F == nil {
		resp.F, err = OpenFile(req.FileName)
		if err != nil {
			resp.Err = err
			fmt.Println(err)
			return
		}
	} else {
		resp.F = req.F
	}
	resp.SheetList = resp.F.GetSheetList()
	if req.SheetName == "" {
		req.SheetName = "Sheet1"
	}
	if !slices.Contains(resp.SheetList, req.SheetName) {
		resp.Err = fmt.Errorf("sheet name %s not exist", req.SheetName)
		return
	}
	resp.MergeCell, resp.Err = resp.F.GetMergeCells(req.SheetName)
	resp.MergeCell = sortMergeCells(resp.MergeCell)
	if resp.Err != nil {
		fmt.Println(resp.Err)
		return
	}
	//sort.Slice(resp.MergeCell, func(i, j int) bool {
	//	colI, rowI := parseRange(resp.MergeCell[i])
	//	colJ, rowJ := parseRange(resp.MergeCell[j])
	//
	//	if rowI != rowJ {
	//		return rowI < rowJ // 先按行
	//	}
	//	return colI < colJ // 行相同按列
	//})
	defer func() {
		if err = resp.F.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	//// 获取工作表中指定单元格的值
	//cell, err := f.GetCellValue("Sheet1", "B2")
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//fmt.Println(cell)

	// 获取 Sheet1 上所有单元格
	resp.RowData, err = resp.F.GetRows(req.SheetName)
	if err != nil {
		fmt.Println(err)
		resp.Err = err
		return
	}
	return resp
	//for _, row := range rows {
	//	for _, colCell := range row {
	//		fmt.Print(colCell, "\t")
	//	}
	//	fmt.Println()
	//}
}

// excel表，数字转大写字母列名
func NumToExcelColumn(n int) string {
	result := ""
	for n > 0 {
		n-- // 注意这个
		result = string(rune('A'+n%26)) + result
		n /= 26
	}
	return result
}

// 索引转列名
func ExcelizeIndexToColName(index int) string {
	var colName string
	colName, _ = excelize.ColumnNumberToName(index)
	return colName
}

// 合并的单元格排序
// 示例：[[B100:B104 ] [C100:C104 ] [D100:D104 ] [E100:E104 ] [F100:F104 ] [B105:B107 ]
func colToNum(col string) int {
	n := 0
	for _, c := range col {
		n = n*26 + int(c-'A'+1)
	}
	return n
}
func parseAxis(rangeStr string) (col int, row int) {
	// rangeStr example: "B100:B104"
	parts := strings.Split(rangeStr, ":")
	axis := parts[0] // start axis, e.g. B100

	i := 0
	for i < len(axis) && axis[i] >= 'A' && axis[i] <= 'Z' {
		i++
	}
	colStr := axis[:i]
	rowStr := axis[i:]

	row, _ = strconv.Atoi(rowStr)
	col = colToNum(colStr)
	return
}

// 对 []excelize.MergeCell 排序
func sortMergeCells(cells []excelize.MergeCell) []excelize.MergeCell {
	sort.Slice(cells, func(i, j int) bool {
		colI, rowI := parseAxis(cells[i][0])
		colJ, rowJ := parseAxis(cells[j][0])

		if rowI != rowJ {
			return rowI < rowJ
		}
		return colI < colJ
	})
	return cells
}

// 获取单元格的值
func CellValue(f *excelize.File, sheet, cell string) string {
	con, _ := f.GetCellValue(sheet, cell)
	return strings.TrimSpace(con)
}

type ColMergeRange struct {
	Cell    excelize.MergeCell
	S       int
	E       int
	Content string
}
type ColMergeRangeResp struct {
	Ranges  []ColMergeRange
	RowNums []int
	Content []string
}

// 获取某一列合并的单元格的范围以及内容
func GetColMergeRowRange(mergeCell []excelize.MergeCell, headerCol string) *ColMergeRangeResp {
	res := &ColMergeRangeResp{
		Ranges: make([]ColMergeRange, 0),
	}
	for _, cell := range mergeCell {
		if !strings.HasPrefix(cell[0], headerCol) {
			continue
		}
		con := strings.TrimSpace(cell.GetCellValue())
		s, _ := strconv.Atoi(strings.TrimPrefix(cell.GetStartAxis(), headerCol))
		e, _ := strconv.Atoi(strings.TrimPrefix(cell.GetEndAxis(), headerCol))
		res.Ranges = append(res.Ranges, ColMergeRange{
			Cell:    cell,
			S:       s,
			E:       e,
			Content: con,
		})
		res.Content = append(res.Content, con)
		res.RowNums = append(res.RowNums, s, e)
	}
	return res
}

// 获取单元格被删除线划掉的内容
func GetCellFontStrike(f *excelize.File, sheet, cell string) (res []string) {
	//runs, _ := f.GetCellRichText("CarControl", "H31")
	runs, _ := f.GetCellRichText(sheet, cell)
	for _, r := range runs {
		if r.Font != nil && r.Font.Strike {
			res = append(res, r.Text)
		}
	}
	return
}

// 如果单元格内容有被划掉（删除线）的内容，返回过滤后的内容
func GetCellValueWithoutStrike(f *excelize.File, sheet, axis string) (string, error) {
	// 1️⃣ 先判断是否整格删除线
	styleID, err := f.GetCellStyle(sheet, axis) //获取单元格样式索引
	if err == nil {
		style, err := f.GetStyle(styleID) //获取单元格样式
		if err == nil && style.Font != nil && style.Font.Strike {
			// 整个单元格被划线 → 直接返回空
			return "", nil
		}
	}

	// 2️⃣ 再处理富文本部分删除线
	runs, err := f.GetCellRichText(sheet, axis) //获取单元格富文本样式
	if err != nil {
		return "", err
	}

	// 不是富文本
	if len(runs) == 0 {
		return f.GetCellValue(sheet, axis)
	}

	var result strings.Builder

	for _, r := range runs {
		if r.Font != nil && r.Font.Strike {
			continue
		}
		result.WriteString(r.Text)
	}

	return result.String(), nil
}

// 判断单元格是否被合并
func IsMergeCell(mergeCells []excelize.MergeCell, sheetName, cell string) bool {
	for _, mergeCell := range mergeCells {
		if mergeCell.GetStartAxis() == cell || mergeCell.GetEndAxis() == cell || (cell > mergeCell.GetStartAxis() && cell < mergeCell.GetEndAxis()) {
			return true
		}
	}
	return false
}
