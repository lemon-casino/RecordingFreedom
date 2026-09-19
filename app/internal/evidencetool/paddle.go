package evidencetool

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ExtractPaddleOCRCharacterDict parses the PaddleOCR inference.yml
// PostProcess.character_dict sequence.
// Source: cmd/ocr-model-package, cmd/ocr-model-source-audit.
func ExtractPaddleOCRCharacterDict(data []byte) ([]string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse PaddleOCR inference.yml: %w", err)
	}
	postProcess := YamlMappingValue(&root, "PostProcess")
	if postProcess == nil {
		return nil, errors.New("PaddleOCR inference.yml missing PostProcess")
	}
	dict := YamlMappingValue(postProcess, "character_dict")
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

// YamlMappingValue returns the value node mapped to key in node, unwrapping a
// document node, or nil when node is not a mapping containing key.
// Source: cmd/ocr-model-package, cmd/ocr-model-source-audit.
func YamlMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return YamlMappingValue(node.Content[0], key)
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Kind == yaml.ScalarNode && node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
