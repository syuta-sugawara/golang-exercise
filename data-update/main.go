package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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


type InfectedPeople struct {
	gorm.Model
	Date time.Time
	Number int
}

func connectDb() *gorm.DB {
	HOST := os.Getenv("HOST")
	USER := os.Getenv("USER")
	PASS := os.Getenv("PASS")
	DB_NAME := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", HOST,USER,PASS,DB_NAME)
	db, err := gorm.Open(postgres.Open(dsn),&gorm.Config{})
	if err != nil {
		log.Println("db failed")
		panic(err)
	}
	return db
}


func fetchData(url string) [][]string{
	resp,err := http.Get(url)
		if err != nil{
			panic(err)
		}		
	defer resp.Body.Close()
	reader := csv.NewReader(resp.Body)
	data, err := reader.ReadAll()
	if err != nil {
		log.Println(err)
	}
	return data
}

func insertAllData(data [][]string,db *gorm.DB){
	for idx, element := range data {
		if idx == 0 {
			continue
		}
		loc, _ := time.LoadLocation("Asia/Tokyo")
		t, e := time.ParseInLocation("2006-01-02", element[3], loc)
		if e != nil {
			fmt.Println(e)
		}
		number , _ := strconv.Atoi(element[4]) 
		infectedPeople := InfectedPeople{
			Date: t,
			Number: number,
		}
		result := db.Create(&infectedPeople)
		if result.Error != nil {
			log.Println(result.Error)
		}
	}
}

func dataUpdate() {
    db := connectDb() 
	db.Logger = db.Logger.LogMode(logger.Info)
	sqlDB , err := db.DB()
	if err != nil{
		panic(err)
	}
	defer sqlDB.Close()
	db.AutoMigrate(&InfectedPeople{})

	// https://catalog.data.metro.tokyo.lg.jp/dataset/t000001d0000000011/resource/e2e1c3c7-1c15-44a3-a3f5-142df784c4c5
	// 日別要請者数
	URL := "https://data.stopcovid19.metro.tokyo.lg.jp/130001_tokyo_covid19_patients_per_report_date.csv"
	data :=fetchData(URL)	

	var infectedPeople []InfectedPeople
	db.Last(&infectedPeople)

	if len(infectedPeople) == 0 {
		insertAllData(data,db)
		return
	}
	t := infectedPeople[0].Date
	if t.Format("2006-01-02") != data[len(data)-1][3]{
		insertAllData(data,db)
	}
}

func main() {
	lambda.Start(dataUpdate)
}
