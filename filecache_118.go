//go:build go1.18

package filecache

import (
	"os"
	"time"
)

func cacheFile(path string, maxSize int64) (itm *cacheItem, err error) {
	fi, err := os.Stat(path)
	if err != nil {
		return
	} else if fi.Mode().IsDir() {
		return nil, ItemIsDirectory
	} else if fi.Size() > maxSize {
		return nil, ItemTooLarge
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	itm = &cacheItem{
		content:    content,
		Size:       fi.Size(),
		Modified:   fi.ModTime(),
		Lastaccess: time.Now(),
	}
	return
}

// ReadFile retrieves the file named by 'name'.
// If the file is not in the cache, load the file and cache the file in the
// background. If the file was not in the cache and the read was successful,
// the error ItemNotInCache is returned to indicate that the item was pulled
// from the filesystem and not the cache, unless the SquelchItemNotInCache
// global option is set; in that case, returns no error.
func (cache *FileCache) ReadFile(name string) (content []byte, err error) {
	if cache.InCache(name) {
		content, _ = cache.GetItem(name)
	} else {
		go cache.Cache(name)
		content, err = os.ReadFile(name)
		if err == nil && !SquelchItemNotInCache {
			err = ItemNotInCache
		}
	}
	return
}
