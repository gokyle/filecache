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
	itm.Content = []byte("this cache item should be expired")
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

func TestCacheStartStop(t *testing.T) {
	fmt.Printf("[+] testing cache start up and shutdown: ")
	cache := NewDefaultCache()
	cache.Start()
	time.Sleep(1 * time.Second)
	cache.Stop()
	fmt.Println("ok")
}
func TestTimeExpiration(t *testing.T) {
	fmt.Printf("[+] ensure item expires after ExpireItem: ")
	cache := NewDefaultCache()
        cache.Start()
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

func TestFileChanged(t *testing.T) {
	fmt.Printf("[+] validate file modification expires item: ")
	cache := NewDefaultCache()
	cache.Start()
	tmpf, err := ioutil.TempFile("", "fctest")
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't create temporary file: ", err.Error())
		cache.Stop()
		t.FailNow()
	}
	tf_name := tmpf.Name()
	tmpf.Close()
	err = ioutil.WriteFile(tf_name, []byte("before modification"), 0600)
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't write temporary file: ", err.Error())
		os.Remove(tf_name)
		cache.Stop()
		t.FailNow()
	}
	if err = cache.CacheNow(tf_name); err != nil {
		fmt.Println("failed!")
		fmt.Println("[!] failed to cache item")
		os.Remove(tf_name)
		cache.Stop()
		t.FailNow()
	} else if !cache.InCache(tf_name) {
		fmt.Println("failed")
		fmt.Println("[!] failed to cache item")
		os.Remove(tf_name)
		cache.Stop()
		t.FailNow()
	}
	time.Sleep(1 * time.Second)
	err = ioutil.WriteFile(tf_name, []byte("after modification"), 0600)
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't write temporary file: ", err.Error())
		os.Remove(tf_name)
		cache.Stop()
		t.FailNow()
	}
	if !cache.changed(tf_name) {
		fmt.Println("failed")
		fmt.Println("[!] item should have expired!")
		os.Remove(tf_name)
		cache.Stop()
		t.FailNow()
	}
	os.Remove(tf_name)
	cache.Stop()
	fmt.Println("ok")
}

func TestCache(t *testing.T) {
	cache := NewDefaultCache()
	fmt.Printf("[+] testing asynchronous file caching: ")
	tmpf, err := ioutil.TempFile("", "fctest")
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't create temporary file: ", err.Error())
		t.FailNow()
	}
	name := tmpf.Name()
	tmpf.Close()

	cache.Start()
	err = ioutil.WriteFile(name, []byte("lorem ipsum dolor sit amet."), 0600)
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't write temporary file: ", err.Error())
		os.Remove(name)
		cache.Stop()
		t.FailNow()
	}

	if cache.InCache(name) {
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
	dur, err = time.ParseDuration(fmt.Sprintf("%dµs", step))
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
	} else {
		fmt.Println("ok")
		fmt.Printf("\t[*] item cached in %dµs\n", delay)
	}
	cache.Stop()
	os.Remove(name)
	if !ok {
		t.FailNow()
	}

}

func TestExpireAll(t *testing.T) {
	cache := NewDefaultCache()
	cache.Every = 1
	cache.ExpireItem = 2
	cache.Start()
	fmt.Printf("[+] testing background expiration: ")
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

	tmpf, err = ioutil.TempFile("", "fctest")
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't create temporary file: ", err.Error())
		t.FailNow()
	}
	name2 := tmpf.Name()
	tmpf.Close()

	err = ioutil.WriteFile(name2, []byte("lorem ipsum dolor sit amet."), 0600)
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't write temporary file: ", err.Error())
		os.Remove(name)
		os.Remove(name2)
		cache.Stop()
		t.FailNow()
	}

	cache.CacheNow(name)
	time.Sleep(500 * time.Millisecond)
	cache.CacheNow(name2)
	time.Sleep(500 * time.Millisecond)

	err = ioutil.WriteFile(name2, []byte("lorem ipsum dolor sit amet."), 0600)
	if err != nil {
		fmt.Println("failed")
		fmt.Println("[!] couldn't write temporary file: ", err.Error())
		os.Remove(name)
		os.Remove(name2)
		cache.Stop()
		t.FailNow()
	}

	time.Sleep(1250 * time.Millisecond)
	if cache.Size() > 0 {
		fmt.Println("failed")
		fmt.Printf("[!] %d items still in cache", cache.Size())
		os.Remove(name)
		os.Remove(name2)
		cache.Stop()
		t.FailNow()
	}
	fmt.Println("ok")
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
	cache := NewDefaultCache()
	cache.MaxItems = 5
	cache.Start()
	names := make([]string, 0)
	fmt.Printf("[+] validating item limit on cache: ")

	for i := 0; i < 1000; i++ {
		tmpf, err := ioutil.TempFile("", "fctest")
		if err != nil {
			fmt.Println("failed")
			fmt.Println("[!] couldn't create temporary file: ", err.Error())
			destroyNames(names)
			t.FailNow()
		}
		name := tmpf.Name()
		names = append(names, name)
		tmpf.Close()

		err = ioutil.WriteFile(name, []byte("lorem ipsum dolor sit amet."), 0600)
		if err != nil {
			fmt.Println("failed")
			fmt.Println("[!] couldn't write temporary file: ", err.Error())
			destroyNames(names)
			cache.Stop()
			t.FailNow()
		}
		cache.CacheNow(name)
	}

	if cache.Size() > cache.MaxItems {
		fmt.Println("failed")
		fmt.Printf("[!] %d items in cache (limit should be %d)",
			cache.Size(), cache.MaxItems)
		cache.Stop()
		destroyNames(names)
		t.FailNow()
	}
	fmt.Println("ok")
	cache.Stop()
	destroyNames(names)
}

func TestNeverExpire(t *testing.T) {
	cache := NewDefaultCache()
	cache.ExpireItem = 0
	cache.Start()

	fmt.Printf("[+] validating no time limit expirations: ")
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
	time.Sleep(5 * time.Second)
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
