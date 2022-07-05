package main

import (
	"encoding/json"
	"os"

	"github.com/kennethklee/gin-gorm-rest/generator"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB = createDB()

// Some example structs
type Owner struct {
	ID   uint   `json:"id" gorm:"primary_key"`
	Name string `json:"name"`

	Animals []Animal `json:"animals,omitempty" gorm:"foreignkey:OwnerID"`
}

type Animal struct {
	ID      uint   `json:"id" gorm:"primary_key"`
	OwnerID uint   `json:"owner_id,omitempty"`
	Owner   *Owner `json:"owner,omitempty" gorm:"foreignkey:OwnerID"`
	Name    string `json:"name"`
	Species string `json:"species"`
	Age     int    `json:"age,omitempty"`
}

// Parent child association used in owners_animals.go example
var OwnerAnimalAssoc = generator.Association{ParentName: "owner", Association: "Animals"}

func init() {
	// Populate database with some data
	if err := createFixture("../tests/fixtures/owners.json", &[]Owner{}); err != nil {
		panic(err)
	}
	if err := createFixture("../tests/fixtures/animals.json", &[]Animal{}); err != nil {
		panic(err)
	}
}

func createDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Owner{})
	db.AutoMigrate(&Animal{})

	return db
}

func createFixture(file string, models interface{}) error {
	// Open json file
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	// Decode json file
	if err = json.NewDecoder(f).Decode(models); err != nil {
		return err
	}

	if results := DB.CreateInBatches(models, 100); results.Error != nil {
		return results.Error
	}
	return nil
}
