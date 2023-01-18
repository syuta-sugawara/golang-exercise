package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// DefaultHTTPGetAddress Default Address
	DefaultHTTPGetAddress = "https://checkip.amazonaws.com"

	// ErrNoIP No IP found in response
	ErrNoIP = errors.New("No IP in HTTP response")

	// ErrNon200Response non 200 status code in response
	ErrNon200Response = errors.New("Non 200 Response found")
)

func connectDb() *gorm.DB {
	HOST := os.Getenv("HOST")
	DBUSER := os.Getenv("DBUSER")
	PASS := os.Getenv("PASS")
	DB_NAME := os.Getenv("DB_NAME")
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", HOST,DBUSER,PASS,DB_NAME)
	db, err := gorm.Open(postgres.Open(dsn),&gorm.Config{})
	if err != nil {
		log.Println("db failed")
		panic(err)
	}
	return db
}

type ResponseJson struct {
	Date string
	Number int
}

type InfectedPeople struct {
	gorm.Model
	Date time.Time
	Number int
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	resp, err := http.Get(DefaultHTTPGetAddress)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if resp.StatusCode != 200 {
		return events.APIGatewayProxyResponse{}, ErrNon200Response
	}

	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if len(ip) == 0 {
		return events.APIGatewayProxyResponse{}, ErrNoIP
	}

	date := request.QueryStringParameters["year"] + "-" + request.QueryStringParameters["month"] + "-" + request.QueryStringParameters["day"]
	
	db := connectDb() 
	db.Logger = db.Logger.LogMode(logger.Info)
	sqlDB , err := db.DB()
	if err != nil{
		panic(err)
	}
	defer sqlDB.Close()

	var infectedPeople InfectedPeople
	loc, _ := time.LoadLocation("Asia/Tokyo")
	t, e := time.ParseInLocation("2006-01-02", date , loc)
	if e != nil {
		log.Println(e)
	}

	db.Where("Date = ?", t).Find(&infectedPeople)

	response := ResponseJson{
		Date: date,
		Number: infectedPeople.Number,
	}

	jsonBytes,_ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		Body:       string(jsonBytes),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
