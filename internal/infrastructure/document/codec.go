package document

import (
	"bytes"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"go.yaml.in/yaml/v3"
	"strings"
)

type Codec struct{}

func (Codec) Decode(data []byte) (model.Document, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	doc := model.Document{Body: text}
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
	return model.Document{Properties: props, Body: body, HasFrontmatter: true}, nil
}
func (Codec) Encode(doc model.Document) ([]byte, error) {
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
	if len(doc.Metadata) > 0 {
		extra, err := yaml.Marshal(doc.Metadata)
		if err != nil {
			return nil, err
		}
		body += "\n## Metadata\n\n```yaml\n" + string(extra) + "```\n"
	}
	return []byte("---\n" + b.String() + "---\n" + body), nil
}
