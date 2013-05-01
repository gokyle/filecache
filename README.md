# filecache.go
## a simple Go file cache

## Overview

A file cache can be created with either the `NewDefaultCache()` function to
get a cache with the defaults set, or `NewCache()` to get a new cache with
`0` values for everything; you will not be able to store items in this cache
until the values are changed; specifically, at a minimum, you should set
the `MaxItems` field to be > 0.

Let's start with a basic example; we'll create a basic cache and give it a
maximum item size of 128M:

```
cache := filecache.NewDefaultCache()
cache.MaxSize = 128 * filecache.Megabyte
cache.Start()
```

The `Kilobyte`, `Megabyte`, and `Gigabyte` constants are provided as a
convience when setting cache sizes.

You can transparently read and cache a file using `ReadFile`  (and
`ReadFileString`); if the file is not in the cache, it will be read
from the file system and returned; the cache will start a background
thread to cache the file. Similarly, the `WriterFile` method will
write the file to the specified `io.Writer`. For example, you could
create a `FileServer` function along the lines of

```
   func FileServer(w http.ResponseWriter, r *http.Request) {
           path := r.URL.Path
           if len(path) > 1 {
                   path = path[1:len(path)]
           } else {
                   path = "."
           }

           err := cache.WriteFile(w, path)
           if err == nil {
                   ServerError(w, r)
           } else if err == filecache.ItemIsDirectory {
                   DirServer(w, r)
           }
      }
```

When `cache.Start()` is called, a goroutine is launched in the background
that routinely checks the cache for expired items. The delay between
runs is specified as the number of seconds given by `cache.Every` ("every
`cache.Every` seconds, check for expired items"). There are three criteria
used to determine whether an item in the cache should be expired; they are:

   1. Has the file been modified on disk? (The cache stores the last time
      of modification at the time of caching, and compares that to the
      file's current last modification time).
   2. Has the file been in the cache for longer than the maximum allowed
      time?
   3. Is the cache at capacity? When a file is being cached, a check is
      made to see if the cache is currently filled. If it is, the item that
      was last accessed the longest ago is expired and the new item takes
      its place. When loading items asynchronously, this check might miss
      the fact that the cache will be at capacity; the background scanner
      performs a check after its regular checks to ensure that the cache is
      not at capacity.

The background scanner can be disabled by setting `cache.Every` to 0; if so,
cache expiration is only done when the cache is at capacity.

Once the cache is no longer needed, a call to `cache.Stop()` will close down
the channels and signal the background scanner that it should stop.


## Usage

### Initialisation and Startup

The public fields of the `FileCache` struct are:

```
    MaxItems   int   // Maximum number of files to cache
    MaxSize    int64 // Maximum file size to store
    ExpireItem int   // Seconds a file should be cached for
    Every      int   // Run an expiration check Every seconds
```

You can create a new file cache with one of two functions:

* `NewCache()`: creates a new bare repository that just has the underlying
cache structure initialised. The public fields are all set to `0`, which is
very likely not useful (at a minimum, a `MaxItems` of `0` means no items can
or will be stored in the cache).
* `NewDefaultCache()` returns a new file cache initialised to some basic
defaults. The defaults are:

```
	DefaultExpireItem int   = 300 // 5 minutes
	DefaultMaxSize    int64 =  4 * Megabyte
	DefaultMaxItems   int   = 32
	DefaultEvery      int   = 60 // 1 minute
```

These defaults are public variables, and you may change them to more useful
values to your program.

Once the cache has been initialised, it needs to be started using the
`Start()` method. This is important for initialising the goroutine responsible
for ensuring the cache remains updated, as well as setting up the asynchronous
caching goroutine. The `Active` method returns true if the cache is currently
running. `Start()` returns an `error` if an error occurs; if one is returned,
the cache should not be used.

### Cache Information

The `FileCache` struct has several methods to return information about the
cache:

* `Size()` returns the number of files that are currently in the cache.
* `FileSize()` returns the sum of the file sizes stored in the cache; each 
item on the cache takes up approximately 32 bytes on top of this as overhead.
* `StoredFiles()` returns a list of strings containing the names of the files
currently cached. These are not sorted in any way.
* `InCache(name string)` returns true if `name` is in the cache.

### Primary Methods
While the cache has several methods available, there are four main functions
you will likely use to interact with cache apart from initialisation and
shutdown. All three of them provide transparent access to files; if the file
is in the cache, it is read from the cache. Otherwise, the file is checked
to make sure it is not a directory or uncacheable file, returning an error
if this is the case. Finally, a goroutine is launched to cache the file in
the background while the file is read and its contents provided directly from
the filesystem.

* `ReadFile(name string) ([]byte, error)` is used to get the contents of
file as a byte slice.
* `ReadFileString(name string) (string, error)` is used to get the
contents of a file as a string.
* `WriteFile(w io.Writer, name) error` is used to write the contents of the
file to the `io.Writer` interface given.
* `HttpWriteFile(w http.ResponseWriter, r *http.Request)` will write the
contents of the file transparently over an HTTP connect. This should be
used when the writer is an HTTP connection and will handle the
appropriate HTTP headers.

If you are using the file cache in an HTTP server, you might find the
following function useful:

* `HttpHandler(*FileCache) func(w http.ResponseWriter, r *http.Request)`
returns a function that can then be used directly in `http.HandleFunc`
calls.

Most people can now skip to the *Shutting Down* section.

### Reading from the Cache

If you are certain a file has been cached, and you want to access it directly
from the cache, you can use these functions:

* `GetItem(name string) ([]byte, bool)` will retrieve a byte slice containing
the contents of the file and a boolean indiciating whether the file was in
the cache. If it is not in the cache, the byte slice will be empty and no
attempt is made to add the file to the cache.
* `GetItemString(name string) (string, bool)` is the same as `GetItem` except
that it returns a string in place of the byte slice.
* `WriteItem(w io.Writer, name string) (err error)` is the same as `WriteFile`
except that no attempt is made to add the file to cache if it is not present.

### Add to the Cache

You can cache files without reading them using the two caching functions:

* `Cache(name string)` will cache the file in the background. It returns
immediately and errors are not reported; you can determine if the item is
in the cache with the `InCache` method; note that as this is a background
cache, the file may not immediately be cached.
* `CacheNow(name string) error` will immediately cache the file and block
until it has been cached, or until an error is returned.

### Removing from the Cache

The `Remove(name string) (bool, error)` method will remove the file named
from the cache. If the file was not in the cache or could not be removed,
it returns false.

### Shutting Down

Once you are done with the cache, the `Stop` method takes care of all the
necessary cleanup.

## Examples

Take a look at [cachesrv](https://github.com/gokyle/cachesrv) for
an example of a caching fileserver.

## License

`filecache` is released under the ISC license:

```
Copyright (c) 2012 Kyle Isom <kyle@tyrfingr.is>

Permission to use, copy, modify, and distribute this software for any
purpose with or without fee is hereby granted, provided that the above 
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE. 
```
