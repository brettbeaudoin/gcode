# G-code Modifier Utility

This is a command-line tool written in Go that modifies G-code files to change either the hotend temperature or the fan speed at a specific layer. The script generates a new G-code file with the requested modifications.

## Features
- Modify **hotend temperature** at a specific layer (`M104` command).
- Modify **fan speed** at a specific layer (`M106` command).
- Automatically saves a new G-code file with the changes.
- Command-line interface for ease of use.

## Installation

1. **Install Go** (if not installed):
   [Download Go](https://golang.org/dl/)
   
2. **Build the script**:
   ```sh
   go build gcode_modifier.go
   ```

## Usage

```sh
./gcode_modifier -file="path/to/your/file.gcode" -layer=<layer_number> -mode=<temperature|fanspeed> -value=<temperature_or_fan_speed>
```

### Parameters:
- `-file` : Path to the input G-code file (required).
- `-layer` : The layer number where the change should be applied (required).
- `-mode` : `temperature` or `fanspeed` (required).
- `-value` :
  - For `temperature` mode: Hotend temperature in °C (e.g., `210`).
  - For `fanspeed` mode: Fan speed as a percentage (0–100).

### Example Commands:

**Modify Hotend Temperature at Layer 50:**
```sh
./gcode_modifier -file="example.gcode" -layer=50 -mode="temperature" -value=210
```

**Modify Fan Speed to 75% at Layer 30:**
```sh
./gcode_modifier -file="example.gcode" -layer=30 -mode="fanspeed" -value=75
```

## Output
A new G-code file is generated with a modified filename, indicating the layer and change applied. For example:
- `example_layer50_temp210.gcode`
- `example_layer30_fan75.gcode`

## Error Handling
- If the specified layer is not found, the program will notify you and exit without modifying the file.
- Fan speed values are automatically constrained between 0 and 100%.

## License
This tool is open-source and available under the MIT License.

## Contributions
Contributions and improvements are welcome! Feel free to fork the repository and submit a pull request.
