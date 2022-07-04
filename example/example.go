package example

import (
	"encoding/json"
	"os"

	"github.com/gin-gonic/gin"
)

var app = gin.Default()
var db = connectDB()

func Start(listenAddr string) {
	createData()

	app.Run(listenAddr)
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

func createData() {
	if err := createFixture("./generator/fixtures/owners.json", &[]Owner{}); err != nil {
		panic(err)
	}
	if err := createFixture("./generator/fixtures/animals.json", &[]Animal{}); err != nil {
		panic(err)
	}
}
