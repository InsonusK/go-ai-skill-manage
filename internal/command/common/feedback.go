package common

import (
	"path/filepath"
	"time"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// FeedbackDraftsDir is where feedback drafts lie, from the project's root
// (the config file's folder). They are meant to be committed.
const FeedbackDraftsDir = ".ai-skills/feedback"

// FeedbackService is the feedback service of req's project and its drafts
// folder -- the same for the CLI and the MCP server.
func (a *App) FeedbackService(req model.Request) (handler.FeedbackService, string) {
	now := a.Now
	if now == nil {
		now = time.Now
	}
	dir := filepath.Join(req.Base, FeedbackDraftsDir)
	return handler.FeedbackService{Markers: a.Markers, Drafts: a.Drafts(dir), Trackers: a.Trackers, Now: now, Version: a.Version}, dir
}
