package filecache

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func getTimeExpiredCacheItem() *cacheItem {
	TwoHours, err := time.ParseDuration("-2h")
	if err != nil {
		panic(err.Error())
	}
	itm := new(cacheItem)
	itm.content = []byte("this cache item should be expired")
	itm.Lastaccess = time.Now().Add(TwoHours)
	return itm
}

func (cache *FileCache) _add_cache_item(name string, itm *cacheItem) {
	cache.items[name] = itm
}

func dumpModTime(name string) {
	fi, err := os.Stat(name)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("[-] %s mod time: %v\n", name, fi.ModTime().Unix())
}

func writeTempFile(t *testing.T, contents string) string {
	tmpf, err := ioutil.TempFile("", "fctest")
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't create temporary file: ", err.Error())
		t.Fail()
	}
	name := tmpf.Name()
	tmpf.Close()
	err = ioutil.WriteFile(name, []byte(contents), 0600)
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't write temporary file: ", err.Error())
		os.Remove(name)
		t.Fail()
		name = ""
		return name
	}
	return name
}

func TestCacheStartStop(t *testing.T) {
	fmt.Printf("[+] testing cache start up and shutdown: ")
	cache := NewDefaultCache()
	if err := cache.Start(); err != nil {
		fmt.Println("failed")
		fmt.Println("[!] cache failed to start: ", err.Error())
	}
	time.Sleep(1 * time.Second)
	cache.Stop()
	fmt.Println("ok")
}

func TestTimeExpiration(t *testing.T) {
	fmt.Printf("[+] ensure item expires after ExpireItem: ")
	cache := NewDefaultCache()
	if err := cache.Start(); err != nil {
		fmt.Println("failed")
		fmt.Println("[!] cache failed to start: ", err.Error())
	}
	name := "expired"
	itm := getTimeExpiredCacheItem()
	cache._add_cache_item(name, itm)
	if !cache.expired(name) {
		fmt.Println("failed")
		fmt.Println("[!] item should have expired!")
		t.Fail()
	} else {
		fmt.Println("ok")
	}
	cache.Stop()
}

func TestTimeExpirationUpdate(t *testing.T) {
	fmt.Printf("[+] ensure accessing an item prevents it from expiring: ")
	cache := NewDefaultCache()
	cache.ExpireItem = 2
	cache.Every = 1
	if err := cache.Start(); err != nil {
		fmt.Println("failed")
		fmt.Println("[!] cache failed to start: ", err.Error())
	}
	testFile := "filecache.go"
	cache.CacheNow(testFile)
	if !cache.InCache(testFile) {
		fmt.Println("failed")
		fmt.Println("[!] failed to cache file")
		cache.Stop()
		t.FailNow()
	}
	time.Sleep(1500 * time.Millisecond)
	contents, err := cache.ReadFile(testFile)
	if err != nil || !ValidateDataMatchesFile(contents, testFile) {
		fmt.Println("failed")
		fmt.Printf("[!] file read failed: ")
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("cache contents do not match file")
		}
		cache.Stop()
		t.FailNow()
	}
	time.Sleep(1 * time.Second)
	if !cache.InCache(testFile) {
		fmt.Println("failed")
		fmt.Println("[!] item should not have expired")
		t.Fail()
	} else {
		fmt.Println("ok")
	}
	cache.Stop()
}

func TestFileChanged(t *testing.T) {
	fmt.Printf("[+] validate file modification expires item: ")
	cache := NewDefaultCache()
	if err := cache.Start(); err != nil {
		fmt.Println("failed")
		fmt.Println("[!] cache failed to start: ", err.Error())
	}

	name := writeTempFile(t, "lorem ipsum blah blah")
	if name == "" {
		fmt.Println("failed!")
		fmt.Println("[!] failed to cache item")
		cache.Stop()
		t.FailNow()
	} else if err := cache.CacheNow(name); err != nil {
		fmt.Println("failed!")
		fmt.Println("[!] failed to cache item")
		cache.Stop()
		t.FailNow()
	} else if !cache.InCache(name) {
		fmt.Println("failed")
		fmt.Println("[!] failed to cache item")
		os.Remove(name)
		cache.Stop()
		t.FailNow()
	}
	time.Sleep(1 * time.Second)
	err := ioutil.WriteFile(name, []byte("after modification"), 0600)
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't write temporary file: ", err.Error())
		t.Fail()
	} else if !cache.changed(name) {
		fmt.Println("failed")
		fmt.Println("[!] item should have expired!")
		t.Fail()
	}
	os.Remove(name)
	cache.Stop()
	fmt.Println("ok")
}

func TestCache(t *testing.T) {
	fmt.Printf("[+] testing asynchronous file caching: ")
	cache := NewDefaultCache()
	if err := cache.Start(); err != nil {
		fmt.Println("failed")
		fmt.Println("[!] cache failed to start: ", err.Error())
	}
	name := writeTempFile(t, "lorem ipsum akldfjsdlf")
	if name == "" {
		cache.Stop()
		t.FailNow()
	} else if cache.InCache(name) {
		fmt.Println("failed")
		fmt.Println("[!] item should not be in cache yet!")
		os.Remove(name)
		cache.Stop()
		t.FailNow()
	}

	cache.Cache(name)

	var (
		delay int
		ok    bool
		dur   time.Duration
		step  = 10
		stop  = 500
	)
	dur, err := time.ParseDuration(fmt.Sprintf("%dµs", step))
	if err != nil {
		panic(err.Error())
	}

	for ok = cache.InCache(name); !ok; ok = cache.InCache(name) {
		time.Sleep(dur)
		delay += step
		if delay >= stop {
			break
		}
	}

	if !ok {
		fmt.Println("failed")
		fmt.Printf("\t[*] cache check stopped after %dµs\n", delay)
		t.Fail()
	} else {
		fmt.Println("ok")
		fmt.Printf("\t[*] item cached in %dµs\n", delay)
	}
	cache.Stop()
	os.Remove(name)

}

func TestExpireAll(t *testing.T) {
	fmt.Printf("[+] testing background expiration: ")
	cache := NewDefaultCache()
	cache.Every = 1
	cache.ExpireItem = 2
	if err := cache.Start(); err != nil {
		fmt.Println("failed")
		fmt.Println("[!] cache failed to start: ", err.Error())
	}

	name := writeTempFile(t, "this is a first file and some stuff should go here")
	if name == "" {
		cache.Stop()
		t.Fail()
	}
	name2 := writeTempFile(t, "this is the second file")
	if name2 == "" {
		cache.Stop()
		os.Remove(name)
		t.Fail()
	}
	if t.Failed() {
		t.FailNow()
	}

	cache.CacheNow(name)
	time.Sleep(500 * time.Millisecond)
	cache.CacheNow(name2)
	time.Sleep(500 * time.Millisecond)

	err := ioutil.WriteFile(name2, []byte("lorem ipsum dolor sit amet."), 0600)
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't write temporary file: ", err.Error())
		t.FailNow()
	}

	if !t.Failed() {
		time.Sleep(1250 * time.Millisecond)
		if cache.Size() > 0 {
			fmt.Println("failed")
			fmt.Printf("[!] %d items still in cache", cache.Size())
			t.Fail()
		}
	}

	if !t.Failed() {
		fmt.Println("ok")
	}
	os.Remove(name)
	os.Remove(name2)
	cache.Stop()
}

func destroyNames(names []string) {
	for _, name := range names {
		os.Remove(name)
	}
}

func TestExpireOldest(t *testing.T) {
	fmt.Printf("[+] validating item limit on cache: ")
	cache := NewDefaultCache()
	cache.MaxItems = 5
	if err := cache.Start(); err != nil {
		fmt.Println("failed")
		fmt.Println("[!] cache failed to start: ", err.Error())
	}

	names := make([]string, 0)
	for i := 0; i < 1000; i++ {
		name := writeTempFile(t, fmt.Sprintf("file number %d\n", i))
		if t.Failed() {
			break
		}
		names = append(names, name)
		cache.CacheNow(name)
	}

	if !t.Failed() && cache.Size() > cache.MaxItems {
		fmt.Println("failed")
		fmt.Printf("[!] %d items in cache (limit should be %d)",
			cache.Size(), cache.MaxItems)
		t.Fail()
	}
	if !t.Failed() {
		fmt.Println("ok")
	}
	cache.Stop()
	destroyNames(names)
}

func TestNeverExpire(t *testing.T) {
	fmt.Printf("[+] validating no time limit expirations: ")
	cache := NewDefaultCache()
	cache.ExpireItem = 0
	cache.Every = 1
	if err := cache.Start(); err != nil {
		fmt.Println("failed")
		fmt.Println("[!] cache failed to start: ", err.Error())
	}

	tmpf, err := ioutil.TempFile("", "fctest")
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't create temporary file: ", err.Error())
		t.FailNow()
	}
	name := tmpf.Name()
	tmpf.Close()

	err = ioutil.WriteFile(name, []byte("lorem ipsum dolor sit amet."), 0600)
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't write temporary file: ", err.Error())
		os.Remove(name)
		cache.Stop()
		t.FailNow()
	}
	cache.Cache(name)
	time.Sleep(2 * time.Second)
	if !cache.InCache(name) {
		fmt.Println("failed")
		fmt.Println("[!] item should not have been expired")
		t.Fail()
	} else {
		fmt.Println("ok")
	}
	cache.Stop()
	os.Remove(name)
}

func BenchmarkAsyncCaching(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cache := NewDefaultCache()
		if err := cache.Start(); err != nil {
			fmt.Println("[!] cache failed to start: ", err.Error())
		}

		cache.Cache("filecache.go")
		for {
			if cache.InCache("filecache.go") {
				break
			}
			<-time.After(10 * time.Microsecond)
		}
		cache.Remove("filecache.go")
		cache.Stop()
	}
}

func TestCacheReadFile(t *testing.T) {
	fmt.Printf("[+] testing transparent file reads: ")
	testFile := "filecache.go"
	cache := NewDefaultCache()
	if err := cache.Start(); err != nil {
		fmt.Println("failed")
		fmt.Println("[!] cache failed to start: ", err.Error())
	}

	if cache.InCache(testFile) {
		fmt.Println("failed")
		fmt.Println("[!] file should not be in cache yet")
		cache.Stop()
		t.FailNow()
	}

	out, err := cache.ReadFile(testFile)
	if (err != nil && err != ItemNotInCache) || !ValidateDataMatchesFile(out, testFile) {
		fmt.Println("failed")
		fmt.Printf("[!] transparent file read has failed: ")
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("file does not match cache contents")
		}
		cache.Stop()
		t.FailNow()
	}

	time.Sleep(10 * time.Millisecond)
	out, err = cache.ReadFile(testFile)
	if err != nil || !ValidateDataMatchesFile(out, testFile) {
		fmt.Println("failed")
		fmt.Println("[!] ReadFile has failed")
		t.Fail()
	} else {
		fmt.Println("ok")
	}
	cache.Stop()
}

func BenchmarkSyncCaching(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cache := NewDefaultCache()

		if err := cache.Start(); err != nil {
			fmt.Println("[!] cache failed to start: ", err.Error())
		}
		cache.CacheNow("filecache.go")
		cache.Stop()
	}
}

func ValidateDataMatchesFile(out []byte, filename string) bool {
	fileData, err := ioutil.ReadFile(filename)
	if err != nil {
		return false
	} else if len(fileData) != len(out) {
		return false
	}

	for i := 0; i < len(out); i++ {
		if out[i] != fileData[i] {
			return false
		}
	}
	return true
}
