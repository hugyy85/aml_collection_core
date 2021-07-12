package main

import (
	"aml_collection_core/models"
	"aml_collection_core/protocol"
	"aml_collection_core/transport"
	"aml_collection_core/utils"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	router := gin.Default()

	router.Use(makeMeAuth())
	//c.JSON(200, gin.H{"result": "get product", "token": token})
	router.POST("/create", createTask)
	router.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
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
	//Нужно обработать индивидуалов корпорейтов и методы для них доступные
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
			Ref interface{}   `bson:"$ref"`
			ID  interface{}   `bson:"$id"`
		}{Ref: "application", ID: ApplicationObj.Id}},
		{"application_id", ApplicationObj.Id},
	}
	res, insertErr := collection.InsertOne(ctx, insertData)
	if insertErr != nil {
		log.Fatal(insertErr)
	}
	defer client.Disconnect(ctx)
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

var ApplicationObj Application // make global because can use in more then one function

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
		db, ctx, client := models.GetDb()
		collection := db.Collection("apitoken")
		err := collection.FindOne(context.TODO(), bson.D{{"token", token}}).Decode(&result)
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
		//var (
		//	someMap      = map[string]interface{}{}
		//	someMapMutex = sync.RWMutex{}
		//)
		//go func() {
		//	someMapMutex.Lock()
		//	someMap["_id"] = result.Application.ID
		//	someMapMutex.Unlock()
		//}()
		//someMapMutex.RLock()
		err = collection.FindOne(context.TODO(), bson.D{{"_id", result.Application.ID}}).Decode(&ApplicationObj)
		//err = collection.FindOne(context.TODO(), bson.D{{"_id", result.Application.ID}}).Decode(&ApplicationObj)
		//someMapMutex.RUnlock()
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Application %s is not found", ApplicationObj.Id)})
			c.Abort()
			return
		}

		defer client.Disconnect(ctx)
		c.Next()
	}

}
