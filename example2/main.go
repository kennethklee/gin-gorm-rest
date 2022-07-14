/**
 * Smallest example of how to use the library.
 */
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/kennethklee/gin-gorm-rest/generator"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// a model
type User struct {
	gorm.Model
	Name string `json:"name"`
}

// initialize gin, gorm, and the generator
var app = gin.Default()
var db = createDB()
var userHandlers = generator.New(db, User{}, "user").Handlers(nil, mergeUsers)

// in-memory sqlite db
func createDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&User{})

	db.Create(&User{Name: "kenneth"})

	return db
}

// handles record updates
func mergeUsers(src interface{}, dst interface{}) error {
	srcUser := src.(*User)
	dstUser := dst.(*User)
	dstUser.Name = srcUser.Name

	return nil
}

func main() {
	// create the routes with the generated handlers
	userHandlers.Register(app, "/users")

	// start the server
	app.Run(":3000")
}
