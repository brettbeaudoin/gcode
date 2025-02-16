package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// detectLayerChange determines if a line indicates a layer change.
func detectLayerChange(line string) bool {
	return strings.Contains(line, "; layer num/total_layer_count") || strings.HasPrefix(line, "M73 L") || strings.HasPrefix(line, "G1 Z")
}

// calculateDistance calculates the distance between two points in the XY plane.
func calculateDistance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}

func analyzeGcode(lines []string) {
	currentLayer := -1
	previousPerimeterLength := 0.0
	currentPerimeterLength := 0.0
	var lastX, lastY float64
	extruding := false
	for lineNumber, line := range lines {
		if detectLayerChange(line) {
			if currentLayer >= 0 && previousPerimeterLength > 50.0 && math.Abs(currentPerimeterLength-previousPerimeterLength)/previousPerimeterLength > 0.3 {
				fmt.Printf("Line %d: Suggestion: Increase temperature by 10°C to improve adhesion at layer %d\n", lineNumber+1, currentLayer)
			}
			previousPerimeterLength = currentPerimeterLength
			currentPerimeterLength = 0.0
			currentLayer++
		} else if strings.HasPrefix(line, "G1") {
			// Extract X, Y values and calculate perimeter length
			var x, y float64
			var hasX, hasY bool
			fields := strings.Fields(line)
			for _, field := range fields {
				switch field[0] {
				case 'X':
					x, _ = strconv.ParseFloat(field[1:], 64)
					hasX = true
				case 'Y':
					y, _ = strconv.ParseFloat(field[1:], 64)
					hasY = true
				}
			}
			if hasX && hasY {
				if extruding {
					currentPerimeterLength += calculateDistance(lastX, lastY, x, y)
				}
				extruding = true
				lastX, lastY = x, y
			}
		}
	}
}

func main() {
	// Define command-line flags
	inputFilePath := flag.String("file", "", "Path to the input G-code file")
	flag.Parse()

	if *inputFilePath == "" {
		fmt.Println("Usage: -file <G-code file>\n")
		os.Exit(1)
	}

	// Read the input file
	inputFile, err := os.Open(*inputFilePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer inputFile.Close()

	var lines []string
	scanner := bufio.NewScanner(inputFile)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Analyze the G-code and provide suggestions
	fmt.Println("Analyzing G-code for potential adhesion problems...\n")
	analyzeGcode(lines)
	fmt.Println("Analysis complete. Suggestions provided above.\n")
}
