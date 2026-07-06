package ocr

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func generatePaddleOCRCharacterDictKeys(data []byte) ([]byte, error) {
	characters, err := extractPaddleOCRCharacterDict(data)
	if err != nil {
		return nil, err
	}
	if len(characters) == 0 {
		return nil, errors.New("PaddleOCR inference.yml character_dict is empty")
	}
	for _, character := range characters {
		if strings.ContainsAny(character, "\r\n") {
			return nil, fmt.Errorf("PaddleOCR character_dict item contains newline: %q", character)
		}
	}
	return []byte(strings.Join(characters, "\n") + "\n"), nil
}

func extractPaddleOCRCharacterDict(data []byte) ([]string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse PaddleOCR inference.yml: %w", err)
	}
	postProcess := yamlMappingValue(&root, "PostProcess")
	if postProcess == nil {
		return nil, errors.New("PaddleOCR inference.yml missing PostProcess")
	}
	dict := yamlMappingValue(postProcess, "character_dict")
	if dict == nil {
		return nil, errors.New("PaddleOCR inference.yml missing PostProcess.character_dict")
	}
	if dict.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("PaddleOCR PostProcess.character_dict kind = %v, want sequence", dict.Kind)
	}
	characters := make([]string, 0, len(dict.Content))
	for _, item := range dict.Content {
		if item.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("PaddleOCR character_dict item kind = %v, want scalar", item.Kind)
		}
		characters = append(characters, item.Value)
	}
	return characters, nil
}

func yamlMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return yamlMappingValue(node.Content[0], key)
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Kind == yaml.ScalarNode && node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}
