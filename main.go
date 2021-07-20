package main

import (
	"aml_collection_core/models"
	"aml_collection_core/protocol"
	"aml_collection_core/transport"
	"aml_collection_core/utils"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	router := gin.Default()
	router.Use(makeMeAuth())
	router.POST("/create", createTask)
	err := router.Run()
	if err != nil {
		log.Fatal(err)
	} // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func createTask(c *gin.Context) {
	var task protocol.CreateTaskRequest
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//Создать заявку в монго, выдать идентификатор и отправить в очередь на исполнение
	for source, checkMethods := range task.CheckMethods {
		allowedMethods, ok := ApplicationObj.AllowedCheckMethods[source]
		// проверка источника
		if ok == false {
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Methods are not allowed. '%s' source does not support for your token", source)})
			return
		} else if allowedMethods != nil { // проверка метода
			for _, method := range checkMethods {
				if !utils.Contains(allowedMethods, method) {
					c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Methods are not allowed. '%s' method does not support for your token", method)})
					return
				}
			}
		}
	}
	isValid, err := validEntityData(task)
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	db, ctx, client := models.GetDb()
	collection := db.Collection("checks")
	insertData := bson.D{
		{"data", bson.D{
			{"entity_type", task.EntityType},
			{"payload", task.Payload},
			{"application_id", task.Application}},
		},
		{"meta", bson.D{
			{"v", "2.1"},
			{"date_update", time.Now()},
			{"ttl", utils.GetDefaultString(task.Ttl, "60")},
			{"priority", utils.GetDefaultString(task.Priority, "3")},
			{"status", "OK"},
			{"checks_start_time", time.Now()},
		}},
		{"methods", task.CheckMethods},
		{"application", struct {
			Ref interface{} `bson:"$ref"`
			ID  interface{} `bson:"$id"`
		}{Ref: "application", ID: ApplicationObj.Id}},
		{"application_id", ApplicationObj.Id},
	}
	res, insertErr := collection.InsertOne(ctx, insertData)
	if insertErr != nil {
		log.Fatal(insertErr)
	}
	defer func(client *mongo.Client, ctx context.Context) {
		err := client.Disconnect(ctx)
		if err != nil {
			log.Println(err)
		}
	}(client, ctx)
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		objId := oid.Hex()
		transport.SendObjectIdToQueue(objId, ApplicationObj.Description)
		c.JSON(http.StatusCreated, gin.H{"success": true, "payload": protocol.ObjectId{Id: objId}})
	}
}

type Token struct {
	Application struct {
		Ref interface{} `bson:"$ref"`
		ID  interface{} `bson:"$id"`
	}
	Token    string
	IsActive bool `bson:"is_active"`
}

type Application struct {
	Id                  string              `bson:"id"`
	Description         string              `bson:"description"`
	AllowedCheckMethods map[string][]string `bson:"allowed_check_methods"`
}

var (
	ApplicationObj   = Application{} // make global because can use in more then one function after makeMeAuth()
	ApplicationMutex = sync.RWMutex{}
)

func makeMeAuth() gin.HandlerFunc {
	// create a value into which the result can be decoded
	return func(c *gin.Context) {
		AuthSchema := c.Request.Header["Authorization"]
		if AuthSchema == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "unsupported auth schema - use 'Authorization: Bearer TOKEN'"})
			c.Abort()
			return
		}
		AuthSchema = strings.Split(AuthSchema[0], " ")
		if AuthSchema[0] != "Bearer" {
			c.JSON(http.StatusForbidden, gin.H{"error": "unsupported auth schema - use 'Authorization: Bearer TOKEN'"})
			c.Abort()
			return
		}
		token := AuthSchema[1]
		var result Token
		db, ctx, _ := models.GetDb()
		collection := db.Collection("apitoken")
		err := collection.FindOne(ctx, bson.D{{"token", token}}).Decode(&result)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Token %s is not found", token)})
			c.Abort()
			return
		}

		if result.IsActive == false {
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Token %s is not active", token)})
			c.Abort()
			return
		}
		if oid, ok := result.Application.ID.(primitive.ObjectID); ok {
			ApplicationObj.Id = oid.Hex()
		}
		collection = db.Collection("application")
		ApplicationMutex.Lock()
		err = collection.FindOne(ctx, bson.M{"_id": result.Application.ID}).Decode(&ApplicationObj)
		ApplicationMutex.Unlock()
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Application %s is not found", ApplicationObj.Id)})
			c.Abort()
			return
		}

		//defer client.Disconnect(ctx)
		c.Next()
	}

}

// validEntityData - Проверяет входные данные, на предмет корректности checkMethods, entityType, requireKeys, args
func validEntityData(task protocol.CreateTaskRequest) (bool, string) {
	entityTypes := []string{"individual", "corporate", "entrepreneur"}
	if !utils.Contains(entityTypes, task.EntityType) {
		return false, fmt.Sprintf("'%s' entity type does not exist. Use %s", task.EntityType, entityTypes)
	}
	var sourceMethods map[string]map[string]map[string]protocol.Method
	jsonFile, err := os.Open("sourceMethods.json")
	if err != nil {
		log.Println(err)
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &sourceMethods)
	if err != nil {
		log.Println(err)
	}
	defer jsonFile.Close()
	// Необходимо проверить источник и метод а также пейлоад, тип лица и аргументы
	for source, methods := range task.CheckMethods {
		for _, method := range methods {
			existMethod, ok := sourceMethods[source]["methods"][method]
			// check method exists
			if !ok {
				return false, fmt.Sprintf("Method '%s' in source '%s' does not exists", method, source)
			}
			// check entity type for exists method
			if !utils.Contains(existMethod.EntityTypes, task.EntityType) {
				return false, fmt.Sprintf("Not correct entity type '%s' for '%s - %s' method. Use %s",
					task.EntityType, source, method, existMethod.EntityTypes)
			}

			// check require keys for exists method
			for _, requireKey := range existMethod.RequiredKeys {
				_, ok := task.Payload.GetField(requireKey)
				if !ok {
					return false, fmt.Sprintf("Missing require keys. '%s' source '%s' method - has require keys %s",
						source, method, existMethod.RequiredKeys)
				}
			}
			// check args for exists method
			// ****************

			///
		}
	}
	return true, ""
}

//curl -v -X POST http://localhost:8080/create -H 'content-type: application/json' \
//-d '{"application_id": "test","check_methods": {"spark": ["status"]},"payload": {"inn": "1234578901", "passport_number": "123456"}, "entity_type": "individual"}' \
//-H 'Authorization: Bearer 2xFcJM599JxF2TDsOFWK3GKWbXHm5yL3FvG4b1tnxGFzyxq3yxfyhNZh'
