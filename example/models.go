package main

import (
	"github.com/kennethklee/gin-gorm-rest/generator"
	"gorm.io/gorm"
)

// Some example structs
type Owner struct {
	ID   uint   `json:"id" gorm:"primary_key"`
	Name string `json:"name"`

	Animals []Animal `json:"animals,omitempty" gorm:"foreignkey:OwnerID"`
}

type Animal struct {
	ID      uint   `json:"id" gorm:"primary_key"`
	OwnerID uint   `json:"owner_id"`
	Owner   *Owner `json:"owner,omitempty" gorm:"foreignkey:OwnerID"`
	Name    string `json:"name"`
	Species string `json:"species"`
	Age     int    `json:"age"`
}

var OwnerAnimalAssoc = generator.Association{ParentName: "owner", Association: "Animals"}

func connectDB() *gorm.DB {
	// db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	// if err != nil {
	// 	panic("failed to connect database")
	// }

	db.AutoMigrate(&Owner{})
	db.AutoMigrate(&Animal{})

	return db
}
