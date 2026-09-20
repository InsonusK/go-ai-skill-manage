package planning

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"sort"
)

func Fingerprint(files []model.OutputFile) string {
	ordered := append([]model.OutputFile{}, files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	h := sha256.New()
	for _, f := range ordered {
		fmt.Fprintf(h, "%d:%s:%d:%d:", len(f.Path), f.Path, f.Mode, len(f.Data))
		h.Write(f.Data)
	}
	return hex.EncodeToString(h.Sum(nil))
}
