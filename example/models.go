package main

import (
	"github.com/kennethklee/gin-gorm-rest/generator"
	"gorm.io/driver/sqlite"
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

func createDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Owner{})
	db.AutoMigrate(&Animal{})

	// Populate database with some data
	if err := createFixture("./tests/fixtures/owners.json", &[]Owner{}); err != nil {
		panic(err)
	}
	if err := createFixture("./tests/fixtures/animals.json", &[]Animal{}); err != nil {
		panic(err)
	}

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

	if results := db.CreateInBatches(models, 100); results.Error != nil {
		return results.Error
	}
	return nil
}
