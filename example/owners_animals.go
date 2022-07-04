package example

import "github.com/kennethklee/gin-gorm-rest/generator"

var animalGenerator = generator.New(db, Animal{}, "animal")

var ListOwnerAnimals = animalGenerator.ListAssociated(OwnerAnimalAssoc, nil)
var RenderAnimal = animalGenerator.Render()
var FetchOwnerAnimal = animalGenerator.FetchAssociated(OwnerAnimalAssoc)
var CreateOwnerAnimal = animalGenerator.CreateAssociated(OwnerAnimalAssoc)
var UpdateAnimal = animalGenerator.Update(mergeAnimals)
var DeleteAnimal = animalGenerator.Delete()

func init() {
	ownerAnimals := app.Group("/owners/:owner/animals", FetchOwner)

	ownerAnimals.GET("", ListOwnerAnimals)
	ownerAnimals.POST("", CreateOwnerAnimal, RenderAnimal)
	ownerAnimals.GET("/:animal", FetchOwnerAnimal, RenderAnimal)
	ownerAnimals.PUT("/:animal", FetchOwnerAnimal, UpdateAnimal, RenderAnimal)
	ownerAnimals.DELETE("/:animal", DeleteAnimal)
}

func mergeAnimals(src, dest interface{}) {
	srcAnimal := src.(*Animal)
	destAnimal := dest.(*Animal)

	destAnimal.Name = srcAnimal.Name
	destAnimal.Species = srcAnimal.Species
	destAnimal.Age = srcAnimal.Age
}
