package blobstore

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	"github.com/c0-ops/goblob/nfs"
	"github.com/c0-ops/goblob/tar"
)

type BlobstoreFactory interface {
	NewBlobstore(
		username string,
		password string,
		ip string,
		extractor tar.CmdExtractor,
		logger boshlog.Logger,
	) (Blobstore, error)
}

type blobstoreFactory struct {
	fs     boshsys.FileSystem
	logger boshlog.Logger
}

func NewRemoteBlobstoreFactory(fs boshsys.FileSystem, logger boshlog.Logger) BlobstoreFactory {
	return blobstoreFactory{
		fs:     fs,
		logger: logger,
	}
}

func (f blobstoreFactory) NewBlobstore(
	username string,
	password string,
	ip string,
	extractor tar.CmdExtractor,
	logger boshlog.Logger,
) (Blobstore, error) {
	nfsClient, err := nfs.NewNFSClient(username, password, ip, extractor, f.fs, logger)
	if err != nil {
		return nil, err
	}
	return NewBlobstore(nfsClient, f.fs, extractor, f.logger), nil
}