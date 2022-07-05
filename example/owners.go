package main

import "github.com/kennethklee/gin-gorm-rest/generator"

var ownerHandlers = generator.New(DB, Owner{}, "owner").Handlers(nil, mergeOwners)

func init() {
	owners := app.Group("/owners")
	owners.GET("", ownerHandlers.List)
	owners.POST("", ownerHandlers.Create, ownerHandlers.Render)
	owners.GET("/:owner", ownerHandlers.Fetch, ownerHandlers.Render)
	owners.PUT("/:owner", ownerHandlers.Fetch, ownerHandlers.Update, ownerHandlers.Render)
	owners.DELETE("/:owner", ownerHandlers.Delete)

	// Short form of the above would be:
	// ownerHandlers.Register(app, "/owners")

	// NOTE: you can use the generator directly like in animals.go
}

func mergeOwners(src, dest interface{}) {
	srcOwner := src.(*Owner)
	destOwner := dest.(*Owner)
	destOwner.Name = srcOwner.Name
}
