package entity

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

type SkillDocument struct {
	Properties     map[string]any
	Body           string
	HasFrontmatter bool
}

func MakeSkillDocument(data []byte) (SkillDocument, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	doc := SkillDocument{Body: text}
	if !strings.HasPrefix(text, "---\n") {
		return doc, nil
	}
	end := -1
	offset := 4
	for _, line := range strings.Split(text[4:], "\n") {
		if strings.TrimSpace(line) == "---" {
			end = offset
			break
		}
		offset += len(line) + 1
	}
	if end < 0 {
		return doc, fmt.Errorf("frontmatter: missing closing delimiter")
	}
	var props map[string]any
	if err := yaml.Unmarshal([]byte(text[4:end]), &props); err != nil {
		return doc, fmt.Errorf("frontmatter: %w", err)
	}
	tail := text[end:]
	newline := strings.IndexByte(tail, '\n')
	body := ""
	if newline >= 0 {
		body = tail[newline+1:]
	}
	if props == nil {
		props = map[string]any{}
	}
	return SkillDocument{Properties: props, Body: body, HasFrontmatter: true}, nil
}
func (doc *SkillDocument) Encode() ([]byte, error) {
	if !doc.HasFrontmatter {
		return []byte(doc.Body), nil
	}
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(doc.Properties); err != nil {
		return nil, fmt.Errorf("frontmatter: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	body := doc.Body
	return []byte("---\n" + b.String() + "---\n" + body), nil
}
