package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"bufio"
	"io"
	"os/exec"

	"regexp"

	"strconv"

	"github.com/go-redis/redis/v8"

	"github.com/joho/godotenv"

	"os"

	"log"
	"time"

	"encoding/json"
)

type dataStruct struct {
	ID           string `json:"id"`
	Device       string `json:"device"`
	ReturnedData string `json:"returnedData"`
	Value        string `json:"value"`
}

var datas = []dataStruct{
	{ID: "1", Device: "0x5432", ReturnedData: "[{sdffg:asdf}]", Value: "{'temp': 1}"},
	{ID: "2", Device: "0x5462", ReturnedData: "[{sdffg:asdf}]", Value: "{'temp': 1}"},
	{ID: "3", Device: "0x5462", ReturnedData: "[{sdffg:asdf}]", Value: "{'temp': 1}"},
	{ID: "4", Device: "0x5432", ReturnedData: "[{sdffg:asdf}]", Value: "{'temp': 1}"},
}

func index(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, datas)
}

// type URI struct {
// 	Details string `json:"name" uri:"details" binding:"required"`
// }

// type Binding interface {
// 	Name() string
// 	Bind(*http.Request, interface{}) error
// }

// type Toml struct {
// }

// // return the name of binding
// func (t Toml) Name() string {
// 	return "toml"
// }

// // parse request
// func (t Toml) Bind(request *http.Request, i interface{}) error {
// 	// using go-toml package
// 	tD := toml.NewDecoder(request.Body)
// 	// decoding the interface
// 	//return tD.Decode(i)
// 	fmt.Println(tD.Decode(i))
// 	return tD
// }

type dataJSON struct {
	app_eui string `json:"app_eui"`
	dc      string `json:"dc"`
	decoded string `json:"decoded"`
}

type DataValue struct {
	Data     string `json: "data"`
	DeviceId string `json: "deviceId"`
}

func copyOutput(r io.Reader, metaData map[string]interface{}) {
	scanner := bufio.NewScanner(r)
	var data string
	for scanner.Scan() {
		//fmt.Println(scanner.Text())
		// if scanner.Text() != "" {
		// 	fmt.Println(data)
		// 	data += scanner.Text()
		// }
		// r = regexp.MustCompile("p([a-z]+)ch")
		// fmt.Println("regexp:", r)
		// match, _ := regexp.MatchString("p([a-z]+)ch", "peach")
		// fmt.Println(match)
		// r, _ := regexp.Compile("p([a-z]+)ch")
		// fmt.Println(r.FindString("peach punch"))
		if data != " " {
			data += scanner.Text()
		}
	}
	// match, _ := regexp.MatchString("p([a-z]+)ch", "peach")
	// fmt.Println(match)
	pop, _ := regexp.Compile(`\{.*?\}`)
	result := pop.FindString(data)
	result1 := strings.Trim(result, " ")
	fmt.Println(result1)

	fmt.Println("End data: ", data)
	//string(hash.Sum([]byte(data)))

	encodedSaveData, err := json.Marshal(&DataValue{Data: result1, DeviceId: fmt.Sprintf("%v", metaData["dev_eui"])})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(encodedSaveData)

	var currentIndex = retreiveData("index")
	indexInt, _ := strconv.Atoi(currentIndex)
	var index = indexInt + 1
	SaveClient("index", strconv.Itoa(index))
	var indexString = "key" + strconv.Itoa(index)
	SaveClient(indexString, string(encodedSaveData))
	// r, _ := regexp.Compile("p([a-z]+)ch")
	// fmt.Println(r.FindString("peach punch"))
	//return data
}

func uploadData(c *gin.Context) {
	var data map[string]interface{}

	//json.Unmarshal([]byte(c.Request.Body), &data)
	//var responseData dataJSON

	//dec := json.NewDecoder(c.Request.Body)
	//fmt.Println(dec)
	//err := dec.Decode(&responseData)

	// if err != nil {
	// 	log.Println(err)
	// }
	//heliumData := dataJSON{}

	//log.Println(responseData.app_eui)

	// reads the json and prints it
	// body, _ := ioutil.ReadAll(c.Request.Body)
	// bodyString := string(body)
	//log.Println(bodyString)

	//println(string(c.Request.Body))
	if err := c.BindJSON(&data); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	result := data["decoded"].(map[string]interface{})["payload"].(map[string]interface{})
	str := fmt.Sprintf("%v", result["data"])

	fmt.Println(str)
	cmd := exec.Command("python3", "../mqps/config_decoder.py", "-p", str)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	if stdout != nil {
		go copyOutput(stdout, data) // fix this function
	} else {
		go copyOutput(stderr, data) // fix this function
	}

	cmd.Wait()

	c.IndentedJSON(http.StatusOK, &data)

}

type queryMessage struct {
	message []string `json: "message"`
}

func queryData(c *gin.Context) {
	var querys = []string{}
	//hasFirst := c.Request.URL.Query().Has("time")
	paramPairs := c.Request.URL.Query()
	//fmt.Println(time)
	for key, values := range paramPairs {
		if key == "time" {
			data, err := strconv.Atoi(values[0])
			indexId, _ := strconv.Atoi(retreiveData("index"))
			if err != nil {
				panic(err)
			}
			if data > 90 {
				//fmt.Println("data did not send")
				//var message = errorMessage{errorMessage: "can't return that much data"}
				//message = {"error": "can't return that much data"}
				//message.message = "can't return that much data"
				c.IndentedJSON(http.StatusNotFound, gin.H{"error": "can't return that much data"})
			} else if data > indexId {
				c.IndentedJSON(http.StatusNotFound, gin.H{"error": "requesting more data then available"})
			} else {
				for i := 1; i <= data; i++ {
					var indexString = "key" + strconv.Itoa(i)
					var result = retreiveData(indexString)
					//fmt.Println(result)

					querys = append(querys, result)

				}
				c.IndentedJSON(http.StatusOK, gin.H{"message": querys})
			}
		}
		//fmt.Printf("key = %v, value(s) = %v\n", key, values)
	}
}

var ctx = context.Background()

// Handles retrieving the url for redis
func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}

// Retreives data from the redis database
func retreiveData(key string) string {
	opt, _ := redis.ParseURL(goDotEnvVariable("redisURL"))
	rdb := redis.NewClient(opt)

	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("key", val)
	return val
}

// Save data to the redis database
func SaveClient(key string, value string) {
	opt, _ := redis.ParseURL(goDotEnvVariable("redisURL"))
	rdb := redis.NewClient(opt)

	// rdb := redis.NewClient(&redis.Options{
	// 	Addr:     "rediss://:03390b565b794bb196c1704e06ebb212",
	// 	Password: "03390b565b794bb196c1704e06ebb212", // no password set
	// 	//DB:       0,  // use default DB
	// })

	if key == "index" {
		err := rdb.Set(ctx, key, value, 0).Err()
		if err != nil {
			panic(err)
		}
	} else {
		_, err := rdb.Set(ctx, key, value, ((24 * 90) * time.Hour)).Result()
		if err != nil {
			panic(err)
		}
	}
	//err := rdb.Set(ctx, key, value, 0).Err()

}

func main() {
	//gin.SetMode(gin.DebugMode)
	router := gin.Default()
	router.GET("/", index)
	router.POST("/upload", uploadData)
	router.GET("/data", queryData)

	router.Run("localhost:8000")
}
