package loopbackfs

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

func NewFileSystemServer(srcPath string) (fuse.Server, error) {
	// initialize inode sync.Map
	inodes := &sync.Map{}

	// create root inode entry
	root := &inodeEntry{
		id:   fuseops.RootInodeID,
		path: srcPath,
	}

	// add root inode entry to sync.Map
	inodes.Store(root.Id(), root)

	// initialize file system
	fs := &FileSystemServer{
		srcPath: srcPath,
		inodes:  inodes,
	}
	return fuseutil.NewFileSystemServer(fs), nil
}

type FileSystemServer struct {
	fuseutil.NotImplementedFileSystem

	inodes *sync.Map

	srcPath string
}

// Read Functions

// TODO: IMPLEMENT
// Return statistics about the file system's capacity and available resources.
func (f *FileSystemServer) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	return nil
}

func (f *FileSystemServer) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	entry, err := getOrCreateInode(f.inodes, op.Parent, op.Name)
	if err != nil {
		log.Printf("fs.LookUpInode for '%v' on '%v': %v", entry, op.Name, err)
		return fuse.EIO
	}

	// file not found
	if entry == nil {
		return fuse.ENOENT
	}

	outputEntry := &op.Entry
	outputEntry.Child = entry.Id()

	attributes, err := entry.Attributes()
	if err != nil {
		log.Printf("fs.LookUpInode.Attributes for '%v' on '%v': %v", entry, op.Name, err)
		return fuse.EIO
	}

	outputEntry.Attributes = *attributes

	return nil
}

// This is the first thing done on mount...
func (f *FileSystemServer) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	var entry, found = f.inodes.Load(op.Inode)
	if !found {
		return fuse.ENOENT
	}

	attributes, err := entry.(Inode).Attributes()
	if err != nil {
		log.Printf("fs.GetInodeAttributes for '%v': %v", entry, err)
		return fuse.EIO
	}

	op.Attributes = *attributes

	return nil
}

func (f *FileSystemServer) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	// Allow opening any directory...
	// TODO: check permissions?
	return nil
}

func (f *FileSystemServer) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	var entry, found = f.inodes.Load(op.Inode)
	if !found {
		return fuse.ENOENT
	}
	children, err := entry.(Inode).ListChildren(f.inodes)
	if err != nil {
		log.Printf("fs.ReadDir for '%v': %v", entry, err)
		return fuse.EIO
	}

	if op.Offset > fuseops.DirOffset(len(children)) {
		return fuse.EIO
	}

	children = children[op.Offset:]

	for _, child := range children {
		bytesWritten := fuseutil.WriteDirent(op.Dst[op.BytesRead:], *child)
		if bytesWritten == 0 {
			break
		}
		op.BytesRead += bytesWritten
	}
	return nil
}

func (f *FileSystemServer) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	// Allow opening any file.
	// TODOL check permissions?
	return nil
}

func (f *FileSystemServer) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	var entry, found = f.inodes.Load(op.Inode)
	if !found {
		return fuse.ENOENT
	}
	contents, err := entry.(Inode).Contents()
	if err != nil {
		log.Printf("fs.ReadFile for '%v': %v", entry, err)
		return fuse.EIO
	}

	if op.Offset > int64(len(contents)) {
		return fuse.EIO
	}

	contents = contents[op.Offset:]
	op.BytesRead = copy(op.Dst, contents)
	return nil
}

func (f *FileSystemServer) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) error {
	// TODO: Implement
	return nil // fuse.ENOSYS
}

func (f *FileSystemServer) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	// TODO: Implement
	return nil // fuse.ENOSYS
}

func (f *FileSystemServer) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	// TODO: Implement
	return nil // fuse.ENOSYS
}

func (f *FileSystemServer) ForgetInode(ctx context.Context, op *fuseops.ForgetInodeOp) error {
	// TODO: Implement
	return nil // fuse.ENOSYS
}

func (f *FileSystemServer) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) error {
	// TODO: Implement
	return nil // fuse.ENOSYS
}

func (f *FileSystemServer) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	return fuse.ENOSYS
}

func (f *FileSystemServer) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) error {
	// TODO: Implement
	return nil // fuse.ENOSYS
}

// Create Functions

func (f *FileSystemServer) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	return fuse.ENOSYS
}

func (f *FileSystemServer) MkNode(ctx context.Context, op *fuseops.MkNodeOp) error {
	parentInode, found := getInodeById(f.inodes, op.Parent)
	if !found {
		return fuse.ENOENT
	}

	newEntryPath := filepath.Join(parentInode.Path(), op.Name)

	mode := syscallMode(op.Mode)

	// I don't think the following two conditions are valid in mknod, so they
	// are no in the following switch case
	//    case mode&os.ModeSymlink == os.ModeSymlink:
	//    case mode&os.ModeDir == os.ModeDir:
	//
	// switch handles NamedPipes, Sockets, BlockDevs, and CharDevs.
	switch {
	// Make a named pipe
	case op.Mode&os.ModeNamedPipe == os.ModeNamedPipe:
		log.Printf("making %s\n", newEntryPath)
		log.Printf("with mode %+v\n", op.Mode)
		log.Printf("with mode %+v\n", op.Mode|syscall.S_IFIFO)

		//mode |= syscall.S_IFIFO
		if err := syscall.Mkfifo(newEntryPath, mode); err != nil {
			log.Printf("%s\n", err)
		}

	// Make a socket
	case op.Mode&os.ModeSocket == os.ModeSocket:
		// TODO: implement

	// make devices
	case op.Mode&os.ModeDevice == os.ModeDevice:
		// Make device here
		dev, err := syscallMakeDev(parentInode.Path())
		if err != nil {
			return fuse.EIO
		}

		// handle devices
		switch {
		// make a chardev
		case op.Mode&os.ModeCharDevice == os.ModeCharDevice:
			mode |= syscall.S_IFCHR

		// make a block device
		default:
			mode |= syscall.S_IFBLK
		}

		// run mknod
		if err := syscall.Mknod(newEntryPath, mode, dev); err != nil {
			log.Printf("%s\n", err)
		}

	// no cases were matched, return an error
	default:
		return fuse.ENOSYS
	}

	// matched cases fall through to here
	return nil
}

func (f *FileSystemServer) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) error {
	return fuse.ENOSYS
}

func (f *FileSystemServer) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) error {
	return fuse.ENOSYS
}

func (f *FileSystemServer) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) error {
	return fuse.ENOSYS
}

// Update Functions

func (f *FileSystemServer) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) error {
	return fuse.ENOSYS
}

func (f *FileSystemServer) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	// TODO: implement RENAME_EXCHANGE situation for pivot root? Linux only?

	oldParent, found := getInodeById(f.inodes, op.OldParent)
	if !found {
		return fuse.ENOENT
	}

	newParent, found := getInodeById(f.inodes, op.NewParent)
	if !found {
		return fuse.ENOENT
	}

	p1 := filepath.Join(oldParent.Path(), op.OldName)
	p2 := filepath.Join(newParent.Path(), op.NewName)

	return syscall.Rename(p1, p2)
}

func (f *FileSystemServer) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) error {
	return fuse.ENOSYS
}

func (f *FileSystemServer) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) error {
	return fuse.ENOSYS
}

func (f *FileSystemServer) SyncFile(ctx context.Context, op *fuseops.SyncFileOp) error {
	return fuse.ENOSYS
}

// Delete Functions

func (f *FileSystemServer) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	return fuse.ENOSYS
}

func (f *FileSystemServer) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	entry, err := getInodeByName(f.inodes, op.Parent, op.Name)
	if err != nil {
		return err
	}

	if err := syscall.Unlink(entry.Path()); err != nil {
		return err
	}

	f.inodes.Delete(entry.Id())

	return nil
}

func (f *FileSystemServer) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	return fuse.ENOSYS
}

// Fallocate is a very linux specific function ref: https://pkg.go.dev/syscall#Fallocate
func (f *FileSystemServer) Fallocate(ctx context.Context, op *fuseops.FallocateOp) error {
	return fuse.ENOSYS
}

// Regard all inodes (including the root inode) as having their lookup counts
// decremented to zero, and clean up any resources associated with the file
// system. No further calls to the file system will be made.
func (f *FileSystemServer) Destroy() {
	log.Printf("Running Destroy\n")
}
