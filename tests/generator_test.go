package generator_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kennethklee/gin-gorm-rest/generator"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test structs
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

var ownerAnimalAssoc = generator.Association{"owner", "Animals"}

var origDB *gorm.DB
var ownerGenerator *generator.Generator
var animalGenerator *generator.Generator

func init() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info), // For debugging
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(Animal{})

	origDB = db
	ownerGenerator = generator.New(db, Owner{}, "owner")
	animalGenerator = generator.New(db, Animal{}, "animal")
	gin.SetMode(gin.ReleaseMode) // omit gin-debug logs

	createTestFixtures()
}

func mockContext(req *http.Request) (*gin.Context, *httptest.ResponseRecorder) {
	resp := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(resp)
	context.Request = req
	return context, resp
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

	if results := animalGenerator.DB.CreateInBatches(models, 100); results.Error != nil {
		return results.Error
	}
	return nil
}

func testSetup() {
	animalGenerator.DB = animalGenerator.DB.Begin() // start a transaction
}

func createTestFixtures() {
	if err := createFixture("fixtures/owners.json", &[]Owner{}); err != nil {
		panic(err)
	}
	if err := createFixture("fixtures/animals.json", &[]Animal{}); err != nil {
		panic(err)
	}
}

func testTearDown() {
	animalGenerator.DB.Rollback()
	animalGenerator.DB = origDB // Restore to original DB
}

func TestRenderModel(t *testing.T) {
	testSetup()
	defer testTearDown()

	// mock request
	req, _ := http.NewRequest("GET", "", nil)
	context, resp := mockContext(req)
	context.Set("animal", map[string]string{"test": "test"})

	// test
	animalGenerator.Render()(context)

	// check response
	body, _ := io.ReadAll(resp.Body)
	if resp.Code != http.StatusOK {
		t.Errorf("failed call with %d code: %s", resp.Code, string(body))
		return
	}

	results := map[string]string{}
	err := json.Unmarshal(body, &results)
	if err != nil {
		t.Errorf("failed JSON response decode: %v\nfull body: %s", err, string(body))
		return
	}

	if results["test"] != "test" {
		t.Errorf("failed response: %s", string(body))
		return
	}
}

func TestListModels(t *testing.T) {
	testSetup()
	defer testTearDown()

	// mock request
	req, _ := http.NewRequest("GET", "", nil)
	context, resp := mockContext(req)

	// test
	animalGenerator.List(func(ctx *gin.Context, qs *gorm.DB) bool {
		qs = qs.Limit(2).Order("id asc")
		return true
	})(context)

	// check response
	body, _ := io.ReadAll(resp.Body)
	if resp.Code != http.StatusOK {
		t.Errorf("failed call with %d code: %s", resp.Code, string(body))
		return
	}

	results := []Animal{}
	err := json.Unmarshal(body, &results)
	if err != nil {
		t.Errorf("failed JSON response decode: %v\nfull body: %s", err, string(body))
		return
	}

	if len(results) != 2 {
		t.Errorf("failed count: %s", string(body))
		return
	}

	if results[0].Name != "Alfred" {
		t.Errorf("failed response: %s", string(body))
		return
	}

	if results[1].Name != "Bella" {
		t.Errorf("failed response: %s", string(body))
		return
	}
}

func TestListAssociatedModels(t *testing.T) {
	testSetup()
	defer testTearDown()

	// mock request
	req, _ := http.NewRequest("GET", "", nil)
	context, resp := mockContext(req)
	context.Set("owner", Owner{ID: 1})

	// test
	animalGenerator.ListAssociated(ownerAnimalAssoc, func(ctx *gin.Context, qs *gorm.DB) bool {
		qs = qs.Limit(2).Order("id asc")
		return true
	})(context)

	// check response
	body, _ := io.ReadAll(resp.Body)
	if resp.Code != http.StatusOK {
		t.Errorf("failed call with %d code: %s", resp.Code, string(body))
		return
	}

	results := []Animal{}
	err := json.Unmarshal(body, &results)
	if err != nil {
		t.Errorf("failed JSON response decode: %v\nfull body: %s", err, string(body))
		return
	}

	if len(results) != 2 {
		t.Errorf("failed count: %s", string(body))
		return
	}

	if results[0].Name != "Alfred" {
		t.Errorf("failed response: %s", string(body))
		return
	}

	if results[1].Name != "Bella" {
		t.Errorf("failed response: %s", string(body))
		return
	}
}

func TestFetchModel(t *testing.T) {
	testSetup()
	defer testTearDown()

	// mock request
	req, _ := http.NewRequest("GET", "", nil)
	context, resp := mockContext(req)
	context.Params = gin.Params{gin.Param{Key: "animal", Value: "1"}}

	// test
	animalGenerator.Fetch()(context)

	// check response
	if resp.Code != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("failed call with %d code: %s", resp.Code, string(body))
		return
	}

	animal := context.MustGet("animal").(*Animal)

	if animal.Name != "Alfred" {
		t.Errorf("incorrect name: %s", animal.Name)
		return
	}
}

func TestFetchModelNotFound(t *testing.T) {
	testSetup()
	defer testTearDown()

	// mock request
	req, _ := http.NewRequest("GET", "", nil)
	context, resp := mockContext(req)
	context.Params = gin.Params{gin.Param{Key: "animal", Value: "0"}}

	// test
	animalGenerator.Fetch()(context)

	// check response
	if resp.Code != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("failed call with %d code: %s", resp.Code, string(body))
		return
	}

	if _, exists := context.Get("animal"); exists {
		t.Errorf("should not have found animal in context")
		return
	}
}

func TestFetchAssociatedModel(t *testing.T) {
	testSetup()
	defer testTearDown()

	// mock request
	req, _ := http.NewRequest("GET", "", nil)
	context, resp := mockContext(req)
	context.Set("owner", Owner{ID: 1})
	context.Params = gin.Params{gin.Param{Key: "animal", Value: "1"}}

	// test
	animalGenerator.FetchAssociated(ownerAnimalAssoc)(context) // Chained association

	// check response
	if resp.Code != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("failed call with %d code: %s", resp.Code, string(body))
		return
	}

	animal := context.MustGet("animal").(*Animal)

	if animal.Name != "Alfred" {
		t.Errorf("incorrect name: %s", animal.Name)
		return
	}
}

func TestFetchAssociatedModelNotFound(t *testing.T) {
	testSetup()
	defer testTearDown()

	// mock request
	req, _ := http.NewRequest("GET", "", nil)
	context, resp := mockContext(req)
	context.Set("owner", Owner{ID: 1})
	context.Params = gin.Params{gin.Param{Key: "animal", Value: "4"}}

	// test
	animalGenerator.FetchAssociated(ownerAnimalAssoc)(context) // Chained association

	// check response
	if resp.Code != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("failed call with %d code: %s", resp.Code, string(body))
		return
	}

	if _, exists := context.Get("animal"); exists {
		t.Errorf("should not have found animal in context")
		return
	}
}

func TestCreateModel(t *testing.T) {
	testSetup()
	defer testTearDown()

	req, _ := http.NewRequest("POST", "/api/animals", strings.NewReader(`{"name": "test", "species": "test", "age": 1}`))
	context, resp := mockContext(req)

	animalGenerator.Create()(context)

	if context.Writer.Status() != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("failed call with %d code: %s", context.Writer.Status(), string(body))
		return
	}

	animal := context.MustGet("animal").(*Animal)
	if animal.Name != "test" {
		t.Errorf("incorrect name: %s", animal.Name)
		return
	}
	if animal.Species != "test" {
		t.Errorf("incorrect species: %s", animal.Species)
		return
	}
	if animal.Age != 1 {
		t.Errorf("incorrect age: %d", animal.Age)
		return
	}

	// check database for change
	finalAnimal := Animal{}
	animalGenerator.DB.Take(&finalAnimal, animal.ID)

	if animal.ID != finalAnimal.ID {
		t.Errorf("failed to create vendor in database")
		return
	}

	if animal.Name != finalAnimal.Name {
		t.Errorf("input vendor name should match created vendor in db: %s != %s", animal.Name, finalAnimal.Name)
		return
	}
}

func TestUpdateModel(t *testing.T) {
	testSetup()
	defer testTearDown()
	targetAnimal := Animal{}
	animalGenerator.DB.Take(&targetAnimal, 1)

	req, _ := http.NewRequest("PUT", "/api/animals/1", strings.NewReader(`{"name": "changed"}`))
	context, resp := mockContext(req)
	context.Set("animal", &Animal{ID: 1})

	animalGenerator.Update(func(src, dest interface{}) error {
		srcVendor := src.(*Animal)
		destVendor := dest.(*Animal)
		destVendor.Name = srcVendor.Name
		return nil
	})(context)

	if context.Writer.Status() != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("failed call with %d code: %s", context.Writer.Status(), string(body))
		return
	}

	vendor := context.MustGet("animal").(*Animal)
	if vendor.Name != "changed" {
		t.Errorf("incorrect response name: %s", vendor.Name)
		return
	}

	// check database for change
	finalAnimal := Animal{}
	animalGenerator.DB.Take(&finalAnimal, 1)

	if targetAnimal.ID != finalAnimal.ID {
		t.Errorf("ID do not match: %d != %d", targetAnimal.ID, finalAnimal.ID)
		return
	}

	if targetAnimal.Name == finalAnimal.Name {
		t.Errorf("names shouldn't match: %s != %s", targetAnimal.Name, finalAnimal.Name)
		return
	}

	if finalAnimal.Name != "changed" {
		t.Errorf("incorrect db record name: %s", finalAnimal.Name)
		return
	}
}

func TestDeleteModel(t *testing.T) {
	testSetup()
	defer testTearDown()
	req, _ := http.NewRequest("DELETE", "/animals/1", nil)
	context, resp := mockContext(req)
	context.Set("animal", &Animal{ID: 1})

	animalGenerator.Delete()(context)

	if context.Writer.Status() != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("failed call with %d code: %s", context.Writer.Status(), string(body))
		return
	}

	// check database for change
	finalVendor := Animal{}
	result := animalGenerator.DB.Take(&finalVendor, 1)
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		t.Errorf("failed to delete vendor from database: %v", result.Error)
		return
	}
}
