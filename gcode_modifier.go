package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var lastZHeight float64 = -1 // Track the last significant Z height
var mapLayerLines map[int]int
var mapSupportOnlyLayers map[int]bool

const (
	MIN_PREV_PERIM            = 10.0
	PERIM_PCT_CHG_UPPER       = -50.0
	PERIM_PCT_CHG_LOWER       = -95.0
	MIN_PROB_LAYER            = 20 // Ignore "problematic" layers below this
	FAN_SPEED_PCT_PROB_LAYERS = 1  // Percent
	TEMP_INCREASE_PROB_LAYERS = 20 // Celcius
)

func main() {
	// Define command-line flags
	inputFilePath := flag.String("f", "", "Path to the input G-code file")
	dirPath := flag.String("d", "", "Path directory of G-code files")
	overwrite := flag.Bool("o", false, "Overwrite existing G-code file (Default=false)")

	flag.Parse()

	if *inputFilePath == "" && *dirPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *dirPath != "" {
		filepath.WalkDir(*dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() && strings.HasSuffix(d.Name(), ".gcode") && !strings.HasSuffix(d.Name(), "_modified.gcode") {
				fullPath := filepath.Join(*dirPath, d.Name())
				processFile(fullPath, *overwrite)
				fmt.Println(fullPath)
			}
			return nil
		})
	}

	if *inputFilePath == "" {
		processFile(*inputFilePath, *overwrite)
		fmt.Println(*inputFilePath)
	}
}

func processFile(filePath string, overwrite bool) {
	// Read the input file
	fmt.Printf("Processing '%s'\n", filePath)
	inputFile, err := os.Open(filePath)
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

	layerCount := countLayers(lines)
	fmt.Printf("File '%s' has %d layers\n", filePath, layerCount)

	mapLayerLines = getMapOfLayerStartLines(lines)
	// fmt.Printf("mapLayerLines: %+v\n", mapLayerLines)

	mapSupportOnlyLayers = getMapOfSupportLayers(lines)
	// fmt.Printf("mapSupportOnlyLayers: %+v\n", mapSupportOnlyLayers)

	// Process the file based on the selected mode
	probLayers := detectProblematicLayers(lines)
	fmt.Printf("Problematic layers: %v\n", probLayers)

	defaultTemp := getDefaultTemp(lines)
	maxFanSpeed := getMaxFanSpeed(lines)
	for _, layer := range probLayers {
		// Decrease the fan speed & increase the temp for the layer below
		lines = modifyGcodeFanSpeed(lines, layer-3, FAN_SPEED_PCT_PROB_LAYERS)
		lines = modifyGcodeTemperature(lines, layer-3, defaultTemp+TEMP_INCREASE_PROB_LAYERS)

		// Reset the fan speed & temp for the layer above
		lines = modifyGcodeFanSpeed(lines, layer+2, maxFanSpeed)
		lines = modifyGcodeTemperature(lines, layer+2, defaultTemp)
	}

	// Save the modified lines to a new file
	outputFilePath := strings.Replace(filePath, ".gcode", "_modified.gcode", 1)
	if overwrite {
		outputFilePath = filePath
	}
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	for _, line := range lines {
		outputFile.WriteString(line + "\n")
	}

	fmt.Printf("Modification complete. New file saved as %s.\n", outputFilePath)
}

func detectLayerChange(line string) bool {
	if strings.HasPrefix(line, "; layer num/total_layer_count: ") {
		return true // Detect layer changes based on explicit comments like "; layer n"
	}
	return false
}

// extractZValue extracts the Z value from a G-code line
func extractZValue(line string) (float64, error) {
	fields := strings.Fields(line)
	for _, field := range fields {
		if field[0] == 'Z' {
			return strconv.ParseFloat(field[1:], 64)
		}
	}
	return 0, fmt.Errorf("Z value not found")
}

// calculateDistance calculates the distance between two points in the XY plane.
func calculateDistance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}

// getDefaultTemp gets the overall nozzle temp (e.g. "; nozzle_temperature = 235")
func getDefaultTemp(lines []string) int {
	var tempC = 0
	for _, line := range lines {
		// e.g. "; nozzle_temperature = 235"
		if strings.HasPrefix(line, "; nozzle_temperature = ") {
			strTemp := strings.Split(line, " = ")[1]
			tempC, _ = strconv.Atoi(strTemp)
			break
		}
	}
	return tempC
}

// getMaxFanSpeed gets the overall max cooling fan speed
func getMaxFanSpeed(lines []string) int {
	var fanSpeed = 0
	for _, line := range lines {
		// e.g. "; fan_max_speed = 100"
		if strings.HasPrefix(line, "; fan_max_speed = ") {
			strTemp := strings.Split(line, " = ")[1]
			fanSpeed, _ = strconv.Atoi(strTemp)
			break
		}
	}
	return fanSpeed
}

// countLayers uses detectLayerChange() to count the layers
func countLayers(lines []string) int {
	var count = 0
	for _, line := range lines {
		if detectLayerChange(line) {
			count++
		}
	}
	return count
}

// getMapOfSupportLayers returns a map of layer number and true/false
func getMapOfSupportLayers(lines []string) map[int]bool {
	mapSupportOnlyLayers = make(map[int]bool)
	currentLayer := 0
	hasOtherFeature := false
	for _, line := range lines {
		if detectLayerChange(line) {
			if hasOtherFeature {
				// Previous layer had a non-support feature
				mapSupportOnlyLayers[currentLayer] = false
			}

			currentLayer++
			// reset var hasOtherFeature
			hasOtherFeature = false
			mapSupportOnlyLayers[currentLayer] = false
		} else if strings.HasPrefix(line, "; FEATURE:") && !strings.Contains(line, "Support") {
			hasOtherFeature = true
		} else if line == "; FEATURE: Support" {
			mapSupportOnlyLayers[currentLayer] = true
		}
	}
	return mapSupportOnlyLayers
}

// getMapOfLayerStartLines returns a map of layer number to the line in the gcode file where that layer begins
func getMapOfLayerStartLines(lines []string) map[int]int {
	mapLayerLines = make(map[int]int)
	currentLayer := 0
	for i, line := range lines {
		if detectLayerChange(line) {
			currentLayer++
			if currentLayer > 0 {
				mapLayerLines[currentLayer] = i + 1
			}
		}
	}
	return mapLayerLines
}

// modifyGcodeTemperature modifies the hotend temperature at a specific layer using improved layer detection.
func modifyGcodeTemperature(lines []string, layerNumber int, temperature int) []string {
	modifiedLines := []string{}
	currentLayer := -1

	for _, line := range lines {
		modifiedLines = append(modifiedLines, line)
		if detectLayerChange(line) {
			currentLayer++
			if currentLayer == layerNumber {
				modifiedLines = append(modifiedLines, fmt.Sprintf("M104 S%d ; Set hotend temperature to %d°C at layer %d\n", temperature, temperature, layerNumber))
			}
		}
	}
	return modifiedLines
}

// modifyGcodeFanSpeed modifies the fan speed at a specific layer using improved layer detection.
func modifyGcodeFanSpeed(lines []string, layerNumber int, fanSpeedPercent int) []string {
	fanSpeedValue := int(float64(fanSpeedPercent) / 100.0 * 255)
	modifiedLines := []string{}
	currentLayer := -1

	for _, line := range lines {
		modifiedLines = append(modifiedLines, line)
		if detectLayerChange(line) {
			currentLayer++
			if currentLayer == layerNumber {
				modifiedLines = append(modifiedLines, fmt.Sprintf("M106 S%d ; Set fan speed to %d%% at layer %d\n", fanSpeedValue, fanSpeedPercent, layerNumber))
			}
		}
	}
	return modifiedLines
}

func detectProblematicLayers(lines []string) []int {
	currentLayer := -1
	previousPerimeterLength := 0.0
	currentPerimeterLength := 0.0
	problematicLayers := []int{}
	var lastX, lastY float64
	extruding := false

	for _, line := range lines {
		if detectLayerChange(line) {
			currentLayer++
			// Analyze conditions to detect problematic layers
			if currentLayer > 1 {
				absolutePerimeterChange := currentPerimeterLength - previousPerimeterLength
				perimeterPercentageChange := absolutePerimeterChange / previousPerimeterLength * 100

				if perimeterPercentageChange < PERIM_PCT_CHG_UPPER && perimeterPercentageChange > PERIM_PCT_CHG_LOWER && currentPerimeterLength > 80 {
					// Only add non-support layers and layers above MIN_PROB_LAYER
					if currentLayer > MIN_PROB_LAYER && !mapSupportOnlyLayers[currentLayer] {
						problematicLayers = append(problematicLayers, currentLayer)
					}
				}
				// if mapSupportOnlyLayers[currentLayer] {
				// 	fmt.Printf("Layer %d has length %f (chg %d%%) SUPPORT ONLY\n", currentLayer, currentPerimeterLength, int(perimeterPercentageChange))
				// } else {
				// 	fmt.Printf("Layer %d has length %f (chg %d%%)\n", currentLayer, currentPerimeterLength, int(perimeterPercentageChange))
				// }

			}
			//  else if currentLayer >= 1 {
			// 	fmt.Printf("Layer %d has length %f\n", currentLayer, currentPerimeterLength)
			// }

			// Reset values for the new layer
			previousPerimeterLength = currentPerimeterLength
			currentPerimeterLength = 0.0
		} else if strings.HasPrefix(line, "G1") {
			// Extract X, Y, and E values from the G-code line
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

			// Calculate perimeter length and extrusion volume
			if hasX && hasY {
				if extruding {
					currentPerimeterLength += calculateDistance(lastX, lastY, x, y)
				}
				extruding = true
				lastX, lastY = x, y
			}
		}
	}

	return problematicLayers
}
