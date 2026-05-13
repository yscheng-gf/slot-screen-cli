package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
)

type mode int

const (
	modeNormal mode = iota
	modeInputID
	modeInputBT
	modeInputW
	modeInputL
)

type Symbol struct {
	ID     int32 `json:"id"`
	BT     int   `json:"bt"`
	Width  int   `json:"w"`
	Length int   `json:"l"`
	X, Y   int
}

type SymbolOut struct {
	ID      int32 `json:"id"`
	BT      int   `json:"bt,omitempty"`
	Length  int   `json:"l"`
	Width   int   `json:"w"`
	Index   int32 `json:"i"`
	Index2D Idx2D `json:"i2d"`
}

type Idx2D struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type model struct {
	layout      []int
	grid        [][]*Symbol
	cursorX     int
	cursorY     int
	selectedID  int32
	selectedBT  int
	ids         []int32
	showJSON    bool
	finalOutput string // 儲存最終 JSON 避免 View 被重複計算

	inputMode mode
	textInput textinput.Model
	selectedW int
	selectedL int
}

// --- 初始化與邏輯 ---

func initialModel(layout []int) model {
	ti := textinput.New()
	ti.Placeholder = "輸入數字..."
	ti.Focus()
	ti.CharLimit = 5
	ti.SetWidth(10)
	// layout := []int{3, 4, 5, 5, 4, 3}
	grid := make([][]*Symbol, len(layout))
	for x, rows := range layout {
		grid[x] = make([]*Symbol, rows)
		for y := range rows {
			grid[x][y] = &Symbol{ID: 92, BT: 0, X: x, Y: y}
		}
	}

	return model{
		layout:     layout,
		grid:       grid,
		ids:        []int32{0, 1, 2, 3, 4, 11, 12, 13, 14, 91, 92},
		selectedID: 0,
		textInput:  ti,
		selectedW:  1,
		selectedL:  1,
		inputMode:  modeNormal,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.inputMode != modeNormal {
		var cmd tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				val, _ := strconv.Atoi(m.textInput.Value())
				switch m.inputMode {
				case modeInputID:
					m.selectedID = int32(val)
				case modeInputBT:
					m.selectedBT = val
				case modeInputW:
					m.selectedW = val
				case modeInputL:
					m.selectedL = val
				}
				m.inputMode = modeNormal
				m.textInput.Blur()
				m.textInput.Reset()
				return m, nil
			case "esc":
				m.inputMode = modeNormal
				m.textInput.Blur()
				m.textInput.Reset()
				return m, nil
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			m.inputMode = modeInputID
			return m, m.textInput.Focus()
		case "b":
			m.inputMode = modeInputBT
			return m, m.textInput.Focus()
		case "w":
			m.inputMode = modeInputW
			return m, m.textInput.Focus()
		case "l":
			m.inputMode = modeInputL
			return m, m.textInput.Focus()
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up":
			if m.cursorY > 0 {
				m.cursorY--
			}
		case "down":
			if m.cursorY < m.layout[m.cursorX]-1 {
				m.cursorY++
			}
		case "left":
			if m.cursorX > 0 {
				m.cursorX--
				if m.cursorY >= m.layout[m.cursorX] {
					m.cursorY = m.layout[m.cursorX] - 1
				}
			}
		case "right":
			if m.cursorX < len(m.layout)-1 {
				m.cursorX++
				if m.cursorY >= m.layout[m.cursorX] {
					m.cursorY = m.layout[m.cursorX] - 1
				}
			}
		case "enter":
			m.showJSON = true
			m.finalOutput = m.generateJSON()
			return m, tea.Quit
		}
		switch msg.Key().Code {
		case tea.KeySpace:
			target := m.grid[m.cursorX][m.cursorY]
			target.ID = m.selectedID
			target.BT = m.selectedBT
			target.Width = m.selectedW
			target.Length = m.selectedL
		}
	}
	return m, nil
}

func (m model) generateJSON() string {
	var finalGrid [][][]*SymbolOut
	reelData := make([][]*SymbolOut, len(m.layout))
	idxCounter := 0
	for x, col := range m.grid {
		reelData[x] = make([]*SymbolOut, len(col))
		for y, s := range col {
			if s.ID != 0 {
				reelData[x][y] = &SymbolOut{
					ID:      s.ID,
					BT:      s.BT,
					Length:  s.Length,
					Width:   s.Width,
					Index:   int32(idxCounter),
					Index2D: Idx2D{X: x, Y: y},
				}
			}
			idxCounter++
		}
	}
	finalGrid = append(finalGrid, reelData)
	output := map[string]any{"main_game": finalGrid}
	res, _ := json.Marshal(output)
	return string(res)
}

func (m model) View() tea.View {
	if m.showJSON {
		return tea.NewView("正在產出 JSON 並複製到剪貼簿...\n")
	}

	var head string
	if m.inputMode != modeNormal {
		var title string
		switch m.inputMode {
		case modeInputID:
			title = "修改選定 ID"
		case modeInputBT:
			title = "修改選定 BorderType"
		case modeInputW:
			title = "修改選定 Width"
		case modeInputL:
			title = "修改選定 Length"
		}
		head = lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Render(fmt.Sprintf("👉 %s: ", title)) + m.textInput.View()
	} else {
		head = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Render("Slot Machine Editor") + "\n"
		head = fmt.Sprintf("Template: ID [%d] BT [%d] W [%d] L [%d]\n",
			m.selectedID, m.selectedBT, m.selectedW, m.selectedL)
		head += "快捷鍵: [s]ID [b]BT [w]Width [l]Length | [Space]填入 [Enter]產出"
	}

	var columns []string
	for x := 0; x < len(m.layout); x++ {
		var colCells []string
		for y := 0; y < m.layout[x]; y++ {
			cellContent := fmt.Sprintf("%d", m.grid[x][y].ID)
			style := lipgloss.NewStyle().Width(6).Align(lipgloss.Center).Border(lipgloss.NormalBorder())

			// 根據 BorderType (BT) 決定邊框顏色
			switch m.grid[x][y].BT {
			case 1: // 銀色
				style = style.BorderForeground(lipgloss.Color("250")) // Light Gray
			case 2: // 金色
				style = style.BorderForeground(lipgloss.Color("214")) // Gold/Orange
			default: // 0: 正常 (預設顏色)
				style = style.BorderForeground(lipgloss.Color("240")) // Dark Gray
			}

			// 如果是游標所在位置，覆蓋背景色
			if x == m.cursorX && y == m.cursorY {
				style = style.Background(lipgloss.Color("201")).Foreground(lipgloss.Color("230"))
			}
			colCells = append(colCells, style.Render(cellContent))
		}
		columns = append(columns, lipgloss.JoinVertical(lipgloss.Left, colCells...))
	}
	s := lipgloss.JoinHorizontal(lipgloss.Top, columns...)
	return tea.NewView(head + "\n\n" + s)
}

func main() {
	layoutFlag := flag.String("layout", "3,4,5,5,4,3", "輸入盤面佈局，例如: 3,4,5,5,4,3")
	flag.Parse()

	sList := strings.Split(*layoutFlag, ",")
	var customLayout []int
	for _, s := range sList {
		v, err := strconv.Atoi(strings.TrimSpace(s))
		if err == nil {
			customLayout = append(customLayout, v)
		}
	}

	p := tea.NewProgram(initialModel(customLayout))
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(model); ok && m.showJSON {
		// 1. 複製到剪貼簿
		err := clipboard.WriteAll(m.finalOutput)
		if err != nil {
			fmt.Println("❌ 剪貼簿複製失敗")
		}

		// 2. 寫入檔案 (備份)
		_ = os.WriteFile("debug_command.json", []byte(m.finalOutput), 0644)

		// 3. 輸出到 Terminal (即使被截斷也沒關係，因為剪貼簿已經有了)
		fmt.Println("\n✨ JSON 已產出！")
		fmt.Println("✅ 已自動複製到剪貼簿 (Clipboard)")
		fmt.Println("💾 已儲存至 debug_command.json")
		fmt.Println("---------------------------------------")
		fmt.Println(m.finalOutput)
	}
}
