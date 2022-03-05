package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/jacobsa/fuse"

	"github.com/protosam/go-fuse-loopback"
)

//var fType = flag.String("type", "", "The name of the samples/ sub-dir.")
//var fReadyFile = flag.Uint64("ready_file", 0, "FD to signal when ready.")
var fMountPoint = flag.String("mount-point", "", "Path to mount point.")
var fSource = flag.String("source", "", "Path to mirror.")

var fReadOnly = flag.Bool("read-only", false, "Mount in read only mode.")
var fDebug = flag.Bool("debug", false, "Enable debug logging.")

func main() {
	flag.Parse()

	if *fMountPoint == "" {
		log.Fatalf("You must set --mount-point.")
	}

	cfg := &fuse.MountConfig{
		ReadOnly: *fReadOnly,
	}

	if *fDebug {
		cfg.DebugLogger = log.New(os.Stderr, "fuse: ", 0)
	}

	server, err := loopbackfs.NewFileSystemServer(*fSource)
	if err != nil {
		log.Fatal(err)
	}

	mfs, err := fuse.Mount(*fMountPoint, server, cfg)
	if err != nil {
		log.Fatalf("Mount: %+v\n", err)
	}

	// Wait for it to be unmounted.
	if err = mfs.Join(context.Background()); err != nil {
		log.Fatalf("Join: %+v\n", err)
	}
}
