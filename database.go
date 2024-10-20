package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/boltdb/bolt"
)

type Page struct {
	CurrentPage int
	Limit       int
	TotalPages  int
	Items       []Video
}

type Database interface {
	SaveVideo(ctx context.Context, video Video) error
	GetVideo(ctx context.Context, videoID string) (Video, error)
	GetVideos(ctx context.Context, page int, size int) (Page, error)
}

func NewBoltDB(databasePath string) *BoltDB {
	return &BoltDB{
		DatabasePath: databasePath,
	}
}

type BoltDB struct {
	DatabasePath string
}

func (b *BoltDB) SaveVideo(ctx context.Context, video Video) error {
	db, err := bolt.Open(b.DatabasePath, 0600, nil)

	if err != nil {
		log.Fatal(err)
		return err
	}

	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("videos"))

		if err != nil {
			return err
		}

		json, err := json.Marshal(video)
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(video.ID), []byte(json))

		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (b *BoltDB) GetVideo(ctx context.Context, videoID string) (Video, error) {
	db, err := bolt.Open(b.DatabasePath, 0600, nil)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	var video Video

	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("videos"))

		if bucket == nil {
			return nil
		}

		data := bucket.Get([]byte(videoID))
		return json.Unmarshal(data, &video)
	})

	return video, err
}

func (b *BoltDB) GetVideos(ctx context.Context, page int, size int) (Page, error) {
	db, err := bolt.Open(b.DatabasePath, 0600, nil)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	var videos []Video

	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("videos"))

		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var video Video

			err := json.Unmarshal(v, &video)
			if err != nil {
				return err
			}

			videos = append(videos, video)
		}

		return nil
	})

	totalPages := len(videos) / size

	if len(videos)%size != 0 {
		totalPages++
	}

	start := (page - 1) * size
	end := start + size

	if end > len(videos) {
		end = len(videos)
	}

	return Page{
		CurrentPage: page,
		Limit:       size,
		TotalPages:  totalPages,
		Items:       videos[start:end],
	}, err
}
