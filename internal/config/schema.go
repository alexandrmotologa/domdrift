package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// FieldDefinition specifies how an attribute is located and validated.
type FieldDefinition struct {
	Name            string         `json:"name"`
	Type            string         `json:"type"` // string, number, boolean, array
	Description     string         `json:"description,omitempty"`
	SemanticRole    string         `json:"semanticRole,omitempty"`
	Pattern         string         `json:"pattern,omitempty"`
	InitialSelector string         `json:"initialSelector,omitempty"`
	Required        bool           `json:"required"`
	compiledRegex   *regexp.Regexp `json:"-"`
}

// ExtractionSchema represents a target structured entity to extract.
type ExtractionSchema struct {
	Title       string                     `json:"title"`
	Description string                     `json:"description,omitempty"`
	Type        string                     `json:"type"`
	Required    []string                   `json:"required,omitempty"`
	Properties  map[string]FieldDefinition `json:"properties"`
	Fields      []FieldDefinition          `json:"-"` // Ordered list
}

// LoadSchema reads and validates an extraction schema file.
func LoadSchema(path string) (*ExtractionSchema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed reading schema file %s: %w", path, err)
	}

	var raw struct {
		Title       string                     `json:"title"`
		Description string                     `json:"description,omitempty"`
		Type        string                     `json:"type"`
		Required    []string                   `json:"required,omitempty"`
		Properties  map[string]FieldDefinition `json:"properties"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed parsing schema json: %w", err)
	}

	requiredMap := make(map[string]bool)
	for _, req := range raw.Required {
		requiredMap[req] = true
	}

	schema := &ExtractionSchema{
		Title:       raw.Title,
		Description: raw.Description,
		Type:        raw.Type,
		Required:    raw.Required,
		Properties:  make(map[string]FieldDefinition),
	}

	for name, field := range raw.Properties {
		field.Name = name
		if requiredMap[name] {
			field.Required = true
		}
		if field.Type == "" {
			field.Type = "string"
		}
		if field.Pattern != "" {
			re, err := regexp.Compile(field.Pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid regex pattern %q for field %s: %w", field.Pattern, name, err)
			}
			field.compiledRegex = re
		}
		schema.Properties[name] = field
		schema.Fields = append(schema.Fields, field)
	}

	return schema, nil
}

// ValidateField checks whether an extracted raw value matches schema constraints.
func (s *ExtractionSchema) ValidateField(fieldName string, value string) error {
	field, ok := s.Properties[fieldName]
	if !ok {
		return fmt.Errorf("field %s not defined in schema", fieldName)
	}

	trimmed := strings.TrimSpace(value)
	if field.Required && trimmed == "" {
		return fmt.Errorf("required field %s is empty", fieldName)
	}

	if trimmed != "" && field.compiledRegex != nil {
		if !field.compiledRegex.MatchString(trimmed) {
			return fmt.Errorf("value %q for field %s does not match pattern %s", trimmed, fieldName, field.Pattern)
		}
	}

	return nil
}
