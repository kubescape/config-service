package dbhandler

import (
	"context"
	"fmt"
	"kubescape-config-service/mongo"
	"kubescape-config-service/types"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

var cachedDocuments = make(map[string]interface{})

func AddCachedDocument[T types.DocContent](cacheKey, collection string, queryFilter bson.D, updateInterval time.Duration) {
	cachedDocuments[cacheKey] = newCachedDocument[T](collection, queryFilter, updateInterval)
}

func GetCachedDocument[T types.DocContent](cacheKey string) (T, error) {
	if cachedDoc, ok := cachedDocuments[cacheKey]; ok {
		return cachedDoc.(*cachedDocument[T]).get()
	}
	return nil, fmt.Errorf("cached document %s not found", cacheKey)
}

type cachedDocument[T types.DocContent] struct {
	doc              T
	lastRefreshError error
	timeUpdated      time.Time
	mutex            sync.RWMutex
	updateInterval   time.Duration
	queryFilter      bson.D
	collection       string
}

func newCachedDocument[T types.DocContent](collection string, queryFilter bson.D, updateInterval time.Duration) *cachedDocument[T] {
	return &cachedDocument[T]{
		doc:            nil,
		updateInterval: updateInterval,
		queryFilter:    queryFilter,
		collection:     collection,
		mutex:          sync.RWMutex{},
		timeUpdated:    time.Time{},
	}
}

func (c *cachedDocument[T]) get() (T, error) {
	c.refresh()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.doc, c.lastRefreshError
}

func (c *cachedDocument[T]) refresh() {
	if time.Since(c.timeUpdated) > c.updateInterval {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		//check if not updated by another thread
		if time.Since(c.timeUpdated) > c.updateInterval {
			var doc T
			if err := mongo.GetReadCollection(c.collection).FindOne(context.Background(), c.queryFilter).Decode(&doc); err != nil {
				zap.L().Error("Failed to refresh cached document", zap.Error(err), zap.String("collection", c.collection), zap.Any("queryFilter", c.queryFilter))
				c.lastRefreshError = err
				return
			}
			c.doc = doc
			c.lastRefreshError = nil
			c.timeUpdated = time.Now()
		}
	}
}