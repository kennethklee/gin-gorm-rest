package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/kennethklee/gin-gorm-rest/generator"
	"gorm.io/gorm"
)

var animalGenerator = generator.New(DB, Animal{}, "animal")

// Manually create handlers as an example. See owners.go for an example of using the Handlers helper
var ListAnimals = animalGenerator.List(animalResolvers)
var RenderAnimal = animalGenerator.Render()
var FetchAnimal = animalGenerator.Fetch()
var CreateAnimal = animalGenerator.Create()
var UpdateAnimal = animalGenerator.Update(mergeAnimals)
var DeleteAnimal = animalGenerator.Delete()

func init() {
	animals := app.Group("/animals")

	animals.GET("", ListAnimals)
	// animals.POST("", CreateAnimal, RenderAnimal)
	// animals.GET("/:animal", FetchAnimal, RenderAnimal)
	// animals.PUT("/:animal", FetchAnimal, UpdateAnimal, RenderAnimal)
	// animals.DELETE("/:animal", DeleteAnimal)
}

// When performing a PUT, we need to merge the input data with the existing data
func mergeAnimals(src, dest interface{}) {
	srcAnimal := src.(*Animal)
	destAnimal := dest.(*Animal)

	destAnimal.Name = srcAnimal.Name
	destAnimal.Species = srcAnimal.Species
	destAnimal.Age = srcAnimal.Age
}

// When listing, we need to resolve query params. We can also use this to do pagination or limit results or even set headers
func animalResolvers(ctx *gin.Context, queryset *gorm.DB) {
	// Let's add some sort of search
	if search := ctx.Query("search"); search != "" {
		startsWith := search + "%"
		queryset = queryset.Where("name LIKE ?", startsWith)
	}

	// Let's limit the results
	queryset = queryset.Select("id, name, species")
	queryset = queryset.Limit(20)

	// Even add some headers
	var count int64
	if err := queryset.Count(&count).Error; err != nil {
		ctx.AbortWithStatus(500)
	} else {
		ctx.Header("X-Total-Count", fmt.Sprintf("%d", count))
	}
}
