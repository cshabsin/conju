package keyutil

import "cloud.google.com/go/datastore"

func ToMap[dbItem any](keys []*datastore.Key, items []dbItem) map[*datastore.Key]dbItem {
	m := make(map[*datastore.Key]dbItem, len(keys))
	for i, k := range keys {
		m[k] = items[i]
	}
	return m
}
