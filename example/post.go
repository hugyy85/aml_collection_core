package main

import (
	"aml_collection_core/protocol"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	//"github.com/streadway/amqp"
	"log"
	"net/http"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	var resultTime []int64
	// make gorutins-threads
	countThreads := 5
	countChecks := 10
	for thread:=0;thread < countThreads;thread++{
		go func() {
			for i:=0;i < countChecks / countThreads;i++{
				start := time.Now().UnixNano()

				inputData := protocol.CreateTaskRequest{
					Application:  "testmazafaka",
					EntityType:   "corporate",
					CheckMethods: map[string][]string{"spark": {"status"}},
					Payload:      protocol.Payload{"1234657897"},
				}

				jsonValue, _ := json.Marshal(inputData)

				//_, err := doPOSTRequest("http://127.0.0.1:5003/v2/check/create",
				_, err := doPOSTRequest("http://127.0.0.1:8080/create",
					"2xFcJM599JxF2TDsOFWK3GKWbXHm5yL3FvG4b1tnxGFzyxq3yxfyhNZh", bytes.NewBuffer(jsonValue))
				if err != nil{
					log.Fatal(err)
				}
				//log.Println(string(resp))
				t := time.Now().UnixNano() - start
				resultTime = append(resultTime, t)
			}
			var result int64
			for _, index := range resultTime{
				result += index
			}
			res := result/int64(len(resultTime))
			fmt.Println(time.Unix(0, res))
			fmt.Println(strconv.FormatInt(result/int64(len(resultTime)), 10))
		}()

	}
	<-time.After(time.Second * 5)
	//fmt.Println(resultTime)
	//00:00.005672445 - go 00:00.010611637 - python sync 100 checks
}

func rabbit() {
	//	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	//	failOnError(err, "Failed to connect to RabbitMQ")
	//	defer conn.Close()
	//
	//	ch, err := conn.Channel()
	//	failOnError(err, "Failed to open a channel")
	//	defer ch.Close()
	//
	//	q, err := ch.QueueDeclare(
	//		"hello", // name
	//		false,   // durable
	//		false,   // delete when unused
	//		false,   // exclusive
	//		false,   // no-wait
	//		nil,     // arguments
	//	)
	//	failOnError(err, "Failed to declare a queue")
	//
	//	msgs, err := ch.Consume(
	//		q.Name, // queue
	//		"",     // consumer
	//		true,   // auto-ack
	//		false,  // exclusive
	//		false,  // no-local
	//		false,  // no-wait
	//		nil,    // args
	//	)
	//	failOnError(err, "Failed to register a consumer")
	//
	//	forever := make(chan bool)
	//
	//	go func() {
	//		for d := range msgs {
	//			log.Printf("Received a message: %s", d.Body)
	//		}
	//	}()
	//
	//	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	//	<-forever
	//	checkMethods := map[string][]string {"spark": {"status"}}
}

func doPOSTRequest(url string, token string, jsonBody io.Reader) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, jsonBody)
	if err != nil {
		fmt.Println("ERR", err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("ERR", err)
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("ERRRROR", err)
		return nil, err
	}
	return body, nil
}

