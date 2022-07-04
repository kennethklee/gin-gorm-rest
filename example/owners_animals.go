package example

var ListAnimals = g.ListAssociatedModels(OwnerAnimalAssoc, []Animal{}, nil)
var RenderAnimal = g.RenderModel("animal")
var FetchAnimal = g.FetchAssociatedModel(OwnerAnimalAssoc, Animal{}, "animal")
var CreateAnimal = g.CreateAssociatedModel(OwnerAnimalAssoc, Animal{}, "animal")
var UpdateAnimal = g.UpdateModel(Animal{}, "animal", mergeAnimals)
var DeleteAnimal = g.DeleteModel("animal")

func init() {
	ownerAnimals := app.Group("/owners/:owner/animals", g.FetchModel(Owner{}, "owner"))

	ownerAnimals.GET("", ListAnimals)
	ownerAnimals.POST("", CreateAnimal, RenderAnimal)
	ownerAnimals.GET("/:animal", FetchAnimal, RenderAnimal)
	ownerAnimals.PUT("/:animal", FetchAnimal, UpdateAnimal, RenderAnimal)
	ownerAnimals.DELETE("/:animal", DeleteAnimal)
}

func mergeAnimals(src, dest interface{}) {
	srcAnimal := src.(*Animal)
	destAnimal := dest.(*Animal)

	destAnimal.Name = srcAnimal.Name
	destAnimal.Species = srcAnimal.Species
	destAnimal.Age = srcAnimal.Age
}
