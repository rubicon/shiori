package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-shiori/shiori/internal/config"
	"github.com/go-shiori/shiori/internal/dependencies"
	"github.com/go-shiori/shiori/internal/model"
	"github.com/go-shiori/shiori/internal/webserver"
	"github.com/gofrs/uuid/v5"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type LegacyAPIRoutes struct {
	logger        *logrus.Logger
	cfg           *config.Config
	deps          *dependencies.Dependencies
	legacyHandler *webserver.Handler
}

func (r *LegacyAPIRoutes) convertHttprouteParams(params gin.Params) httprouter.Params {
	routerParams := httprouter.Params{}
	for _, p := range params {
		routerParams = append(routerParams, httprouter.Param{
			Key:   p.Key,
			Value: p.Value,
		})
	}
	return routerParams
}

func (r *LegacyAPIRoutes) handle(handler func(w http.ResponseWriter, r *http.Request, ps httprouter.Params)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		handler(ctx.Writer, ctx.Request, r.convertHttprouteParams(ctx.Params))
	}
}

func (r *LegacyAPIRoutes) HandleLogin(account *model.AccountDTO, expTime time.Duration) (string, error) {
	// Create session ID
	sessionID, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(err, "failed to create session ID")
	}

	// Save session ID to cache
	strSessionID := sessionID.String()
	r.legacyHandler.SessionCache.Set(strSessionID, account, expTime)

	return strSessionID, nil
}

func (r *LegacyAPIRoutes) HandleLogout(c *gin.Context) error {
	sessionID := r.legacyHandler.GetSessionID(c.Request)
	r.legacyHandler.SessionCache.Delete(sessionID)
	return nil
}

func (r *LegacyAPIRoutes) Setup(g *gin.Engine) {
	r.legacyHandler = webserver.GetLegacyHandler(webserver.Config{
		DB:       r.deps.Database,
		DataDir:  r.cfg.Storage.DataDir,
		RootPath: r.cfg.Http.RootPath,
		Log:      false, // Already done by gin
	}, r.deps)
	r.legacyHandler.PrepareSessionCache()

	legacyGroup := g.Group("/")

	// Use a custom recovery handler to expose the errors that the frontend catch to redirect to
	// the login page and display messages.
	// This will be improved in the new API.
	legacyGroup.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		c.Data(http.StatusInternalServerError, "text/plain", []byte(err.(error).Error()))
	}))

	// router.GET(jp("/api/tags"), withLogging(hdl.apiGetTags))
	legacyGroup.GET("/api/tags", r.handle(r.legacyHandler.ApiGetTags))
	// router.PUT(jp("/api/tag"), withLogging(hdl.apiRenameTag))
	legacyGroup.PUT("/api/tags", r.handle(r.legacyHandler.ApiRenameTag))
	// router.GET(jp("/api/bookmarks"), withLogging(hdl.apiGetBookmarks))
	legacyGroup.GET("/api/bookmarks", r.handle(r.legacyHandler.ApiGetBookmarks))
	// router.POST(jp("/api/bookmarks"), withLogging(hdl.apiInsertBookmark))
	legacyGroup.POST("/api/bookmarks", r.handle(r.legacyHandler.ApiInsertBookmark))
	// router.DELETE(jp("/api/bookmarks"), withLogging(hdl.apiDeleteBookmark))
	legacyGroup.DELETE("/api/bookmarks", r.handle(r.legacyHandler.ApiDeleteBookmark))
	// router.PUT(jp("/api/bookmarks"), withLogging(hdl.apiUpdateBookmark))
	legacyGroup.PUT("/api/bookmarks", r.handle(r.legacyHandler.ApiUpdateBookmark))
	// router.PUT(jp("/api/bookmarks/tags"), withLogging(hdl.apiUpdateBookmarkTags))
	legacyGroup.PUT("/api/bookmarks/tags", r.handle(r.legacyHandler.ApiUpdateBookmarkTags))
	// router.POST(jp("/api/bookmarks/ext"), withLogging(hdl.apiInsertViaExtension))
	legacyGroup.POST("/api/bookmarks/ext", r.handle(r.legacyHandler.ApiInsertViaExtension))
	// router.DELETE(jp("/api/bookmarks/ext"), withLogging(hdl.apiDeleteViaExtension))
	legacyGroup.DELETE("/api/bookmarks/ext", r.handle(r.legacyHandler.ApiDeleteViaExtension))
}

func NewLegacyAPIRoutes(logger *logrus.Logger, deps *dependencies.Dependencies, cfg *config.Config) *LegacyAPIRoutes {
	return &LegacyAPIRoutes{
		logger: logger,
		cfg:    cfg,
		deps:   deps,
	}
}
