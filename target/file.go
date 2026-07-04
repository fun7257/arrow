package target

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fun7257/arrow"
)

// WriteFile serves a file from disk.
func WriteFile(c *arrow.Context, path string) error {
	return Write(c, fileTarget{path: path})
}

// WriteAttachment serves a file as a download.
func WriteAttachment(c *arrow.Context, path, filename string) error {
	if filename == "" {
		filename = filepath.Base(path)
	}
	return Write(c, fileTarget{
		path:        path,
		disposition: fmt.Sprintf("attachment; filename=%q", filename),
	})
}

// WriteFileFS serves a file from fs.FS.
// The entire file is read into memory before serving.
func WriteFileFS(c *arrow.Context, fsys fs.FS, path string) error {
	return Write(c, fileFSTarget{fsys: fsys, path: path})
}

// WriteAttachmentFS serves a file from fs.FS as a download.
// The entire file is read into memory before serving.
func WriteAttachmentFS(c *arrow.Context, fsys fs.FS, path, filename string) error {
	if filename == "" {
		filename = filepath.Base(path)
	}
	return Write(c, fileFSTarget{
		fsys:        fsys,
		path:        path,
		disposition: fmt.Sprintf("attachment; filename=%q", filename),
	})
}

type fileTarget struct {
	path        string
	disposition string
}

func (f fileTarget) StatusCode() int {
	return http.StatusOK
}

func (f fileTarget) Respond(c *arrow.Context) error {
	if c.Written() {
		return nil
	}
	file, err := os.Open(f.path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("target: cannot serve directory %q", f.path)
	}
	if f.disposition != "" {
		c.Writer.Header().Set("Content-Disposition", f.disposition)
	}
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
	return nil
}

type fileFSTarget struct {
	fsys        fs.FS
	path        string
	disposition string
}

func (f fileFSTarget) StatusCode() int {
	return http.StatusOK
}

func (f fileFSTarget) Respond(c *arrow.Context) error {
	if c.Written() {
		return nil
	}
	data, err := fs.ReadFile(f.fsys, f.path)
	if err != nil {
		return err
	}
	info, err := fs.Stat(f.fsys, f.path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("target: cannot serve directory %q", f.path)
	}
	if f.disposition != "" {
		c.Writer.Header().Set("Content-Disposition", f.disposition)
	}
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), bytes.NewReader(data))
	return nil
}