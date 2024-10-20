package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

type Config struct {
	BoltLocation string `yaml:"bolt_location"`
	Storage      struct {
		S3 struct {
			Bucket string `yaml:"bucket"`
			Region string `yaml:"region"`
		} `yaml:"s3"`
		Google struct {
			Bucket  string `yaml:"bucket"`
			Project string `yaml:"project"`
			Region  string `yaml:"region"`
		} `yaml:"google"`
	} `yaml:"storage"`
}

func main() {
	data, err := os.ReadFile("./config.yaml")
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	var config Config

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Error unmarshaling YAML: %v", err)
	}

	if config.BoltLocation == "" {
		log.Fatalf("Bolt location not set")
	}

	codecAvailable, err := SelectH264Encoder()
	if err != nil {
		panic(err)
	}

	if codecAvailable == "" {
		panic("H264 codec not available")
	}

	log.Printf("H264 codec available: %s", codecAvailable)

	storageClients, err := InitStorageClients(config)
	fileStorages := make([]FileStorage, 0)

	if err != nil {
		log.Fatalf("Error initializing storage clients: %v", err)
	}

	if storageClients.AWS != nil {
		fileStorages = append(fileStorages, NewS3FileStorage(storageClients.AWS, config.Storage.S3.Bucket))
	}

	if storageClients.GCP != nil {
		fileStorages = append(fileStorages, NewGCSFileStorage(storageClients.GCP, config.Storage.Google.Bucket))
	}

	if len(fileStorages) == 0 {
		panic("No storage clients initialized")
	}

	db := NewBoltDB(config.BoltLocation)

	api := NewAPI(VideoService{
		Storages: fileStorages,
		Database: db,
	})

	router := gin.Default()
	router.POST("upload", api.HandleUpload)
	router.GET("video/:id", api.GetVideo)
	router.GET("video/:id/manifest", api.GetVideoURL)
	router.Run(":8080")
}
