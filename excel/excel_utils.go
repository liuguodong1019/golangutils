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

// 索引转坐标
func IndexToAxis(col, row int) string {
	axis, _ := excelize.CoordinatesToCellName(col, row)
	return axis
}

// 获取某个单元格（含合并区域）最终值
func GetMergedCellValue(f *excelize.File, sheet, axis string) (string, error) {
	mergeCells, err := f.GetMergeCells(sheet)
	if err != nil {
		return "", err
	}

	// 把当前 axis 转成坐标
	col, row, err := excelize.CellNameToCoordinates(axis)
	if err != nil {
		return "", err
	}

	for _, mc := range mergeCells {
		start := mc.GetStartAxis()
		end := mc.GetEndAxis()

		startCol, startRow, _ := excelize.CellNameToCoordinates(start)
		endCol, endRow, _ := excelize.CellNameToCoordinates(end)

		// 判断是否在合并区域内
		if col >= startCol && col <= endCol &&
			row >= startRow && row <= endRow {

			// 返回起始单元格的值
			return f.GetCellValue(sheet, start)
		}
	}

	// 不在合并区域，正常取值
	return f.GetCellValue(sheet, axis)
}

// 示例，行数范围是34-89，将其中B列中，单个单元格的值和合并的单元格最终值取出来
/*
B34:B36 合并 = OPEN
B37 = CLOSE
B38:B40 合并 = RESET
B43:B67 合并 = OPEN

正确结果应该是：
OPEN
CLOSE
RESET
OPEN
*/
func GetColumnMergedFinalValues(
	f *excelize.File,
	sheet string,
	startRow, endRow int,
	colIndex int, // 列索引，从1开始，例如 B=2
) ([]string, error) {
	var result []string
	mergeCells, err := f.GetMergeCells(sheet)
	if err != nil {
		return nil, err
	}
	// 用于避免同一个合并块被加入多次
	visitedMerge := make(map[string]bool)
	for row := startRow; row <= endRow; row++ {
		axis, _ := excelize.CoordinatesToCellName(colIndex, row)
		col, r := colIndex, row
		isMerged := false
		for _, mc := range mergeCells {
			start := mc.GetStartAxis()
			end := mc.GetEndAxis()
			startCol, startRowM, _ := excelize.CellNameToCoordinates(start)
			endCol, endRowM, _ := excelize.CellNameToCoordinates(end)
			// 判断当前单元格是否属于这个合并区域
			if col >= startCol && col <= endCol &&
				r >= startRowM && r <= endRowM {
				isMerged = true
				// 如果这个合并区域还没加入过
				if !visitedMerge[start] {
					//val, _ := f.GetCellValue(sheet, start)
					val, _ := GetCellValueWithoutStrike(f, sheet, start)
					result = append(result, val)
					visitedMerge[start] = true
				}
				break
			}
		}
		// 如果不是合并单元格
		if !isMerged {
			//val, _ := f.GetCellValue(sheet, axis)
			val, _ := GetCellValueWithoutStrike(f, sheet, axis)
			result = append(result, val)
		}
	}

	return result, nil
}

// 是否单元格所有内容统一设置了删除线
func IsCellStrike(f *excelize.File, sheet, axis string) (bool, error) {
	styleID, err := f.GetCellStyle(sheet, axis)
	if err != nil {
		return false, err
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		return false, err
	}
	if style.Font != nil && style.Font.Strike {
		return true, nil
	}
	return false, nil
}

// 单元格内容是否包含了删除线  示例：RESET：复位/还原/重置,只有“重置”被删除线删除
func HasStrikeInRichText(f *excelize.File, sheet, axis string) (bool, error) {
	richText, err := f.GetCellRichText(sheet, axis)
	if err != nil {
		return false, err
	}
	for _, rt := range richText {
		if rt.Font != nil && rt.Font.Strike {
			return true, nil
		}
	}
	return false, nil
}

// 不确认单元格是所有内容都被删除线删除还是部分内容被删除线删除
func IsCellHasStrike(f *excelize.File, sheet, axis string) (bool, error) {
	// 1️⃣ 先检查富文本
	richText, err := f.GetCellRichText(sheet, axis)
	if err == nil && len(richText) > 0 {
		for _, rt := range richText {
			if rt.Font != nil && rt.Font.Strike {
				return true, nil
			}
		}
	}
	// 2️⃣ 再检查整格样式
	styleID, err := f.GetCellStyle(sheet, axis)
	if err != nil {
		return false, err
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		return false, err
	}
	if style.Font != nil && style.Font.Strike {
		return true, nil
	}
	return false, nil
}

type Resp struct {
	Name      string
	TargetMap map[string][]string
}

// 通过主列获取主列及目标的内容
/*
主列F,目标列G,H,I，  F8-F10是合并单元格：type，G8、G9、G10；H8、H9、H10又不是合并单元格，I8、I9、I10又是合并单元格
返回格式：['type' => ['G' => [],'H' => [],'I' => [] ]]
*/
func GetGroupedByRealBlock(
	f *excelize.File,
	sheet string,
	startRow, endRow int,
	mainColIndex int, //主列
	targetColIndexes []int, //目标列
) ([]Resp, error) {

	var result []Resp

	mergeCells, err := f.GetMergeCells(sheet)
	if err != nil {
		return nil, err
	}

	// 构建：row -> 合并块起始行
	rowToMergeStart := make(map[int]int)

	for _, mc := range mergeCells {

		start := mc.GetStartAxis()
		end := mc.GetEndAxis()

		startCol, startRowM, _ := excelize.CellNameToCoordinates(start)
		_, endRowM, _ := excelize.CellNameToCoordinates(end)

		if startCol != mainColIndex {
			continue
		}

		for r := startRowM; r <= endRowM; r++ {
			rowToMergeStart[r] = startRowM
		}
	}

	for row := startRow; row <= endRow; row++ {

		blockStartRow := row

		// 如果属于合并块
		if startRowM, ok := rowToMergeStart[row]; ok {
			blockStartRow = startRowM
			// 只在合并块的第一行创建结构
			if row != startRowM {
				continue
			}
		}

		mainAxis, _ := excelize.CoordinatesToCellName(mainColIndex, blockStartRow)
		mainValue, _ := GetCellValueWithoutStrike(f, sheet, mainAxis)
		if mainValue == "" {
			continue
		}

		resp := Resp{
			Name:      mainValue,
			TargetMap: make(map[string][]string),
		}

		// 收集该块的所有行
		blockEndRow := blockStartRow
		if _, ok := rowToMergeStart[row]; ok {
			// 找到这个合并块的结束行
			for r := row; r <= endRow; r++ {
				if rowToMergeStart[r] != blockStartRow {
					break
				}
				blockEndRow = r
			}
		}

		// 收集子列
		for r := blockStartRow; r <= blockEndRow; r++ {
			for _, targetCol := range targetColIndexes {

				targetAxis, _ := excelize.CoordinatesToCellName(targetCol, r)

				// 判断该单元格是否是合并块
				merged := false
				mergeStartRow := 0

				for _, mc := range mergeCells {

					start := mc.GetStartAxis()
					end := mc.GetEndAxis()

					startCol, startRowM, _ := excelize.CellNameToCoordinates(start)
					endCol, endRowM, _ := excelize.CellNameToCoordinates(end)

					if targetCol >= startCol && targetCol <= endCol &&
						r >= startRowM && r <= endRowM {

						merged = true
						mergeStartRow = startRowM
						break
					}
				}

				// 如果是合并块，但不是起始行 → 跳过
				if merged && r != mergeStartRow {
					continue
				}

				targetValue, _ := GetCellValueWithoutStrike(f, sheet, targetAxis)
				if targetValue == "" {
					continue
				}

				colName, _ := excelize.ColumnNumberToName(targetCol)
				resp.TargetMap[colName] = append(
					resp.TargetMap[colName],
					targetValue,
				)
			}
		}

		result = append(result, resp)
	}

	return result, nil
}

// 获取合并的单元格起始坐标
func MergeCellStartAndEndIndex(f *excelize.File, sheet string, cell int, rowIndex int) (startRow, endRow int) {
	cell = cell + 1
	mergeCells, _ := f.GetMergeCells(sheet, true)
	cellName, _ := excelize.ColumnNumberToName(cell)
	for _, v := range mergeCells {
		if strings.HasPrefix(v.GetStartAxis(), cellName) && strings.HasPrefix(v.GetEndAxis(), cellName) {
			startRow, _ = strconv.Atoi(strings.TrimPrefix(v.GetStartAxis(), cellName))
			endRow, _ = strconv.Atoi(strings.TrimPrefix(v.GetEndAxis(), cellName))
			if rowIndex == startRow || rowIndex == endRow || rowIndex > startRow && rowIndex < endRow {
				return
			}
		}
	}
	return 0, 0
}

// 获取工作表最大列的数量
func GetMaxColLen(f *excelize.File, sheet string) int {
	rows, _ := f.GetRows(sheet)
	maxCol := 0
	for _, r := range rows {
		if len(r) > maxCol {
			maxCol = len(r)
		}
	}
	return maxCol
}

// 判断是否属于合并行
func IsMergeRow(f *excelize.File, sheet string, rowIndex int) bool {
	mergeCell, _ := f.GetMergeCells(sheet)
	for _, v := range mergeCell {
		s, _ := strconv.Atoi(v.GetStartAxis())
		e, _ := strconv.Atoi(v.GetEndAxis())
		if rowIndex == s || rowIndex == e || rowIndex > s && rowIndex < e {
			return true
		}
	}
	return false
}
