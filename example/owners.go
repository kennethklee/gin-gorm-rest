package example

var ListOwners = g.ListModels(Owner{}, nil)
var CreateOwner = g.CreateModel(Owner{}, "owner")
var RenderOwner = g.RenderModel("owner")
var FetchOwner = g.FetchModel(Owner{}, "owner")
var UpdateOwner = g.UpdateModel(Owner{}, "owner", mergeOwners)
var DeleteOwner = g.DeleteModel("owner")

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
