package main

import (
	"fmt"
	"net/url"
	"os"
	"path"

	"github.com/codegangsta/cli"
	"github.com/pborman/uuid"
	g8ufs "github.com/zero-os/0-fs"
	"github.com/zero-os/0-fs/meta"
	"github.com/zero-os/0-fs/storage"
)

var (
	BaseFSDir = path.Join(os.TempDir(), "zbundle")
)

func reportToRedis(ctx *cli.Context, err error) error {
	return nil
}

func action(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return fmt.Errorf("invalid number of arguments")
	}

	if !isRoot() {
		return fmt.Errorf("please run as root")
	}

	u, err := url.Parse(ctx.GlobalString("storage"))
	if err != nil {
		return fmt.Errorf("invalid storage url: %s", err)
	}

	flist := ctx.Args()[0]
	root := ctx.Args()[1]

	os.MkdirAll(root, 0755)
	// should we do this under temp?
	id := uuid.New()
	namespace := path.Join(BaseFSDir, id)

	metaPath, err := getMetaDB(namespace, flist)
	if err != nil {
		return err
	}

	metaStore, err := meta.NewRocksMeta("", metaPath)
	if err != nil {
		return err
	}

	stor, err := storage.NewARDBStorage(u)
	if err != nil {
		return err
	}

	opt := g8ufs.Options{
		Backend:   namespace,
		Target:    root,
		MetaStore: metaStore,
		Storage:   stor,
		Cache:     path.Join(BaseFSDir, "cache"),
	}

	fs, err := g8ufs.Mount(&opt)

	defer os.RemoveAll(namespace)
	defer os.RemoveAll(fmt.Sprintf("%s.db", namespace))
	defer fs.Unmount()

	err = sandbox(root, ctx.GlobalStringSlice("env"))

	if ctx.GlobalBool("no-exit") {
		if err != nil {
			log.Errorf("%v", err)
		}
		log.Infof("flist exited, waiting for unmount (--no-exit was set)")
		fs.Wait()
	}

	if err != nil {
		if err := reportToRedis(ctx, err); err != nil {
			log.Errorf("failed to report to redis: %s", err)
		}
	}

	return err
}
