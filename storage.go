package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"cloud.google.com/go/storage"
)

type Clients struct {
	AWS *s3.S3
	GCP *storage.Client
}

func InitStorageClients(c Config) (*Clients, error) {
	awsClient, err := initAWS(&c)
	if err != nil {
		return nil, err
	}

	gcpClient, err := initGCP(&c)

	if err != nil {
		return nil, err
	}

	return &Clients{
		AWS: awsClient,
		GCP: gcpClient,
	}, nil
}

type S3FileStorage struct {
	client     *s3.S3
	bucketName string
}

func NewS3FileStorage(client *s3.S3, bucketName string) *S3FileStorage {
	return &S3FileStorage{
		client:     client,
		bucketName: bucketName,
	}
}

func (s *S3FileStorage) Store(filePath string, fileContent []byte) error {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(filePath),
		Body:          aws.ReadSeekCloser(bytes.NewReader(fileContent)),
		ContentLength: aws.Int64(int64(len(fileContent))),
		ContentType:   aws.String("application/octet-stream"),
	}

	_, err := s.client.PutObjectWithContext(aws.BackgroundContext(), input)
	if err != nil {
		return err
	}

	return nil
}

func (s *S3FileStorage) SignedURL(filePath string) (string, error) {
	req, _ := s.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(filePath),
	})

	url, err := req.Presign(60 * time.Minute)
	if err != nil {
		return "", err
	}

	return url, nil
}

type GCSFileStorage struct {
	client     *storage.Client
	bucketName string
}

func NewGCSFileStorage(client *storage.Client, bucketName string) *GCSFileStorage {
	return &GCSFileStorage{
		client:     client,
		bucketName: bucketName,
	}
}

func (g *GCSFileStorage) Store(filePath string, fileContent []byte) error {
	ctx := context.Background()

	bucket := g.client.Bucket(g.bucketName)

	object := bucket.Object(filePath)

	writer := object.NewWriter(ctx)
	defer writer.Close()

	_, err := io.Copy(writer, bytes.NewReader(fileContent))
	if err != nil {
		return err
	}

	return nil
}

func (g *GCSFileStorage) SignedURL(filePath string) (string, error) {
	ctx := context.Background()

	bucket := g.client.Bucket(g.bucketName)
	object := bucket.Object(filePath)

	attrs, err := object.Attrs(ctx)
	if err != nil {
		return "", err
	}

	url := attrs.MediaLink
	return url, nil
}

func initGCP(c *Config) (*storage.Client, error) {
	gcpCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	if gcpCreds == "" {
		log.Println("GCP credentials not found. Google Cloud Storage client will not be initialized.")
		return nil, nil
	}

	log.Println("GCP credentials found, initializing Google Cloud Storage client...")

	ctx := context.Background()
	client, err := storage.NewClient(ctx)

	if err != nil {
		log.Fatalf("Error creating GCP client: %v", err)
		return nil, err
	}

	bucket := client.Bucket(c.Storage.Google.Bucket)
	_, err = bucket.Attrs(ctx)

	if err != nil {
		log.Printf("Error accessing Google Cloud Storage bucket: %v", err)
		return nil, err
	}

	log.Printf("Google Cloud Storage bucket active: %s", c.Storage.Google.Bucket)
	return client, nil
}

func initAWS(c *Config) (*s3.S3, error) {
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if awsAccessKey == "" || awsSecretKey == "" {
		log.Println("AWS credentials not found. S3 client will not be initialized.")
		return nil, nil
	}

	log.Println("AWS credentials found, initializing S3 client...")
	log.Println("AWS Region: ", c.Storage.S3.Region)
	log.Println("S3 Bucket: ", c.Storage.S3.Bucket)
	log.Println("AWS Access Key: ", awsAccessKey)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(c.Storage.S3.Region),
	})

	if err != nil {
		log.Fatalf("Error starting AWS session: %v", err)
		return nil, err
	}

	s3Client := s3.New(sess)

	_, err = s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(c.Storage.S3.Bucket),
	})

	if err != nil {
		log.Printf("Error accessing S3 bucket: %v", err)
		return nil, err
	}

	log.Printf("S3 bucket active: %s", c.Storage.S3.Bucket)
	return s3Client, nil
}
