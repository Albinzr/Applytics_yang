package reader

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"runtime/debug"
	"time"

	db "applytics.in/yang/src/database"
	util "applytics.in/yang/src/helpers"
	kafka "github.com/Albinzr/kafkaGo"
	lz "github.com/Albinzr/lzGo"
)

var env = util.LoadEnvConfig()

var kafkaConfig = &kafka.Config{
	Topic:                 env.KafkaTopic,
	Partition:             env.Partition,
	URL:                   env.KafkaURL,
	GroupID:               env.GroupID,
	MinBytes:              env.MinBytes,
	MaxBytes:              env.MaxBytes,
	ReadWithLimitFinished: true,
}

var dbConfig = &db.Config{
	URL:          env.MongoURL,
	DatabaseName: env.DatabaseName,
}

//Start :- start reader
func Start() {
	fmt.Printf("%+v\n", kafkaConfig)
	//fmt.Printf("%+v\n", dbConfig)
	go func() {
		log.Fatal(http.ListenAndServe(":2000", nil))
	}()

	readFromKafkaUpdateDB()
}

func readFromKafkaUpdateDB() {
	util.LogInfo("trying to connect to mongo DB")
	err := dbConfig.Init()
	util.LogInfo("connection call made")
	if err != nil {
		util.LogError("Database connection issue", err)
		time.AfterFunc(30*time.Second, readFromKafkaUpdateDB)
		return
	}

	util.LogInfo("mongo DB connected")
	readFromKafka()
}

func readFromKafka() {
	//Start kafka
	err := startKafka()
	if err != nil {
		util.LogError("Kafka connection issue", err)
		//if error try after (T) sec
		time.AfterFunc(5*time.Second, readFromKafka)
	}
	util.LogInfo("Starting reading message from kafka")

	// kafkaConfig.Reader(kafkaReaderCallback)
	for {
		if kafkaConfig.ReadWithLimitFinished {
			kafkaConfig.ReaderWithLimit(10, kafkaReaderCallback)
			fmt.Println("in..........................................", time.Now())
		}
	}

}

func getMsg(msg string) (string, error) {
	msg, err := lz.DecompressFromBase64(msg)
	return msg, err
}

func kafkaReaderCallback(reader kafka.Reader, message kafka.Message) {

	enMsg := string(message.Value)
	var err error
	var msg string
	if enMsg[0:2] == "en" {
		msg, err = getMsg(enMsg[3:])
		if err != nil || enMsg == "" {
			fmt.Println("decomperssion failed*------------------------------------>")
		}
	} else {
		msg = enMsg[3:]
	}

	var jsonInterface map[string]interface{}
	err = json.Unmarshal([]byte(msg), &jsonInterface)
	if err != nil {
		fmt.Println("JSON CONVERSION Failled....................................")
	}
	if jsonInterface["type"] == nil {
		fmt.Println(jsonInterface)
	}

	switch jsonInterface["type"] {
	case "session":
		err = dbConfig.Insert("record", jsonInterface)
	case "event":
		err = dbConfig.Insert("subRecord", jsonInterface)
	case "close":
		err = dbConfig.UpdateSession("record", jsonInterface)
	case "track":
		err = dbConfig.Insert("track", jsonInterface)
	case "feedback":
		err = dbConfig.Insert("feedback", jsonInterface)
	case "poll":
		err = dbConfig.Insert("poll", jsonInterface)
	case "form":
		err = dbConfig.Insert("form", jsonInterface)
	case "surveys":
		err = dbConfig.Insert("surveys", jsonInterface)
	default:
		util.LogInfo("wrong data detected _______________********_______________", len(msg), "incoming data", jsonInterface)
	}

	// commitKafkaMessage(err, reader, message)
	//printMemUsage()

}

// func commitKafkaMessage(err error, reader kafka.Reader, message kafka.Message) {
// 	if err == nil {
// 		kafka.Commit(reader, message)
// 	} else {
// 		//TODO: - if duplicare remove else set up a retry system (3 times) then delete
// 		fmt.Println("err-------->", err)
// 		kafka.Commit(reader, message)
// 	}
// }

func startKafka() error {
	if kafkaConfig.IsKafkaReady() {
		util.LogInfo("Connected to kafka")
		return nil
	}
	err := errors.New("Cannot connect to kafka")
	return err
}

//PrintMemUsage -test
func printMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
	fmt.Printf("\tMemory Freed = %v\n", bToMb(m.Frees))

	runtime.GC()
	debug.FreeOSMemory()
}
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
