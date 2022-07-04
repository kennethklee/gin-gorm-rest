package main

import "github.com/kennethklee/gin-gorm-rest/generator"

var ownerHandlers = generator.New(db, Owner{}, "owner").Handlers(nil, mergeOwners)

func init() {
	owners := app.Group("/owners")
	owners.GET("", ownerHandlers.List)
	owners.POST("", ownerHandlers.Create, ownerHandlers.Render)
	owners.GET("/:owner", ownerHandlers.Fetch, ownerHandlers.Render)
	owners.PUT("/:owner", ownerHandlers.Fetch, ownerHandlers.Update, ownerHandlers.Render)
	owners.DELETE("/:owner", ownerHandlers.Delete)

	// Short form of the above would be:
	// ownerHandlers.Register(app, "/owners")

	// Alternatively you can use the following:
	// var owner = generator.New(db, Owner{}, "owner")  // Typically goes top of file
	// owners.GET("", owner.List())
	// owners.POST("", owner.Create(), owner.Render())
	// owners.GET("/:owner", owner.Fetch(), owner.Render())
	// owners.PUT("/:owner", owner.Fetch(), owner.Update(), owner.Render())
	// owners.DELETE("/:owner", owner.Delete())
}

func mergeOwners(src, dest interface{}) {
	srcOwner := src.(*Owner)
	destOwner := dest.(*Owner)
	destOwner.Name = srcOwner.Name
}
