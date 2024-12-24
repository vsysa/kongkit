package template

import (
	"fmt"
	"reflect"
	"strings"
)

// FieldInfo represents a line in the generated YAML template.
type FieldInfo struct {
	Line string
	Help string
}

// GenerateYAMLTemplate generates a YAML template from a given configuration struct.
func GenerateYAMLTemplate(cfg interface{}) string {
	var lines []FieldInfo

	// First pass: Parse the structure
	parseStructure(reflect.TypeOf(cfg), reflect.ValueOf(cfg), 0, &lines)

	// Second pass: Generate aligned YAML
	return generateYAMLWithAlignment(lines)
}

// Recursively parses a structure to build YAML template lines.
func parseStructure(t reflect.Type, v reflect.Value, indent int, lines *[]FieldInfo) {
	indentation := strings.Repeat("  ", indent)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		tag := field.Tag

		// Handle ignored fields
		if tag.Get("kong") == "-" || tag.Get("yaml") == "-" {
			continue
		}

		// Determine the YAML key name
		fieldName := field.Name
		if tagName := tag.Get("yaml"); tagName != "" && tagName != "-" {
			fieldName = strings.Split(tagName, ",")[0]
		} else if tagName := tag.Get("kong"); tagName != "" && tagName != "-" {
			fieldName = tagName
		}
		fieldName = strings.ToLower(fieldName)

		defaultValue := tag.Get("default")
		if defaultValue == "" {
			defaultValue = tag.Get("placeholder")
		}
		helpText := tag.Get("help")

		switch field.Type.Kind() {
		case reflect.Struct:
			*lines = append(*lines, FieldInfo{
				Line: fmt.Sprintf("%s%s:", indentation, fieldName),
				Help: helpText,
			})
			parseStructure(field.Type, v.Field(i), indent+1, lines)

		case reflect.Slice:
			*lines = append(*lines, FieldInfo{
				Line: fmt.Sprintf("%s%s:", indentation, fieldName),
				Help: helpText,
			})

			// Handle array of structs
			if field.Type.Elem().Kind() == reflect.Struct {
				*lines = append(*lines, FieldInfo{
					Line: fmt.Sprintf("%s  -", indentation),
					Help: "",
				})
				// For anonymous structs or uninitialized fields, using v.Field(i) might result in invalid or zero values,
				// especially if the struct field hasn't been initialized yet. Instead, we use reflect.Zero(field.Type)
				// to create a zero value of the field's type. This ensures safe traversal and correct YAML generation
				// even when the struct is empty or contains anonymous sub-structs.
				parseStructure(field.Type.Elem(), reflect.Zero(field.Type.Elem()), indent+2, lines)
			} else {
				// Handle array of primitives
				if defaultValue != "" {
					defaultItems := strings.Split(defaultValue, ",")
					for _, item := range defaultItems {
						*lines = append(*lines, FieldInfo{
							Line: fmt.Sprintf("%s  - %s", indentation, strings.TrimSpace(item)),
							Help: "",
						})
					}
				} else {
					*lines = append(*lines, FieldInfo{
						Line: fmt.Sprintf("%s  - example", indentation),
						Help: "",
					})
				}
			}

		case reflect.Map:
			*lines = append(*lines, FieldInfo{
				Line: fmt.Sprintf("%s%s:", indentation, fieldName),
				Help: helpText,
			})
			*lines = append(*lines, FieldInfo{
				Line: fmt.Sprintf("%s  key: value", indentation),
				Help: "Map example",
			})

		default:
			value := defaultValue
			if value == "" {
				value = "null"
			}

			if field.Type.Kind() == reflect.String {
				value = fmt.Sprintf(`"%s"`, value)
			}

			*lines = append(*lines, FieldInfo{
				Line: fmt.Sprintf("%s%s: %s", indentation, fieldName, value),
				Help: helpText,
			})
		}
	}
}

// Aligns YAML lines with proper spacing for comments.
func generateYAMLWithAlignment(lines []FieldInfo) string {
	var builder strings.Builder
	maxLength := 0

	// Determine max line length (excluding comments)
	for _, line := range lines {
		if len(line.Line) > maxLength {
			maxLength = len(line.Line)
		}
	}

	// Generate aligned lines
	for _, line := range lines {
		builder.WriteString(line.Line)
		if line.Help != "" {
			spaces := strings.Repeat(" ", maxLength-len(line.Line)+1)
			builder.WriteString(spaces + "# " + line.Help)
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
