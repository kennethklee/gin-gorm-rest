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

type Generator struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Generator {
	return &Generator{db}
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

func (g *Generator) ListModels(models interface{}, resolvers func(*gin.Context, *gorm.DB)) gin.HandlerFunc {
	return func(c *gin.Context) {
		instList := reflect.New(reflect.TypeOf(models)).Interface() // clone models

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

func (g *Generator) ListAssociatedModels(assoc Association, models interface{}, resolvers func(*gin.Context, *gorm.DB)) gin.HandlerFunc {
	return func(c *gin.Context) {
		instList := reflect.New(reflect.TypeOf(models)).Interface() // clone models

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

func (g *Generator) RenderModel(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if model, exists := c.Get(name); !exists {
			c.AbortWithStatus(http.StatusNotFound)
		} else {
			c.JSON(c.Writer.Status(), model)
		}
	}
}

func (g *Generator) FetchModel(model interface{}, name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Param(name) == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		inst := reflect.New(reflect.TypeOf(model)).Interface() // clone model
		if err := g.DB.Take(inst, c.Param(name)).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
		} else if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		} else {
			c.Set(name, inst)
		}
	}
}

func (g *Generator) FetchAssociatedModel(assoc Association, model interface{}, name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Param(name) == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		inst := reflect.New(reflect.TypeOf(model)).Interface() // clone model

		// FIXME: gorm doesn't return error when record not found, so do a COUNT first
		if count := g.DB.Model(c.MustGet(assoc.ParentName)).Association(assoc.Association).Count(); count != 1 {
			c.AbortWithStatus(http.StatusNotFound)
		} else if err := g.DB.Model(c.MustGet(assoc.ParentName)).Association(assoc.Association).Find(inst, c.Param(name)); errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
		} else if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err})
		} else {
			c.Set(name, inst)
		}
	}
}

func (g *Generator) CreateModel(model interface{}, name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		inst := reflect.New(reflect.TypeOf(model)).Interface() // clone model
		if errs := g.bindAndValidate(c, inst); errs != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, ValidationErrorResponse{"validation errors", errs})
			return
		}

		if err := g.DB.Create(inst).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.Status(http.StatusCreated)
		c.Set(name, inst)
	}
}

func (g *Generator) CreateAssociatedModel(assoc Association, model interface{}, name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		inst := reflect.New(reflect.TypeOf(model)).Interface() // clone model
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
		c.Set(name, inst)
	}
}

func (g *Generator) UpdateModel(model interface{}, name string, mergeFunc func(src interface{}, dest interface{})) gin.HandlerFunc {
	return func(c *gin.Context) {
		inst := reflect.New(reflect.TypeOf(model)).Interface() // clone model
		dest := c.MustGet(name)

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

		c.Set(name, dest)
	}
}

func (g *Generator) DeleteModel(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		model := c.MustGet(name)
		if err := g.DB.Delete(model).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusNoContent, model)
	}
}

// not implemented
func (g *Generator) ReadableEndpoints(resource *gin.RouterGroup, model interface{}, name string, resolvers func(*gin.Context, *gorm.DB)) {
	panic("not implemented")
	// resource.GET("", g.ListModels(model, resolvers))
	// resource.GET("/:"+name, g.FetchModel(model, name), g.RenderModel(name))
}

// not implemented
func (g *Generator) WritableEndpoints(resource *gin.RouterGroup, model interface{}, name string, mergeFn func(src interface{}, dest interface{})) {
	panic("not implemented")
	// resource.POST("", g.CreateModel(model, name), g.RenderModel(name))
	// resource.PUT("/:"+name, g.FetchModel(model, name), g.UpdateModel(model, name, mergeFn), g.RenderModel(name))
	// resource.DELETE("/:"+name, g.FetchModel(model, name), g.DeleteModel(name))
}

// not implemented
func (g *Generator) ReadableAssociatedEndpoints(resource *gin.RouterGroup, assoc Association, model interface{}, name string, resolvers func(*gin.Context, *gorm.DB)) {
	panic("not implemented")
	// resource.GET("", g.ListAssociatedModels(assoc, model, resolvers))
	// resource.GET("/:"+name, g.FetchAssociatedModel(assoc, model, name), g.RenderModel(name))
}

// not implemented
func (g *Generator) WritableAssociatedEndpoints(resource *gin.RouterGroup, assoc Association, model interface{}, name string, mergeFn func(src interface{}, dest interface{})) {
	panic("not implemented")
	// resource.POST("", g.CreateAssociatedModel(assoc, model, name), g.RenderModel(name))
	// resource.PUT("/:"+name, g.UpdateModel(model, name, mergeFn), g.RenderModel(name))
	// resource.DELETE("/:"+name, g.DeleteModel(name))
}
