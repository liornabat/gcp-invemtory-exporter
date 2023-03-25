package storage

import (
	"bytes"
	"cloud.google.com/go/storage"
	"context"
	"github.com/liornabat/gcp_inventory_exporter/pkg/logger"
	"github.com/liornabat/gcp_inventory_exporter/project"
	"google.golang.org/api/iterator"
	"io"
	"sync"
)

var bucketHeader = []string{
	"Project",
	"Name",
	"Location",
	"Storage Class",
	"Creation Timestamp",
}

type Storage struct {
	client    *storage.Client
	projectID string
}

func NewStorage(ctx context.Context, projectId string) (*Storage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Storage{
		client:    client,
		projectID: projectId,
	}, nil
}

func (s *Storage) Close() error {
	return s.client.Close()
}

func (s *Storage) BucketExistsOrCreate(ctx context.Context, bucketName string) error {
	_, err := s.client.Bucket(bucketName).Attrs(ctx)
	if err == storage.ErrBucketNotExist {
		return s.CreateBucket(ctx, bucketName)
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) CreateBucket(ctx context.Context, bucketName string) error {
	return s.client.Bucket(bucketName).Create(ctx, s.projectID, nil)
}

func (s *Storage) ListBuckets(ctx context.Context) ([]string, error) {
	var buckets []string
	it := s.client.Buckets(ctx, s.projectID)
	for {
		bucketAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		buckets = append(buckets, bucketAttrs.Name)
	}
	return buckets, nil

}
func (s *Storage) SaveFile(ctx context.Context, bucketName, objectName string, objectData []byte) error {
	bucket := s.client.Bucket(bucketName)
	wc := bucket.Object(objectName).NewWriter(ctx)
	wc.ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

	if _, err := io.Copy(wc, bytes.NewReader(objectData)); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}

	return nil
}

func (s *Storage) GetStorageInventory(ctx context.Context, projectsId []*project.Project, log *logger.Logger) ([][]string, error) {
	log.Infof("Getting Cloud Store inventory")
	defer log.Infof("Done Cloud Store inventory")
	var inventory [][]string
	inventory = append(inventory, bucketHeader)
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(len(projectsId))
	for _, projectId := range projectsId {
		go func(projectId *project.Project) {
			defer wg.Done()
			var localInventory [][]string
			log.Infof("Getting Cloud Store inventory for project %s", projectId.Name)
			bucketsIterator := s.client.Buckets(ctx, projectId.ID)
			for {
				bucketAttrs, err := bucketsIterator.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Errorf("Error getting bucket attributes for project %s: %v", projectId.Name, err)
					break
				}
				localInventory = append(localInventory, []string{
					projectId.Name,
					bucketAttrs.Name,
					bucketAttrs.Location,
					bucketAttrs.StorageClass,
					bucketAttrs.Created.String(),
				})
			}
			mutex.Lock()
			inventory = append(inventory, localInventory...)
			mutex.Unlock()
		}(projectId)
	}
	wg.Wait()
	return inventory, nil
}
