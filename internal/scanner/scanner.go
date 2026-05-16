package scanner

import (
	"io/fs"
	"path/filepath"
	"strings"
	"syscall"
)

type Result struct {
	Dirs map[string]int64
}

type inodeKey struct{ dev, ino uint64 }

func Scan(root string, maxDepth int, skipPaths []string) (*Result, error) {
	root = filepath.Clean(root)
	result := &Result{Dirs: make(map[string]int64)}

	skipSet := make(map[string]bool, len(skipPaths))
	for _, p := range skipPaths {
		skipSet[filepath.Clean(p)] = true
	}

	// Track seen inodes to avoid double-counting hard-linked files.
	seenInodes := make(map[inodeKey]bool)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if skipSet[filepath.Clean(path)] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() || d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		fileSize, key, hasInode := sizeAndInode(info)
		if fileSize == 0 {
			return nil
		}
		if hasInode {
			if seenInodes[key] {
				return nil
			}
			seenInodes[key] = true
		}

		// Accumulate into all ancestor dirs within maxDepth.
		dir := filepath.Dir(path)
		for {
			depth := dirDepth(root, dir)
			if depth >= 0 && depth <= maxDepth {
				result.Dirs[dir] += fileSize
			}
			if dir == root {
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}

		return nil
	})

	return result, err
}

// sizeAndInode returns the allocated disk size, the inode key for deduplication,
// and whether inode info was available. Uses block counts (like du) so sparse
// files report actual on-disk usage rather than logical size.
func sizeAndInode(info fs.FileInfo) (size int64, key inodeKey, hasInode bool) {
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		size = sys.Blocks * 512
		if size == 0 {
			size = info.Size() // fallback for zero-block files
		}
		return size, inodeKey{uint64(sys.Dev), uint64(sys.Ino)}, true
	}
	return info.Size(), inodeKey{}, false
}

func dirDepth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}
