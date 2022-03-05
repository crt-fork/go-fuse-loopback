package loopbackfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

// Tracking for current node id
var allocatedInodeId uint64 = fuseops.RootInodeID

// Atomically gets the next node id
func nextInodeID() (next fuseops.InodeID) {
	nextInodeId := atomic.AddUint64(&allocatedInodeId, 1)
	return fuseops.InodeID(nextInodeId)
}

func getInodeById(inodes *sync.Map, parentId fuseops.InodeID) (Inode, bool) {
	entry, found := inodes.Load(parentId)
	// TODO: check permissions?
	return entry.(Inode), found
}

func getInodeByName(inodes *sync.Map, parentId fuseops.InodeID, name string) (Inode, error) {
	parent, found := getInodeById(inodes, parentId)
	if !found {
		return nil, fuse.ENOENT
	}

	entries, err := ioutil.ReadDir(parent.Path())
	if err != nil {
		return nil, err
	}

	// TODO: check permissions?
	for _, entry := range entries {
		if entry.Name() == name {
			inodeEntry := &inodeEntry{
				id:   nextInodeID(),
				path: filepath.Join(parent.Path(), name),
			}
			storedEntry, _ := inodes.LoadOrStore(inodeEntry.id, inodeEntry)
			return storedEntry.(Inode), nil
		}
	}

	return nil, fuse.ENOENT
}

// TODO: document
func getOrCreateInode(inodes *sync.Map, parentId fuseops.InodeID, name string) (Inode, error) {
	entry, err := getInodeByName(inodes, parentId, name)
	if err != nil && err != fuse.ENOENT {
		return nil, err
	}

	return entry, nil
}

// Create a new inode of specified path
func NewInode(path string) (Inode, error) {
	// create a new inode entry
	inodeEntry := &inodeEntry{
		id:   nextInodeID(),
		path: path,
	}

	// return the entry
	return inodeEntry, nil
}

// Interfacting for inodes on the file system
type Inode interface {
	// Id of inode
	Id() fuseops.InodeID

	// File system path of Inode
	Path() string

	// Stringified representation of inode
	String() string

	// Attributes of the inode such as permissions and ownership
	Attributes() (*fuseops.InodeAttributes, error)

	// Provides list of children inodes
	ListChildren(inodes *sync.Map) ([]*fuseutil.Dirent, error)

	// Returns node contents in bytes
	Contents() ([]byte, error)
}

// Implements the Inode interface.
type inodeEntry struct {
	// id of the inode
	id fuseops.InodeID

	// file system path to the node
	path string
}

// See type Inode iterface{} documentation for more info
func (node *inodeEntry) Id() fuseops.InodeID {
	return node.id
}

// See type Inode iterface{} documentation for more info
func (node *inodeEntry) Path() string {
	return node.path
}

// See type Inode iterface{} documentation for more info
func (node *inodeEntry) String() string {
	return fmt.Sprintf("%v::%v", node.id, node.path)
}

// Attributes of the inode such as permissions and ownership
// This method makes an attempt to be OS agnostic
// It is meant to run on MacOS, Linux, and Windows Subsystem for Linux.
//
// TODO: Make it work virtually on Windows itself.
func (node *inodeEntry) Attributes() (*fuseops.InodeAttributes, error) {
	nodeInfo, err := os.Stat(node.path)

	// handle error
	if err != nil {
		return &fuseops.InodeAttributes{}, err
	}

	// create generic node attributes for the requested node
	nodeAttributes := &fuseops.InodeAttributes{
		Size:   uint64(nodeInfo.Size()),
		Nlink:  1,
		Mode:   nodeInfo.Mode(),
		Atime:  nodeInfo.ModTime(), // Time of last access
		Mtime:  nodeInfo.ModTime(), // Time of last modification
		Ctime:  time.Now(),         // Time of last modification to inode
		Crtime: time.Now(),         // Time of creation (OS X only)
		Uid:    0,
		Gid:    0,
	}

	// add *nix stats to the node, more info about *nix specific attributes can
	// be found here: https://man7.org/linux/man-pages/man2/lstat.2.html
	// look for "The stat structure" on the page.
	if nixInfo, ok := nodeInfo.Sys().(*syscall.Stat_t); ok {
		nodeAttributes.Nlink = uint32(nixInfo.Nlink)
		nodeAttributes.Ctime = time.Unix(nixInfo.Ctimespec.Sec, nixInfo.Ctimespec.Nsec)
		nodeAttributes.Uid = nixInfo.Uid
		nodeAttributes.Gid = nixInfo.Gid
	}

	return nodeAttributes, nil
}

func (node *inodeEntry) ListChildren(inodes *sync.Map) ([]*fuseutil.Dirent, error) {
	children, err := ioutil.ReadDir(node.path)
	if err != nil {
		return nil, err
	}
	dirents := make([]*fuseutil.Dirent, len(children))
	for i, child := range children {

		childInode, err := getOrCreateInode(inodes, node.id, child.Name())
		if err != nil || childInode == nil {
			return nil, nil
		}

		var childType fuseutil.DirentType
		if child.IsDir() {
			childType = fuseutil.DT_Directory
		} else if child.Mode()&os.ModeSymlink != 0 {
			childType = fuseutil.DT_Link
		} else {
			childType = fuseutil.DT_File
		}

		dirents[i] = &fuseutil.Dirent{
			Offset: fuseops.DirOffset(i + 1),
			Inode:  childInode.Id(),
			Name:   child.Name(),
			Type:   childType,
		}
	}
	return dirents, nil
}

// See type Inode iterface{} documentation for more info
// TODO: Make this better maybe?
func (node *inodeEntry) Contents() ([]byte, error) {
	return ioutil.ReadFile(node.path)
}
