// Code generated by go-imbed. DO NOT EDIT.

// Package {{.Pkg}} holds binary resources embedded into Go executable
package {{.Pkg}}

import (
	"os"
	"io"
	"path/filepath"
	"io/ioutil"
{{- if .Params.BuildHttpHelperAPI }}
	"strconv"
{{- end }}
{{- if or .Params.BuildHttpFsAPI .Params.BuildHttpHelperAPI }}
	"net/http"
{{- end }}
{{- if or .Params.BuildFsAPI }}
	"strings"
	"path"
{{- end }}
{{- if or .Params.BuildHttpHelperAPI .Params.BuildFsAPI .Params.CompressAssets }}
	"bytes"
{{- end }}
{{- if .Params.CompressAssets }}
	"compress/gzip"
{{- end }}
	"time"
)

func blob_bytes(uint32) []byte
func blob_string(uint32) string

// Asset represents binary resource stored within Go executable. Asset implements
// fmt.Stringer and io.WriterTo interfaces, decompressing binary data if necessary.
type Asset struct {
	name         string // File name
	size         int32  // File size (uncompressed)
	blob         []byte // Resource blob []byte
	str_blob     string // Resource blob as string
{{- if .Params.CompressAssets }}
	isCompressed bool   // true if resources was compressed with gzip
{{- end}}
	mime         string // MIME Type
	tag          string // Tag is essentially a Tag of resource content and can be used as a value for "Etag" HTTP header
}

// Name returns the base name of the asset
func (a *Asset) Name() string       { return a.name }
// MimeType returns MIME Type of the asset
func (a *Asset) MimeType() string   { return a.mime }
{{- if .Params.CompressAssets }}
// IsCompressed returns true of asset has been compressed
func (a *Asset) IsCompressed() bool { return a.isCompressed }
{{- end }}

// Size implements os.FileInfo and returns the size of the asset (uncompressed, if asset has been compressed)
func (a *Asset) Size() int64        { return int64(a.size) }
// Mode implements os.FileInfo and always returns 0444
func (a *Asset) Mode() os.FileMode  { return 0444 }
// ModTime implements os.FileInfo and returns the time stamp when this package has been produced (the same value for all the assets)
func (a *Asset) ModTime() time.Time { return stamp }
// IsDir implements os.FileInfo and returns false
func (a *Asset) IsDir() bool        { return false }
// Sys implements os.FileInfo and returns nil
func (a *Asset) Sys() interface{}   { return nil }

// WriteTo implements io.WriterTo interface and writes content of the asset to w
func (a *Asset) WriteTo(w io.Writer) (int64, error) {
{{- if .Params.CompressAssets }}
	if a.isCompressed {
		ungzip, _ := gzip.NewReader(bytes.NewReader(a.blob))
		defer ungzip.Close()
		return io.Copy(w, ungzip)
	}
{{- end }}
	n, err := w.Write(a.blob)
	return int64(n), err
}

// The CopyTo method copies asset content to the target directory.
// If file with the same name, size and modification time exists,
// it will not be overwritten, unless overwrite = true is specified.
func (a *Asset) CopyTo(target string, mode os.FileMode, overwrite bool) error {
	fname := filepath.Join(target, a.name)
	fs, err := os.Stat(fname)
	if err == nil {
		if fs.IsDir() {
			return os.ErrExist
		} else if !overwrite && fs.Size() == a.Size() && fs.ModTime().Equal(a.ModTime()) {
			return nil
		}
	}
	file, err := ioutil.TempFile(target, ".imbed")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, err = a.WriteTo(file)
	if err != nil {
		return err
	}
	file.Close()
	os.Chtimes(file.Name(), a.ModTime(), a.ModTime())
	os.Chmod(file.Name(), mode)
	return os.Rename(file.Name(), fname)
}

// String returns (uncompressed) content of asset as a string
func (a *Asset) String() string {
{{- if .Params.CompressAssets }}
	if a.isCompressed {
		ungzip, _ := gzip.NewReader(bytes.NewReader(a.blob))
		ret, _ := ioutil.ReadAll(ungzip)
		ungzip.Close()
		return string(ret)
	}
{{- end }}
	return a.str_blob
}

{{- if .Params.BuildRawAccessAPI }}
// Bytes returns a raw byte slice of the asset. Changing content of slice will result into segfault.
func (a *Asset) Bytes() []byte {
	return a.blob
}
{{- end }}

type assetReader struct {
	bytes.Reader
}

func (r *assetReader) Close() error {
	r.Reset(nil)
	return nil
}

// Opens asset as an io.ReadCloser. Returns os.ErrNotExist if no asset is found.
{{- if or .Params.BuildFsAPI }}
func Open(name string) (File, error) {
	return root.Open(name)
}
{{- else }}
func Open(name string) (io.ReadCloser, error) {
	if asset, ok := idx[name]; !ok {
		return nil, os.ErrNotExist
	} else {
{{- if .Params.CompressAssets }}
		if asset.isCompressed {
			ungzip, _ := gzip.NewReader(bytes.NewReader(asset.blob))
			return ungzip, nil
		} else {
{{- end }}
			ret := &assetReader{}
			ret.Reset(asset.blob)
			return ret, nil
{{- if .Params.CompressAssets }}
		}
{{- end }}
	}
}
{{- end }}

// Gets asset by name. Returns nil if no asset found.
func Get(name string) *Asset {
	if entry, ok := idx[name]; ok {
		return entry
	} else {
		return nil
	}
}

// Get asset by name. Panics if no asset found.
func Must(name string) *Asset {
	if entry, ok := idx[name]; ok {
		return entry
	} else {
		panic("asset " + name + " not found")
	}
}

type directoryAsset struct {
	name  string
	dirs  []directoryAsset
	files []Asset
}

var root *directoryAsset

{{- if .Params.BuildHttpFsAPI }}
type httpFileSystem struct {}

func (*httpFileSystem) Open(name string) (http.File, error) {
	return root.Open(name)
}

func HttpFileSystem() http.FileSystem {
	return &httpFileSystem{}
}
{{- end }}
{{- if or .Params.BuildFsAPI }}

// A File is returned by virtual FileSystem's Open method.
// The methods should behave the same as those on an *os.File.
type File interface {
	io.Closer
	io.Reader
	io.Seeker
	Readdir(count int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
	// The CopyTo method copies file content to the target path.
	// If file with the same name, size and modification time exists,
	// it will not be overwritten, unless overwrite = true is specified.
	// {{.Pkg}}.Root().CopyTo(".", mode, false) will effectively
	// extract content of the filesystem to the current directory (which
	// makes it the most space-wise inefficient self-extracting archive
	// ever).
	CopyTo(target string, mode os.FileMode, overwrite bool) error
}

func (d *directoryAsset) Open(name string) (File, error) {
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	p := path.Clean(name)
	if p == "." {
		return &directoryAssetFile{dir: d}, nil
	} else {
		var first, rest string
		i := strings.IndexByte(p, '/')
		if i == -1 {
			first = p
		} else {
			first = p[:i]
			rest = p[i+1:]
		}
		for j := range d.dirs {
			if d.dirs[j].name == first {
				if rest == "" {
					return &directoryAssetFile{dir: &d.dirs[j]}, nil
				} else {
					return d.dirs[j].Open(rest)
				}
			}
		}
		if rest != "" {
			return nil, os.ErrNotExist
		}
		for j := range d.files {
			if d.files[j].name == first {
{{- if .Params.CompressAssets }}
				if d.files[j].isCompressed {
					ret := &assetCompressedFile{asset: &d.files[j]}
					ret.Reset(bytes.NewReader(d.files[j].blob))
					return ret, nil
				} else {
{{- end }}
					ret := &assetFile{asset: &d.files[j]}
					ret.Reset(d.files[j].blob)
					return ret, nil
{{- if .Params.CompressAssets }}
				}
{{- end }}
			}
		}
		return nil, os.ErrNotExist
	}
}

type directoryAssetFile struct {
	dir *directoryAsset
	pos int
}

func (d *directoryAssetFile) Close() error {
	if d.pos < 0 {
		return os.ErrClosed
	}
	d.pos = -1
	return nil
}

func (d *directoryAssetFile) Read([]byte) (int, error) {
	if d.pos < 0 {
		return 0, os.ErrClosed
	}
	return 0, io.EOF
}

func (d *directoryAssetFile) Stat() (os.FileInfo, error) {
	if d.pos < 0 {
		return nil, os.ErrClosed
	}
	return d.dir, nil
}

func (d *directoryAssetFile) Seek(pos int64, whence int) (int64, error) {
	if d.pos < 0 {
		return 0, os.ErrClosed
	}
	if whence == io.SeekStart && pos == 0 {
		d.pos = 0
		return 0, nil
	} else {
		return 0, os.ErrInvalid
	}
}

func (d *directoryAssetFile) Readdir(count int) ([]os.FileInfo, error) {
	if d.pos < 0 {
		return nil, os.ErrClosed
	}
	ret := make([]os.FileInfo, len(d.dir.dirs) + len(d.dir.files))
	i := 0
	for j := range d.dir.dirs {
		ret[j + i] = &d.dir.dirs[j]
	}
	i = len(d.dir.dirs)
	for j := range d.dir.files {
		ret[j + i] = &d.dir.files[j]
	}
	if count <= 0 {
		return ret, nil
	} else if d.pos > len(ret) {
		return nil, io.EOF
	} else {
		return ret[d.pos:d.pos+count], nil
	}
}

func (d *directoryAsset) copyTo(target string, dirmode os.FileMode, mode os.FileMode, overwrite bool) error {
	dname := filepath.Join(target, d.name)
	err := os.MkdirAll(dname, dirmode)
	if err != nil {
		return err
	}
	for i := range d.dirs {
		if err = d.dirs[i].copyTo(dname, dirmode, mode, overwrite); err != nil {
			return err
		}
	}
	for i := range d.files {
		if err = d.files[i].CopyTo(dname, mode, overwrite); err != nil {
			return err
		}
	}
	return nil
}

func (d *directoryAssetFile) CopyTo(target string, mode os.FileMode, overwrite bool) error {
	dirmode := ((mode&0444)>>2)|mode
	return d.dir.copyTo(target, dirmode, mode, overwrite)
}

func (d *directoryAsset) Name() string       { return d.name }
func (d *directoryAsset) Size() int64        { return 0 }
func (d *directoryAsset) Mode() os.FileMode  { return os.ModeDir | 0555 }
func (d *directoryAsset) ModTime() time.Time { return stamp }
func (d *directoryAsset) IsDir() bool        { return true }
func (d *directoryAsset) Sys() interface{}   { return nil }

type assetFile struct {
	assetReader
	asset *Asset
}

func (a *assetFile) Stat() (os.FileInfo, error) {
	return a.asset, nil
}

func (a *assetFile) Readdir(int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func (a *assetFile) CopyTo(target string, mode os.FileMode, overwrite bool) error {
	return a.asset.CopyTo(target, mode, overwrite)
}

{{- if .Params.CompressAssets }}
type assetCompressedFile struct {
	gzip.Reader
	asset *Asset
}

func (a *assetCompressedFile) Stat() (os.FileInfo, error) {
	return a.asset, nil
}

func (a *assetCompressedFile) Seek(int64, int) (int64, error) {
	return 0, os.ErrInvalid
}

func (a *assetCompressedFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func (a *assetCompressedFile) CopyTo(target string, mode os.FileMode, overwrite bool) error {
	return a.asset.CopyTo(target, mode, overwrite)
}

{{- end }}
{{- end }}

var idx = make(map[string]*Asset)
var stamp time.Time

func init() {
	stamp = time.Unix({{.Date}})
	bb := blob_bytes({{.Size}})
	bs := blob_string({{.Size}})
{{ .DirectoryCode -}}
{{ .IndexCode -}}
}

{{- if .Params.BuildHttpHelperAPI }}
{{- if .Has404Asset }}
var http404Asset *Asset
{{- end }}
// ServeHTTP provides a convenience handler whenever embedded content should be served from the root URI.
var ServeHTTP = HTTPHandlerWithPrefix("")

// HTTPHandlerWithPrefix provides a simple way to serve embedded content via
// Go standard HTTP server and returns an http handler function. The "prefix"
// will be stripped from the request URL to serve embedded content from non-root URI
func HTTPHandlerWithPrefix(prefix string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" && req.Method != "HEAD" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasPrefix(req.URL.Path, prefix) {
			http.NotFound(w, req)
			return
		}
		reqPath := req.URL.Path[len(prefix):]
		if strings.HasPrefix(reqPath, "/") {
			reqPath = reqPath[1:]
		}
		var status = http.StatusOK
		asset := Get(reqPath)
{{- if .Has404Asset }}
		if asset == nil {
			asset = http404Asset
			status = http.StatusNotFound
		}
{{- else }}
		if asset == nil {
			http.NotFound(w, req)
			return
		}
{{- end }}
		if tag := req.Header.Get("If-None-Match"); tag != "" {
			if strings.HasPrefix("W/", tag) || strings.HasPrefix("w/", tag) {
				tag = tag[2:]
			}
			if tag, err := strconv.Unquote(tag); err == nil && tag == asset.tag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		if mtime := req.Header.Get("If-Modified-Since"); mtime != "" {
			if ts, err := time.Parse(time.RFC1123, mtime); err == nil && !ts.Before(stamp) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
{{- if .Params.CompressAssets }}
		var deflate = asset.isCompressed
		if encs, ok := req.Header["Accept-Encoding"]; ok {
			for _, enc := range encs {
				if strings.Contains(enc, "gzip") {
					if deflate {
						w.Header().Set("Content-Encoding", "gzip")
					}
					deflate = false
					break
				}
			}
		}
		if !deflate {
			w.Header().Set("Content-Length", strconv.FormatInt(int64(len(asset.blob)), 10))
		}
{{- else }}
		w.Header().Set("Content-Length", strconv.FormatInt(int64(asset.size), 10))
{{- end }}
		w.Header().Set("Content-Type", asset.mime)
		w.Header().Set("Etag", strconv.Quote(asset.tag))
		w.Header().Set("Last-Modified", stamp.Format(time.RFC1123))
		w.WriteHeader(status)
		if req.Method != "HEAD" {
{{- if .Params.CompressAssets }}
			if deflate {
				ungzip, _ := gzip.NewReader(bytes.NewReader(asset.blob))
				defer ungzip.Close()
				io.Copy(w, ungzip)
			} else {
{{- end }}
				w.Write(asset.blob)
{{- if .Params.CompressAssets }}
			}
{{- end }}
		}
	}
}
{{end}}