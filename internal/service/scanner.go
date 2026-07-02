package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"collections/internal/model"

	"github.com/cespare/xxhash/v2"
)

// scanDiff 扫描差异结果（package 内部使用）
type scanDiff struct {
	Added    []string // 新增文件的相对路径
	Removed  []string // 已删除文件的相对路径
	Modified []string // 已修改文件的相对路径（hash 变化）
}

// buildTree 构建文件夹的 Merkle Tree。
// cached 为上次的快照，非 nil 时启用 dir_mtime + mtime/size 双重剪枝。
func buildTree(rootPath string, cached *model.TreeNode) (*model.TreeNode, error) {
	return buildNode(rootPath, "", cached)
}

// buildNode 递归构建目录节点。
// relDir 为从根目录到当前目录的相对路径（空串表示根目录，用 "/" 分隔）。
func buildNode(rootPath, relDir string, cached *model.TreeNode) (*model.TreeNode, error) {
	dirPath := rootPath
	if relDir != "" {
		dirPath = filepath.Join(rootPath, filepath.FromSlash(relDir))
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("stat dir: %w", err)
	}
	dirMtime := info.ModTime()

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	node := &model.TreeNode{
		Type:     model.TreeNodeDir,
		Mtime:    &dirMtime,
		Children: make(map[string]*model.TreeNode),
	}

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			childRelDir := name
			if relDir != "" {
				childRelDir = relDir + "/" + name
			}
			var cachedChild *model.TreeNode
			if cached != nil {
				cachedChild = cached.Children[name]
			}
			childNode, err := buildNode(rootPath, childRelDir, cachedChild)
			if err != nil {
				continue
			}
			node.Children[name] = childNode
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		if !model.IsSupportedMediaExt(ext) {
			continue
		}

		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}

		relPath := name
		if relDir != "" {
			relPath = relDir + "/" + name
		}
		fileMtime := fileInfo.ModTime()
		fileSize := fileInfo.Size()

		if cached != nil {
			if cf, ok := cached.Children[name]; ok &&
				cf.Type == model.TreeNodeFile &&
				cf.Mtime != nil && fileMtime.Equal(*cf.Mtime) &&
				cf.Size != nil && *cf.Size == fileSize {
				node.Children[name] = cf
				continue
			}
		}

		hash := computeFileHash(relPath, fileMtime, fileSize)
		node.Children[name] = &model.TreeNode{
			Type:  model.TreeNodeFile,
			Hash:  hash,
			Mtime: &fileMtime,
			Size:  &fileSize,
		}
	}

	node.Hash = computeDirHash(node.Children)
	return node, nil
}

func computeFileHash(relPath string, mtime time.Time, size int64) string {
	input := relPath + "|" + strconv.FormatInt(mtime.UnixNano(), 10) + "|" + strconv.FormatInt(size, 10)
	return strconv.FormatUint(xxhash.Sum64String(input), 16)
}

func computeDirHash(children map[string]*model.TreeNode) string {
	if len(children) == 0 {
		return "0"
	}
	hashes := make([]string, 0, len(children))
	for _, child := range children {
		hashes = append(hashes, child.Hash)
	}
	sort.Strings(hashes)
	return strconv.FormatUint(xxhash.Sum64String(strings.Join(hashes, "|")), 16)
}

// diffTrees 对比新旧 Merkle Tree，返回文件级变更列表。
// oldRoot/newRoot 为 nil 时视为空树。
func diffTrees(oldRoot, newRoot *model.TreeNode) scanDiff {
	oldFiles := collectFiles(oldRoot, "")
	newFiles := collectFiles(newRoot, "")

	oldMap := make(map[string]string, len(oldFiles))
	for _, f := range oldFiles {
		oldMap[f.relPath] = f.hash
	}

	var diff scanDiff
	seen := make(map[string]struct{}, len(newFiles))
	for _, f := range newFiles {
		seen[f.relPath] = struct{}{}
		oldHash, exists := oldMap[f.relPath]
		if !exists {
			diff.Added = append(diff.Added, f.relPath)
		} else if oldHash != f.hash {
			diff.Modified = append(diff.Modified, f.relPath)
		}
	}

	for _, f := range oldFiles {
		if _, exists := seen[f.relPath]; !exists {
			diff.Removed = append(diff.Removed, f.relPath)
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	sort.Strings(diff.Modified)
	return diff
}

type fileEntry struct {
	relPath string
	hash    string
}

// collectFiles 递归收集树中所有文件节点的相对路径和 hash
func collectFiles(node *model.TreeNode, prefix string) []fileEntry {
	if node == nil || node.Children == nil {
		return nil
	}
	var files []fileEntry
	for name, child := range node.Children {
		childPath := name
		if prefix != "" {
			childPath = prefix + "/" + name
		}
		switch child.Type {
		case model.TreeNodeFile:
			files = append(files, fileEntry{relPath: childPath, hash: child.Hash})
		case model.TreeNodeDir:
			files = append(files, collectFiles(child, childPath)...)
		}
	}
	return files
}
