package sync

import (
	"hash"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
)

const (
	// scannerCopyBufferSize specifies the size of the internal buffer that a
	// scanner uses to copy file data.
	// TODO: Figure out if we should set this on a per-machine basis. This value
	// is taken from Go's io.Copy method, which defaults to allocating a 32k
	// buffer if none is provided.
	scannerCopyBufferSize = 32 * 1024

	// defaultInitialCacheCapacity specifies the default capacity for new caches
	// when the existing cache is nil or empty. It is designed to save several
	// rounds of cache capacity doubling on insert without always allocating a
	// huge cache. Its value is somewhat arbitrary.
	defaultInitialCacheCapacity = 1024
)

type scanner struct {
	root        string
	hasher      hash.Hash
	cache       *Cache
	pathIgnorer *pathIgnorer
	ignoreSize  uint64
	newCache    *Cache
	buffer      []byte
}

func (s *scanner) file(path string, info os.FileInfo) (*Entry, error) {
	// Extract metadata.
	mode := info.Mode()
	modificationTime := info.ModTime()
	size := uint64(info.Size())

	// Compute executability.
	executable := (mode&AnyExecutablePermission != 0)

	// Try to find a cached digest. We only enforce that type, modification
	// time, and size haven't changed in order to re-use digests.
	var digest []byte
	cached, hit := s.cache.Entries[path]
	match := hit &&
		(os.FileMode(cached.Mode)&os.ModeType) == (mode&os.ModeType) &&
		modificationTime.Equal(cached.ModificationTime) &&
		cached.Size_ == size
	if match {
		digest = cached.Digest
	}

	// If we weren't able to pull a digest from the cache, compute one manually.
	if digest == nil {
		// Open the file and ensure its closure.
		file, err := os.Open(filepath.Join(s.root, path))
		if err != nil {
			return nil, errors.Wrap(err, "unable to open file")
		}
		defer file.Close()

		// Reset the hash state.
		s.hasher.Reset()

		// Copy data into the hash and very that we copied as much as expected.
		if copied, err := io.CopyBuffer(s.hasher, file, s.buffer); err != nil {
			return nil, errors.Wrap(err, "unable to hash file contents")
		} else if uint64(copied) != size {
			return nil, errors.New("hashed size mismatch")
		}

		// Compute the digest.
		digest = s.hasher.Sum(nil)
	}

	// Add a cache entry.
	s.newCache.Entries[path] = &CacheEntry{
		Mode:             uint32(mode),
		ModificationTime: modificationTime,
		Size_:            size,
		Digest:           digest,
	}

	// Success.
	return &Entry{
		Kind:       EntryKind_File,
		Executable: executable,
		Digest:     digest,
	}, nil
}

func (s *scanner) symlink(path string) (*Entry, error) {
	// Read the link target and ensure it's sane.
	target, err := os.Readlink(filepath.Join(s.root, path))
	if err != nil {
		return nil, errors.Wrap(err, "unable to read symlink target")
	} else if target == "" {
		return nil, errors.New("empty symlink target read")
	}

	// Success.
	return &Entry{
		Kind:   EntryKind_Symlink,
		Target: target,
	}, nil
}

func (s *scanner) directory(path string) (*Entry, error) {
	// Read directory contents.
	directoryContents, err := filesystem.DirectoryContents(filepath.Join(s.root, path))
	if err != nil {
		return nil, errors.Wrap(err, "unable to read directory contents")
	}

	// Compute entries.
	contents := make(map[string]*Entry, len(directoryContents))
	for _, name := range directoryContents {
		// Compute the content path.
		contentPath := pathpkg.Join(path, name)

		// If this path is ignored, then skip it.
		if s.pathIgnorer.ignored(contentPath) {
			continue
		}

		// Grab stat information for this path. If the path has disappeared
		// between list time and stat time, just ignore it.
		info, err := os.Lstat(filepath.Join(s.root, contentPath))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, errors.Wrap(err, "unable to stat directory content")
		}

		// Compute the kind for this content, skipping if unsupported.
		kind := EntryKind_File
		if mode := info.Mode(); mode&os.ModeDir != 0 {
			kind = EntryKind_Directory
		} else if mode&os.ModeSymlink != 0 {
			kind = EntryKind_Symlink
		} else if mode&os.ModeType != 0 {
			continue
		}

		// If this entry is a file and ignored due to its size, then skip it.
		if s.ignoreSize > 0 {
			if kind == EntryKind_File && uint64(info.Size()) >= s.ignoreSize {
				continue
			}
		}

		// Handle based on kind.
		var entry *Entry
		if kind == EntryKind_File {
			entry, err = s.file(contentPath, info)
		} else if kind == EntryKind_Symlink {
			if symlinksSupported {
				entry, err = s.symlink(contentPath)
			} else {
				continue
			}
		} else if kind == EntryKind_Directory {
			entry, err = s.directory(contentPath)
		} else {
			panic("unhandled entry kind")
		}

		// Watch for errors and add the entry.
		if err != nil {
			return nil, err
		}

		// Add the content.
		contents[name] = entry
	}

	// Success.
	return &Entry{
		Kind:     EntryKind_Directory,
		Contents: contents,
	}, nil
}

func Scan(root string, hasher hash.Hash, cache *Cache, ignores []string, ignoreSize uint64) (*Entry, *Cache, error) {
	// If the cache is nil, create an empty one.
	if cache == nil {
		cache = &Cache{}
	}

	// Create the ignorer.
	pathIgnorer, err := newPathIgnorer(ignores)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create ignorer")
	}

	// Create a new cache to populate. Estimate its capacity based on the
	// existing cache length. If the existing cache is empty, create one with
	// the default capacity.
	initialCacheCapacity := defaultInitialCacheCapacity
	if cacheLength := len(cache.GetEntries()); cacheLength != 0 {
		initialCacheCapacity = cacheLength
	}
	newCache := &Cache{make(map[string]*CacheEntry, initialCacheCapacity)}

	// Create a scanner.
	s := &scanner{
		root:        root,
		hasher:      hasher,
		cache:       cache,
		pathIgnorer: pathIgnorer,
		ignoreSize:  ignoreSize,
		newCache:    newCache,
		buffer:      make([]byte, scannerCopyBufferSize),
	}

	// Create the snapshot. We use os.Stat, as opposed to os.Lstat, because we
	// DO want to follow symbolic links at the root.
	if info, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, newCache, nil
		} else {
			return nil, nil, errors.Wrap(err, "unable to probe snapshot root")
		}
	} else if mode := info.Mode(); mode&os.ModeDir != 0 {
		if rootEntry, err := s.directory(""); err != nil {
			return nil, nil, err
		} else {
			return rootEntry, newCache, nil
		}
	} else if mode&os.ModeType != 0 {
		return nil, nil, errors.New("invalid snapshot root type")
	} else {
		if rootEntry, err := s.file("", info); err != nil {
			return nil, nil, err
		} else {
			return rootEntry, newCache, nil
		}
	}
}
