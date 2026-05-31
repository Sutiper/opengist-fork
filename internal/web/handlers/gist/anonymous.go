package gist

import (
	"github.com/thomiceli/opengist/internal/db"
	"github.com/thomiceli/opengist/internal/web/context"
)

// loadAnonGist loads an anonymous gist by edit token and restores the virtual
// "anonymous" user needed for git operations (UserID is nil in DB).
func loadAnonGist(token string) (*db.Gist, error) {
	gist, err := db.GetGistByEditToken(token)
	if err != nil {
		return nil, err
	}
	gist.User = db.User{Username: "anonymous"}
	return gist, nil
}

// AnonConfirm shows the confirmation page after anonymous gist creation.
func AnonConfirm(ctx *context.Context) error {
	token := ctx.Param("token")
	if token == "" {
		return ctx.NotFound("Token not found")
	}

	gist, err := loadAnonGist(token)
	if err != nil {
		return ctx.NotFound("Gist not found")
	}

	ctx.SetData("gist", gist)
	ctx.SetData("token", token)
	ctx.SetData("htmlTitle", ctx.TrH("gist.anon.confirmation"))
	return ctx.Html("anon_confirm.html")
}

// AnonEdit renders the edit form for an anonymous gist identified by its token.
func AnonEdit(ctx *context.Context) error {
	token := ctx.Param("token")
	if token == "" {
		return ctx.NotFound("Token not found")
	}

	gist, err := loadAnonGist(token)
	if err != nil {
		return ctx.NotFound("Gist not found")
	}

	gistDto, err := gist.ToDTO()
	if err != nil {
		return ctx.ErrorRes(500, "Error getting gist data", err)
	}

	files, _, err := gist.Files("HEAD", false)
	if err != nil {
		return ctx.ErrorRes(500, "Error fetching files", err)
	}

	ctx.SetData("gist", gist)
	ctx.SetData("dto", gistDto)
	ctx.SetData("files", files)
	ctx.SetData("token", token)
	ctx.SetData("htmlTitle", ctx.TrH("gist.edit.edit-gist", gist.Title))
	return ctx.Html("edit.html")
}

// AnonProcessEdit handles POST for anonymous gist editing.
func AnonProcessEdit(ctx *context.Context) error {
	token := ctx.Param("token")
	if token == "" {
		return ctx.NotFound("Token not found")
	}

	gist, err := loadAnonGist(token)
	if err != nil {
		return ctx.NotFound("Gist not found")
	}

	ctx.SetData("gist", gist)
	ctx.SetData("token", token)
	return ProcessCreate(ctx)
}

// AnonDelete renders the delete confirmation page for an anonymous gist.
func AnonDelete(ctx *context.Context) error {
	token := ctx.Param("token")
	if token == "" {
		return ctx.NotFound("Token not found")
	}

	gist, err := loadAnonGist(token)
	if err != nil {
		return ctx.NotFound("Gist not found")
	}

	ctx.SetData("gist", gist)
	ctx.SetData("token", token)
	ctx.SetData("htmlTitle", ctx.TrH("gist.anon.delete-confirm"))
	return ctx.Html("anon_delete.html")
}

// AnonProcessDelete handles POST to actually delete an anonymous gist by token.
func AnonProcessDelete(ctx *context.Context) error {
	token := ctx.Param("token")
	if token == "" {
		return ctx.NotFound("Token not found")
	}

	gist, err := loadAnonGist(token)
	if err != nil {
		return ctx.NotFound("Gist not found")
	}

	if err := gist.Delete(); err != nil {
		return ctx.ErrorRes(500, "Error deleting this gist", err)
	}
	gist.RemoveFromIndex()

	ctx.AddFlash(ctx.Tr("flash.gist.deleted"), "success")
	return ctx.RedirectTo("/")
}
