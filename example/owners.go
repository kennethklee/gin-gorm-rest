package example

import "github.com/kennethklee/gin-gorm-rest/generator"

var ownerResource = generator.New(db, Owner{}, "owner")
var ListOwners = ownerResource.List(nil)
var CreateOwner = ownerResource.Create()
var RenderOwner = ownerResource.Render()
var FetchOwner = ownerResource.Fetch()
var UpdateOwner = ownerResource.Update(mergeOwners)
var DeleteOwner = ownerResource.Delete()

func init() {
	owners := app.Group("/owners")
	owners.GET("", ListOwners)
	owners.POST("", CreateOwner, RenderOwner)
	owners.GET("/:owner", FetchOwner, RenderOwner)
	owners.PUT("/:owner", FetchOwner, UpdateOwner, RenderOwner)
	owners.DELETE("/:owner", DeleteOwner)
}

func mergeOwners(src, dest interface{}) {
	srcOwner := src.(*Owner)
	destOwner := dest.(*Owner)
	destOwner.Name = srcOwner.Name
}
