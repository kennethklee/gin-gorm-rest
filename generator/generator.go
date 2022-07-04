package generator

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

/**
 * These are extras that integrate into database
 */

type Association struct {
	ParentName  string // name of the parent context variable
	Association string // name of the association in model
}

type ValidationErrorResponse struct {
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors"`
}

type ResolverFn func(*gin.Context, *gorm.DB)
type MergerFn func(src interface{}, dest interface{})

type Generator struct {
	DB     *gorm.DB
	model  reflect.Type
	models reflect.Type
	Param  string
}

func New(db *gorm.DB, model interface{}, paramName string) *Generator {
	mt := reflect.TypeOf(model)
	return &Generator{DB: db, model: mt, models: reflect.SliceOf(mt), Param: paramName}
}

func (g *Generator) bindAndValidate(c *gin.Context, model interface{}) map[string]string {
	if err := c.ShouldBindJSON(model); err != nil {
		// check if err is a validator error
		if ve, ok := err.(validator.ValidationErrors); ok {
			errors := make(map[string]string)
			for _, fieldErr := range ve {
				errors[fieldErr.Field()] = fieldErr.Tag()
			}
			return errors
		} else if te, ok := err.(*json.UnmarshalTypeError); ok {
			return map[string]string{te.Field: "invalid " + te.Type.String() + " type"}
		} else {
			return map[string]string{"error": err.Error()}
		}
	}
	return nil
}

// Create an instance of model
func (g *Generator) new() interface{} {
	return reflect.New(g.model).Interface()
}

// Creates a slice of models
func (g *Generator) newSlice() interface{} {
	return reflect.New(g.models).Interface()
}

// Creates a listing handler. Resolvers is a function that can be used to fine-tune the queryset or add pagination.
func (g *Generator) List(resolvers ResolverFn) gin.HandlerFunc {
	return func(c *gin.Context) {
		instList := g.newSlice()

		// Resolvers
		queryset := g.DB.Model(instList)
		if resolvers != nil {
			resolvers(c, queryset)
		}

		// Perform
		if err := queryset.Find(instList).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, instList)
	}
}

// Creates an associated listing handler. This is used for child relationships of a parent association. Resolvers is a function that can be used to fine-tune the queryset or add pagination.
func (g *Generator) ListAssociated(assoc Association, resolvers ResolverFn) gin.HandlerFunc {
	return func(c *gin.Context) {
		instList := g.newSlice()

		// Resolvers
		queryset := g.DB.Model(c.MustGet(assoc.ParentName))
		if resolvers != nil {
			resolvers(c, queryset)
		}

		// Perform
		if err := queryset.Association(assoc.Association).Find(instList); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err})
			return
		}
		c.JSON(http.StatusOK, instList)
	}
}

// Creates a rendering handler. This is used for rendering a model to JSON
func (g *Generator) Render() gin.HandlerFunc {
	return func(c *gin.Context) {
		if model, exists := c.Get(g.Param); !exists {
			c.AbortWithStatus(http.StatusNotFound)
		} else {
			c.JSON(c.Writer.Status(), model)
		}
	}
}

// Creates a handler to retrieve a single model and store it into the context.
func (g *Generator) Fetch() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Param(g.Param) == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		inst := g.new()
		if err := g.DB.Take(inst, c.Param(g.Param)).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
		} else if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		} else {
			c.Set(g.Param, inst)
		}
	}
}

// Creates an associated handler that retrieves a child model from a parent relationship, then stores it into the context.
func (g *Generator) FetchAssociated(assoc Association) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Param(g.Param) == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		inst := g.new()

		// FIXME: gorm doesn't return error when record not found, so do a COUNT first
		if count := g.DB.Model(c.MustGet(assoc.ParentName)).Where(c.Param(g.Param)).Association(assoc.Association).Count(); count != 1 {
			c.AbortWithStatus(http.StatusNotFound)
		} else if err := g.DB.Model(c.MustGet(assoc.ParentName)).Association(assoc.Association).Find(inst, c.Param(g.Param)); errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
		} else if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err})
		} else {
			c.Set(g.Param, inst)
		}
	}
}

// Creates a handler to create a model and store it into the context.
func (g *Generator) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		inst := g.new()
		if errs := g.bindAndValidate(c, inst); errs != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, ValidationErrorResponse{"validation errors", errs})
			return
		}

		if err := g.DB.Create(inst).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.Status(http.StatusCreated)
		c.Set(g.Param, inst)
	}
}

// Creates an associated handler to create a child model from a parent relationship.
func (g *Generator) CreateAssociated(assoc Association) gin.HandlerFunc {
	return func(c *gin.Context) {
		inst := g.new()
		if errs := g.bindAndValidate(c, &inst); errs != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, ValidationErrorResponse{"validation errors", errs})
			return
		}

		if err := g.DB.Model(inst).Create(inst).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		if err := g.DB.Model(c.MustGet(assoc.ParentName)).Association(assoc.Association).Append(inst); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err})
			return
		}

		c.Status(http.StatusCreated)
		c.Set(g.Param, inst)
	}
}

// Creates a handler that updates a single record and stores it into the context.
func (g *Generator) Update(mergeFunc MergerFn) gin.HandlerFunc {
	return func(c *gin.Context) {
		inst := g.new()
		dest := c.MustGet(g.Param)

		if errs := g.bindAndValidate(c, inst); errs != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, ValidationErrorResponse{"validation errors", errs})
			return
		}

		// Merge
		mergeFunc(inst, dest)

		if err := g.DB.Save(dest).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.Set(g.Param, dest)
	}
}

// Creates a deletion handler that deletes a model and responds with 204 No Content.
func (g *Generator) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		model := c.MustGet(g.Param)
		if err := g.DB.Delete(model).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusNoContent, model)
	}
}

// WIP: not fully implemented
func (g *Generator) ReadableEndpoints(resource *gin.RouterGroup, resolvers ResolverFn) {
	resource.GET("", g.List(resolvers))
	resource.GET("/:"+g.Param, g.Fetch(), g.Render())
}

// WIP: not fully implemented
func (g *Generator) WritableEndpoints(resource *gin.RouterGroup, mergeFn MergerFn) {
	resource.POST("", g.Create(), g.Render())
	resource.PUT("/:"+g.Param, g.Fetch(), g.Update(mergeFn), g.Render())
	resource.DELETE("/:"+g.Param, g.Fetch(), g.Delete())
}

// WIP: not fullimplemented
func (g *Generator) ReadableAssociatedEndpoints(resource *gin.RouterGroup, assoc Association, resolvers ResolverFn) {
	resource.GET("", g.ListAssociated(assoc, resolvers))
	resource.GET("/:"+g.Param, g.FetchAssociated(assoc), g.Render())
}

// WIP: not implemented
func (g *Generator) WritableAssociatedEndpoints(resource *gin.RouterGroup, assoc Association, mergeFn MergerFn) {
	resource.POST("", g.CreateAssociated(assoc), g.Render())
	resource.PUT("/:"+g.Param, g.Update(mergeFn), g.Render())
	resource.DELETE("/:"+g.Param, g.Delete())
}

// WIP: Creates all routes for `/<plural>`
func (g *Generator) CreateRoutes(app *gin.Engine, plural string, resolvers ResolverFn, mergeFn MergerFn) {
	endpoint := app.Group("/" + plural)
	g.ReadableEndpoints(endpoint, resolvers)
	g.WritableEndpoints(endpoint, mergeFn)
}

// WIP: Creates all routes for `/<parentPlural>/:<parentGen.Param>/<plural>`
func (g *Generator) CreateAssocatedRoutes(app *gin.Engine, parentPlural string, parentGen *Generator, assoc Association, plural string, resolvers ResolverFn, mergeFn MergerFn) {
	endpoint := app.Group("/"+parentPlural+"/:"+parentGen.Param+"/"+plural, parentGen.Fetch())
	g.ReadableAssociatedEndpoints(endpoint, assoc, resolvers)
	g.WritableAssociatedEndpoints(endpoint, assoc, mergeFn)
}
