package main

// Manually create associated handlers
var ListOwnerAnimals = animalGenerator.ListAssociated(OwnerAnimalAssoc, nil)
var FetchOwnerAnimal = animalGenerator.FetchAssociated(OwnerAnimalAssoc)
var CreateOwnerAnimal = animalGenerator.CreateAssociated(OwnerAnimalAssoc)

func init() {
	ownerAnimals := app.Group("/owners/:owner/animals", ownerHandlers.Fetch)

	ownerAnimals.GET("", ListOwnerAnimals)
	ownerAnimals.POST("", CreateOwnerAnimal, RenderAnimal)
	ownerAnimals.GET("/:animal", FetchOwnerAnimal, RenderAnimal)
	ownerAnimals.PUT("/:animal", FetchOwnerAnimal, UpdateAnimal, RenderAnimal)
	ownerAnimals.DELETE("/:animal", DeleteAnimal)
}
