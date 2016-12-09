package nfs

import (
	"fmt"
	"io"
	"path"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	"github.com/c0-ops/goblob/cmd"
	"github.com/c0-ops/goblob/tar"
)

type Client interface {
	Get(blobPath string, blobID string) (content io.Reader, err error)
	Put(blobPath string, content io.ReadCloser, contentLength int64) (err error)
	GetAll(blobPath string) (extractPath string, err error)
}

type nfsClient struct {
	NfsDirectory string
	fs           boshsys.FileSystem
	extractor    tar.CmdExtractor
	Caller       cmd.Executor
	logger       boshlog.Logger
	logTag       string
}

var SshCmdExecutor = cmd.NewRemoteExecutor

func NewNFSClient(username string, password string, ip string, extractor tar.CmdExtractor, fs boshsys.FileSystem, logger boshlog.Logger) (*nfsClient, error) {
	config := cmd.SshConfig{
		Username: username,
		Password: password,
		Host:     ip,
		Port:     22,
	}
	remoteExecuter, err := SshCmdExecutor(config)
	if err != nil {
		return nil, err
	}
	return &nfsClient{
		fs:           fs,
		Caller:       remoteExecuter,
		extractor:    extractor,
		logger:       logger,
		logTag:       "nfsClient",
	}, nil
}

func (c *nfsClient) Get(blobPath string, blobID string) (io.Reader, error) {
	cmd := fmt.Sprintf("cd %s && cat %s", blobPath, blobID)
	c.logger.Debug(c.logTag, "Fetching blob %s with command: %s", blobID, cmd)
	return c.Caller.ExecuteForRead(cmd)
}

func (c *nfsClient) GetAll(blobPath string) (string, error) {
	src := path.Join("/tmp", blobPath) + ".tgz"
	cmd := fmt.Sprintf("cd %s && tar czf %s %s", "/var/vcap/store/shared", src, blobPath)
	c.logger.Debug(c.logTag, "Compressing blobs with command: %s\n", cmd)
	_, err := c.Caller.ExecuteForRead(cmd)
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Compressing blobs with command %s", cmd)
	}

	tmpFile, err := c.fs.TempFile("bosh-local-blob")
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Opening file for blobs")
	}
	defer tmpFile.Close()
	tmpDir := tmpFile.Name()

	c.logger.Debug(c.logTag, "Downloading tarball of blobs to %s\n", tmpDir)
	err = c.Caller.SecureCopy(src, tmpFile)
	if err != nil {
		return "", bosherr.WrapError(err, "Failed to download blobs")
	}
	extractPath, err := c.extractor.Extract(tmpDir)
	defer c.cleanup(tmpDir)
	if err != nil {
		return extractPath, bosherr.WrapError(err, "Extracting blobs")
	}
	return extractPath, err
}

func (c *nfsClient) Put(path string, content io.ReadCloser, contentLength int64) (err error) {
	return bosherr.WrapErrorf(err, "Put function not yet implemented")
}

func (c *nfsClient) cleanup(path string) {
	err := c.extractor.CleanUp(path)
	if err != nil {
		c.logger.Debug(c.logTag, "Failed to remove blobstore tarball %s", err.Error())
	}
}