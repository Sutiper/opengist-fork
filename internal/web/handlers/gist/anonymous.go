package gist

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/thomiceli/opengist/internal/db"
	"github.com/thomiceli/opengist/internal/git"
	"github.com/thomiceli/opengist/internal/i18n"
	"github.com/thomiceli/opengist/internal/validator"
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
// Does NOT reuse ProcessCreate to avoid isCreate confusion.
func AnonProcessEdit(ctx *context.Context) error {
	token := ctx.Param("token")
	if token == "" {
		return ctx.NotFound("Token not found")
	}

	gist, err := loadAnonGist(token)
	if err != nil {
		return ctx.NotFound("Gist not found")
	}

	dto := new(db.GistDTO)
	if err := ctx.Bind(dto); err != nil {
		return ctx.ErrorRes(400, ctx.Tr("error.cannot-bind-data"), err)
	}

	dto.Files = make([]db.FileDTO, 0)
	names := dto.Name
	contents := dto.Content

	for i, content := range contents {
		if content == "" {
			continue
		}
		name := git.CleanTreePathName(names[i])
		if name == "" {
			name = "gistfile" + strconv.Itoa(len(dto.Files)+1) + ".txt"
		}
		escapedValue, err := url.PathUnescape(content)
		if err != nil {
			return ctx.ErrorRes(400, ctx.Tr("error.invalid-character-unescaped"), err)
		}
		dto.Files = append(dto.Files, db.FileDTO{
			Filename: strings.TrimSpace(name),
			Content:  escapedValue,
		})
	}

	ctx.SetData("dto", dto)

	if err := ctx.Validate(dto); err != nil {
		ctx.AddFlash(validator.ValidationMessages(&err, ctx.GetData("locale").(*i18n.Locale)), "error")
		files, _, _ := gist.Files("HEAD", false)
		ctx.SetData("gist", gist)
		ctx.SetData("files", files)
		ctx.SetData("token", token)
		return ctx.HtmlWithCode(400, "edit.html")
	}

	gist = dto.ToExistingGist(gist)
	gist.User = db.User{Username: "anonymous"}
	gist.NbFiles = len(dto.Files)

	if gist.Title == "" {
		if len(dto.Files) > 0 && dto.Files[0].Filename != "" {
			gist.Title = dto.Files[0].Filename
		}
	}

	if err := gist.AddAndCommitFiles(&dto.Files); err != nil {
		return ctx.ErrorRes(500, "Error adding and committing files", err)
	}

	if err := gist.Update(); err != nil {
		return ctx.ErrorRes(500, "Error updating the gist", err)
	}

	gist.AddInIndex()
	gist.UpdateLanguages()
	if err := gist.UpdatePreviewAndCount(true); err != nil {
		return ctx.ErrorRes(500, "Error updating preview and count", err)
	}

	return ctx.RedirectTo("/anon/confirm/" + token)
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
