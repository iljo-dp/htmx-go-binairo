package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Cell represents a single cell in the grid.
type Cell struct {
	Value    *int // nil if the cell is empty
	ReadOnly bool // true if the cell is read-only (pre-filled)
}

// Grid represents the puzzle grid.
type Grid [][]*Cell

// Main function
func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/solve", solveHandler)
	http.HandleFunc("/validate", validateHandler)
	http.HandleFunc("/toggleCell", toggleCellHandler)
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// indexHandler serves the main page.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, nil)
}

// generateHandler generates a valid puzzle and removes some cells for the user to solve.
func generateHandler(w http.ResponseWriter, r *http.Request) {
	gridSizeStr := r.FormValue("gridSize")
	size, err := strconv.Atoi(gridSizeStr)
	if err != nil || size <= 0 {
		http.Error(w, "Invalid grid size", http.StatusBadRequest)
		return
	}

	grid := createGrid(size)
	solveBinairo(grid) // Start with a solved grid

	// Remove some cells to create the puzzle
	rand.Seed(time.Now().UnixNano())
	numCellsToRemove := (size * size) / 2
	for numCellsToRemove > 0 {
		i, j := rand.Intn(size), rand.Intn(size)
		if grid[i][j].Value != nil {
			grid[i][j].Value = nil      // Empty the cell for the user to fill
			grid[i][j].ReadOnly = false // Set as editable
			numCellsToRemove--
		}
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, renderGridHTML(grid))
}

// toggleCellHandler toggles the value of a cell between "", "0", and "1".
func toggleCellHandler(w http.ResponseWriter, r *http.Request) {
	row, _ := strconv.Atoi(r.PostFormValue("row"))
	col, _ := strconv.Atoi(r.PostFormValue("col"))
	gridSizeStr := r.PostFormValue("gridSize")
	size, _ := strconv.Atoi(gridSizeStr)

	// In a real application, the grid would be stored in the session or a database.
	// Here, we'll just create a new grid to simulate this for simplicity.
	grid := createGrid(size)

	cell := grid[row][col]
	if cell.ReadOnly {
		return // Do not toggle if the cell is readonly
	}

	// Toggle the value
	if cell.Value == nil {
		val := 0
		cell.Value = &val
	} else if *cell.Value == 0 {
		val := 1
		cell.Value = &val
	} else {
		cell.Value = nil
	}

	w.Header().Set("Content-Type", "text/html")
	// Pass the grid size as the fourth argument
	fmt.Fprintln(w, renderCellHTML(cell, row, col, size))
}

// solveHandler receives the grid, solves the puzzle, and returns the solution.
// solveHandler receives the grid, solves the puzzle, and returns the solution.
func solveHandler(w http.ResponseWriter, r *http.Request) {
	grid, _, err := parseGridFromRequest(r) // Remove the unused 'size' variable
	if err != nil {
		http.Error(w, "Invalid grid data", http.StatusBadRequest)
		return
	}

	solvedGrid := solveBinairo(grid)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, renderGridHTML(solvedGrid))
}

// validateHandler checks if the user's solution is valid.
func validateHandler(w http.ResponseWriter, r *http.Request) {
	grid, size, err := parseGridFromRequest(r)
	if err != nil {
		http.Error(w, "Invalid grid data", http.StatusBadRequest)
		return
	}

	valid := validateBinairo(grid, size)
	if valid {
		fmt.Fprintln(w, "valid")
	} else {
		fmt.Fprintln(w, "invalid")
	}
}

// Helper functions

// createGrid creates an empty grid of the given size.
func createGrid(size int) Grid {
	grid := make(Grid, size)
	for i := range grid {
		grid[i] = make([]*Cell, size)
		for j := range grid[i] {
			grid[i][j] = &Cell{}
		}
	}
	return grid
}

// parseGridFromRequest parses the grid data from the request.
func parseGridFromRequest(r *http.Request) (Grid, int, error) {
	gridSizeStr := r.FormValue("gridSize")
	size, err := strconv.Atoi(gridSizeStr)
	if err != nil || size <= 0 {
		return nil, 0, fmt.Errorf("invalid grid size")
	}

	grid := createGrid(size)
	for i := range grid {
		for j := range grid[i] {
			cellName := fmt.Sprintf("cell-%d-%d", i, j)
			cellValue := r.FormValue(cellName)
			if cellValue != "" {
				val, err := strconv.Atoi(cellValue)
				if err != nil {
					return nil, 0, fmt.Errorf("invalid cell value")
				}
				grid[i][j].Value = &val
			}
		}
	}
	return grid, size, nil
}

// solveBinairo implements the Binairo solving algorithm.
func solveBinairo(grid Grid) Grid {
	size := len(grid)
	var solve func() bool
	solve = func() bool {
		for i := 0; i < size; i++ {
			for j := 0; j < size; j++ {
				if grid[i][j].Value == nil {
					for _, val := range []int{0, 1} {
						if isValid(grid, i, j, val) {
							grid[i][j].Value = &val
							if solve() {
								return true
							}
							grid[i][j].Value = nil
						}
					}
					return false
				}
			}
		}
		return true
	}
	solve()
	return grid
}

// validateBinairo checks if the current grid is a valid Binairo solution.
func validateBinairo(grid Grid, size int) bool {
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if grid[i][j].Value == nil || !isValid(grid, i, j, *grid[i][j].Value) {
				return false
			}
		}
	}
	return true
}

// isValid checks if placing a value at grid[row][col] is valid.
func isValid(grid Grid, row, col, value int) bool {
	size := len(grid)
	rowValues := make([]int, size)
	colValues := make([]int, size)

	for i := 0; i < size; i++ {
		if grid[row][i].Value != nil {
			rowValues[i] = *grid[row][i].Value
		} else {
			rowValues[i] = -1
		}
		if grid[i][col].Value != nil {
			colValues[i] = *grid[i][col].Value
		} else {
			colValues[i] = -1
		}
	}

	rowValues[col] = value
	colValues[row] = value

	if countOccurrences(rowValues, value) > size/2 || countOccurrences(colValues, value) > size/2 {
		return false
	}

	if (col > 1 && isSame(rowValues[col-2:col+1], []int{value, value, value})) ||
		(col < size-2 && isSame(rowValues[col:col+3], []int{value, value, value})) ||
		(row > 1 && isSame(colValues[row-2:row+1], []int{value, value, value})) ||
		(row < size-2 && isSame(colValues[row:row+3], []int{value, value, value})) {
		return false
	}

	return true
}

// countOccurrences counts the occurrences of a value in a slice.
func countOccurrences(slice []int, value int) int {
	count := 0
	for _, v := range slice {
		if v == value {
			count++
		}
	}
	return count
}

// isSame checks if two slices are equal in terms of elements and their order.
func isSame(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// renderGridHTML generates the HTML for the grid.
// renderGridHTML generates the HTML for the grid.
func renderGridHTML(grid Grid) string {
	var sb strings.Builder
	size := len(grid)

	sb.WriteString(`
		<div id="grid-container" class="grid-container" style="grid-template-columns: repeat(` + strconv.Itoa(size) + `, 1fr);">`)

	for i, row := range grid {
		for j, cell := range row {
			sb.WriteString(renderCellHTML(cell, i, j, size))
		}
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

// renderCellHTML generates the HTML for a single cell.
func renderCellHTML(cell *Cell, row, col int, gridSize int) string {
	var sb strings.Builder

	sb.WriteString(`<div class="grid-cell">`)
	if cell.Value != nil {
		sb.WriteString(fmt.Sprintf(`<input type="text" name="cell-%d-%d" value="%d"`, row, col, *cell.Value))
	} else {
		sb.WriteString(fmt.Sprintf(`<input type="text" name="cell-%d-%d" value=""`, row, col))
	}

	if cell.ReadOnly {
		sb.WriteString(` readonly="true"`)
	} else {
		sb.WriteString(fmt.Sprintf(` hx-post="/toggleCell" hx-vals='{"row":%d, "col":%d, "gridSize":"%d"}' hx-target="this" hx-swap="outerHTML"`, row, col, gridSize))
	}

	sb.WriteString(` maxlength="1" />`)
	sb.WriteString(`</div>`)

	return sb.String()
}
