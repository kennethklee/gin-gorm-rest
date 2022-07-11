package generator

import "github.com/gin-gonic/gin"

type Handlers struct {
	Param  string
	List   gin.HandlerFunc
	Fetch  gin.HandlerFunc
	Render gin.HandlerFunc
	Create gin.HandlerFunc
	Update gin.HandlerFunc
	Delete gin.HandlerFunc
}

// Register boilderplate handler functions for CRUD operations.
func (h *Handlers) Register(app *gin.Engine, path string, middlewares ...gin.HandlerFunc) {
	group := app.Group(path, middlewares...)
	group.GET("", h.List)
	group.POST("", h.Create, h.Render)
	group.GET("/:"+h.Param, h.Fetch, h.Render)
	group.PUT("/:"+h.Param, h.Fetch, h.Update, h.Render)
	group.DELETE("/:"+h.Param, h.Delete)
}
