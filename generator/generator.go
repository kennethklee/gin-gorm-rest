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

// ResolverFn resolves the queryset and returns whether to continue or not. Use the context to respond with errors if needed.
type ResolverFn func(*gin.Context, *gorm.DB) (ok bool)

// MergerFn provides a way to merge form input with the data model. It can also be used as extra validation besides the binding validator.
type MergerFn func(src interface{}, dest interface{}) error

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
			if ok := resolvers(c, queryset); !ok {
				return
			}
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
		if err := mergeFunc(inst, dest); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

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

// Handy function to create boilerplate handlers for CRUD operations.
func (g *Generator) Handlers(resolvers ResolverFn, mergeFn MergerFn) *Handlers {
	return &Handlers{
		Param:  g.Param,
		List:   g.List(resolvers),
		Fetch:  g.Fetch(),
		Create: g.Create(),
		Update: g.Update(mergeFn),
		Delete: g.Delete(),
	}
}

// Handy function to create boilderplate handlers for CRUD operations with associations.
func (g *Generator) AssociatedHandlers(assoc Association, resolvers ResolverFn, mergerFn MergerFn) *Handlers {
	return &Handlers{
		Param:  g.Param,
		List:   g.ListAssociated(assoc, resolvers),
		Fetch:  g.FetchAssociated(assoc),
		Create: g.CreateAssociated(assoc),
		Update: g.Update(mergerFn),
		Delete: g.Delete(),
	}
}
