package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"bufio"
	"io"
	"os/exec"

	"regexp"

	"crypto/md5"

	"strconv"

	"github.com/go-redis/redis/v8"

	"github.com/joho/godotenv"

	"os"

	"log"
	"time"
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

type dataValue struct {
	data string
}

func copyOutput(r io.Reader) {
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
		data += scanner.Text()
	}
	// match, _ := regexp.MatchString("p([a-z]+)ch", "peach")
	// fmt.Println(match)
	pop, _ := regexp.Compile(`\{.*?\}`)
	result := pop.FindString(data)
	fmt.Println(pop.FindString(data))

	fmt.Println("End data: ", data)

	hash := md5.New()
	hash.Write([]byte(data))
	//string(hash.Sum([]byte(data)))

	var currentIndex = retreiveData("index")
	indexInt, _ := strconv.Atoi(currentIndex)
	var index = indexInt + 1
	SaveClient("index", strconv.Itoa(index))
	var indexString = "key" + strconv.Itoa(index)
	SaveClient(indexString, result)
	// r, _ := regexp.Compile("p([a-z]+)ch")
	// fmt.Println(r.FindString("peach punch"))
	//return data
}

func uploadData(c *gin.Context) {
	var data map[string]interface{}

	//json.Unmarshal([]byte(c.Request.Body), &data)
	//var responseData dataJSON

	// dec := json.NewDecoder(c.Request.Body)
	// fmt.Println(dec)
	// err := dec.Decode(&responseData)

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
		go copyOutput(stdout)
	} else {
		go copyOutput(stderr)
	}

	cmd.Wait()

	c.IndentedJSON(http.StatusOK, &data)

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

	router.Run("localhost:8000")
}
