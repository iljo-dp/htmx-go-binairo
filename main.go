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
	Value    *int `json:"value"`
	ReadOnly bool `json:"readonly"` // Added readonly flag
}

// Grid represents the puzzle grid.
type Grid [][]*Cell

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/solve", solveHandler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/validate", validateHandler)
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// indexHandler serves the main page.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, nil)
}

// solveHandler receives the grid, solves the puzzle, and returns the solution.
func solveHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Solve request received")
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v\n", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	gridSizeStr := r.FormValue("gridSize")
	size, err := strconv.Atoi(gridSizeStr)
	if err != nil || size <= 0 {
		log.Printf("Invalid grid size: %v\n", err)
		http.Error(w, "Invalid grid size", http.StatusBadRequest)
		return
	}

	log.Printf("Grid size: %d\n", size)

	grid := make(Grid, size)
	for i := range grid {
		grid[i] = make([]*Cell, size)
		for j := range grid[i] {
			cellName := fmt.Sprintf("cell-%d-%d", i, j)
			cellValue := r.FormValue(cellName)
			if cellValue == "" {
				grid[i][j] = &Cell{Value: nil}
			} else {
				val, err := strconv.Atoi(cellValue)
				if err != nil {
					log.Printf("Invalid cell value %s: %v\n", cellValue, err)
					http.Error(w, "Invalid cell value", http.StatusBadRequest)
					return
				}
				grid[i][j] = &Cell{Value: &val, ReadOnly: true}
			}
		}
	}

	log.Println("Grid received, solving...")
	solvedGrid := solveBinairo(grid)
	response := renderSolvedGridHTML(solvedGrid)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, response)
	log.Println("Puzzle solved, response sent")
}

// generateHandler generates a valid puzzle and removes some cells for the user to solve.
func generateHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Generate request received")
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v\n", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	gridSizeStr := r.FormValue("gridSize")
	size, err := strconv.Atoi(gridSizeStr)
	if err != nil || size <= 0 {
		log.Printf("Invalid grid size: %v\n", err)
		http.Error(w, "Invalid grid size", http.StatusBadRequest)
		return
	}

	grid := make(Grid, size)
	for i := range grid {
		grid[i] = make([]*Cell, size)
		for j := range grid[i] {
			grid[i][j] = &Cell{Value: nil}
		}
	}

	// Solve the grid first
	solveBinairo(grid)

	// Remove some cells to create the puzzle
	rand.Seed(time.Now().UnixNano())
	numCellsToRemove := (size * size) / 2
	for numCellsToRemove > 0 {
		i, j := rand.Intn(size), rand.Intn(size)
		if grid[i][j].Value != nil {
			grid[i][j].Value = nil      // Empty the cell for user to fill
			grid[i][j].ReadOnly = false // Set as editable
			numCellsToRemove--
		}
	}

	// Mark the remaining cells as readonly (pre-filled)
	for i := range grid {
		for j := range grid[i] {
			if grid[i][j].Value != nil {
				grid[i][j].ReadOnly = true // Pre-filled cells are readonly
			}
		}
	}

	response := renderSolvedGridHTML(grid)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, response)
	log.Println("Puzzle generated, response sent")
}

// validateHandler checks if the user's solution is valid.
func validateHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Validate request received")

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v\n", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Get and validate grid size
	gridSizeStr := r.FormValue("gridSize")
	size, err := strconv.Atoi(gridSizeStr)
	if err != nil || size <= 0 {
		log.Printf("Invalid grid size: %v\n", err)
		http.Error(w, "Invalid grid size", http.StatusBadRequest)
		return
	}

	log.Printf("Grid size: %d\n", size)

	grid := make(Grid, size)
	for i := range grid {
		grid[i] = make([]*Cell, size)
		for j := range grid[i] {
			cellName := fmt.Sprintf("cell-%d-%d", i, j)
			cellValue := r.FormValue(cellName)
			if cellValue == "" {
				log.Println("Grid is not fully filled in")
				fmt.Fprintln(w, "Grid is not fully filled in")
				return
			}
			val, err := strconv.Atoi(cellValue)
			if err != nil {
				log.Printf("Invalid cell value %s: %v\n", cellValue, err)
				http.Error(w, "Invalid cell value", http.StatusBadRequest)
				return
			}
			grid[i][j] = &Cell{Value: &val}
		}
	}

	// Validate the grid
	valid := true
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if !isValid(grid, i, j, *grid[i][j].Value) {
				valid = false
				break
			}
		}
		if !valid {
			break
		}
	}

	if valid {
		log.Println("Grid is valid")
		fmt.Fprintln(w, "valid")
	} else {
		log.Println("Grid is invalid")
		fmt.Fprintln(w, "invalid")
	}
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

// renderSolvedGridHTML generates the HTML for the solved grid.
func renderSolvedGridHTML(grid Grid) string {
	var sb strings.Builder
	size := len(grid)

	sb.WriteString(`
		<div id="solved-grid-container" class="grid-container" style="grid-template-columns: repeat(` + strconv.Itoa(size) + `, 1fr);">`)

	for _, row := range grid {
		for _, cell := range row {
			sb.WriteString(`<div class="grid-cell">`)
			if cell.Value != nil {
				if cell.ReadOnly {
					style := "readonly style='background-color: #ffcccc;'"
					sb.WriteString(fmt.Sprintf(`<input type="text" %s value="%d" />`, style, *cell.Value))
				} else {
					sb.WriteString(fmt.Sprintf(`<input type="text" value="%d" />`, *cell.Value))
				}
			} else {
				sb.WriteString(`<input type="text" value="" />`)
			}
			sb.WriteString(`</div>`)
		}
	}

	sb.WriteString(`</div>`)
	return sb.String()
}
