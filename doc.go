/*
   Copyright (c) 2012 Kyle Isom <kyle@tyrfingr.is>

   Permission to use, copy, modify, and distribute this software for any
   purpose with or without fee is hereby granted, provided that the
   above copyright notice and this permission notice appear in all
   copies.

   THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL
   WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED
   WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE
   AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL
   DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA
   OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER
   TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
   PERFORMANCE OF THIS SOFTWARE.
*/

/*
   package filecache implements a simple file cache.

   A file cache can be created with either the NewDefaultCache() function to
   get a cache with the defaults set, or NewCache() to get a new cache with
   0 values for everything; you will not be able to store items in this cache
   until the values are changed; specifically, at a minimum, you should change
   the MaxItems field to be greater than zero.

   Let's start with a basic example:

     cache := filecache.NewDefaultCache()
     cache.Start()

     readme, err := cache.ReadFile("README.md")
     if err != nil {
        fmt.Println("[!] couldn't read the README:", err.Error())
     } else {
        fmt.Printf("[+] read %d bytes\n", len(readme))
     }

  You can transparently read and cache a file using RetrieveFile (and
  RetrieveFileString); if the file is not in the cache, it will be read
  from the file system and returned - the cache will start a background
  thread to cache the file. Similarly, the WriterFile method will write
  the file to the specified io.Writer. For example, you could create a
  FileServer function along the lines of

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

  When cache.Start() is called, a goroutine is launched in the background
  that routinely checks the cache for expired items. The delay between
  runs is specified as the number of seconds given by cache.Every ("every
  cache.Every seconds, check for expired items"). There are three criteria
  used to determine whether an item in the cache should be expired; they are:

     1. Has the file been modified on disk? (The cache stores the last time
        of modification at the time of caching, and compares that to the
        file's current last modification time).
     2. Has the file been in the cache for longer than the maximum allowed
        time? This check can be disabled by setting the cache's ExpireItem
        field to 0; in this case, the cache will only expire items that have
        been modified since caching or that satisfy the next condition.
     3. Is the cache at capacity? When a file is being cached, a check is
        made to see if the cache is currently filled. If it is, the item that
        was last accessed the longest ago is expired and the new item takes
        its place. When loading items asynchronously, this check might miss
        the fact that the cache will be at capacity; the background scanner
        performs a check after its regular checks to ensure that the cache is
        not at capacity.

  The background scanner can be disabled by setting cache.Every to 0; if so,
  cache expiration is only done when the cache is at capacity.

  Once the cache is no longer needed, a call to cache.Stop() will close down
  the channels and signal the background scanner that it should stop.

*/
package filecache
