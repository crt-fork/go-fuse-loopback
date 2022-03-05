# Go Fuse Loopback Project
The goal is to get a lookbackfs together that works on both Linux and Mac (and possibly other *nix) as well.

After getting this done, it would be nice to have it consolidated back into `github.com/jacobsa/fuse` and this repo deleted.

# Motivation
After having looked at multiple Go based Fuse projects, I liked jacobsa/fuse the most. It targeted both Mac and Linux, and it can also work in Windows Subsystem for Linux. 

The problem I've encountered is that there just isn't a good enough example to springboard off of. We're completely missing the tutorial example, which is this. 

Success is when all the normal stuff you would expect works and is implemented in `loopbackfs.go`.

## Special Files
So `mknod` does special stuff. I've settled on the idea that it should have some stort of "like" behavoir. For example, if mkfifo can't work and be treated as the same fifo, it's fine as long as the path it's referenced at works as a standalone.

My reasoning for this is because no one is really going to use this to make a loopbackfs. We have bind mounts for that. What people will use this for is networked filesystems, and I think it's fair to emulate what standard NFS4 does.

# Status

This is my test setup.
```
$ mkdir -p mnt.testing/{source,mirror}
$ echo hello world > ./mnt.testing/source/hello.txt
```

## Things that work
Mounting works.
```
$ go run ./cmd/main.go --debug --mount-point ./mnt.testing/mirror --source ./mnt.testing/source
```

Listing works.
```
$ ls -lah ./mnt.testing/mirror
```

Reading the entire file works.
```
$ cat ./mnt.testing/mirror/hello.txt 
hello world
```

Delete files works, but not directories yet.
```
$ touch mnt.testing/source/badfile.txt
$ mkdir -p mnt.testing/source/delete/these/dirs

$ ls -lah mnt.testing/mirror/
total 24
drwxr-xr-x  5 protosam  protosam   160B Mar  5 17:46 .
drwxr-xr-x  4 protosam  protosam   128B Mar  5 17:42 ..
-rw-r--r--  1 protosam  protosam     0B Mar  5 17:46 badfile.txt
drwxr-xr-x  3 protosam  protosam    96B Mar  5 17:46 delete
-rw-r--r--  1 protosam  protosam    12B Mar  5 17:43 hello.txt

$ rm mnt.testing/mirror/badfile.txt

$ rm -rf mnt.testing/mirror/delete
rm: mnt.testing/mirror/delete/these/dirs: Function not implemented
rm: mnt.testing/mirror/delete/these: Function not implemented
rm: mnt.testing/mirror/delete: Function not implemented

$ ls -lah mnt.testing/mirror/
total 24
drwxr-xr-x  4 protosam  protosam   128B Mar  5 17:48 .
drwxr-xr-x  4 protosam  protosam   128B Mar  5 17:42 ..
drwxr-xr-x  3 protosam  protosam    96B Mar  5 17:48 delete
-rw-r--r--  1 protosam  protosam    12B Mar  5 17:43 hello.txt

```

## Currently Stuck On
mkfifo... it kinda works... and then doesn't.

```
$ mkfifo mnt.testing/mirror/hello2.pipe
mkfifo: mnt.testing/mirror/hello2.pipe: Invalid argument

$ ls -lah mnt.testing/mirror/hello2.pipe
prw-r--r--  1 protosam  protosam     0B Mar  5 17:52 mnt.testing/mirror/hello2.pipe

$ cat mnt.testing/mirror/hello2.pipe
cat: mnt.testing/mirror/hello2.pipe: Operation not permitted

$ echo hey pipe > mnt.testing/mirror/hello2.pipe
-bash: mnt.testing/mirror/hello2.pipe: Operation not permitted
```

## Todo
There's a lot.

creating unix sockets

figuring out how to test mknod for block devices and char devices

are there other mknod features?

mkdir ./mnt.testing/mirror/another-dir

rm -rf dirs....

chmod, chown, chattr

however you edit attributes

efficient file ops...

everything else?
