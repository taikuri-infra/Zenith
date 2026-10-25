package k8sops

import (
	"archive/tar"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// extractArchive unpacks a backup archive into dst, rejecting unsafe entries.
func extractArchive(archive, dst string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		name := filepath.ToSlash(h.Name)
		if name == "" || strings.HasPrefix(name, "/") || filepath.IsAbs(h.Name) {
			return fmt.Errorf("unsafe path in archive: %q", h.Name)
		}
		for _, part := range strings.Split(name, "/") {
			if part == ".." {
				return fmt.Errorf("unsafe path in archive: %q", h.Name)
			}
		}
		target := filepath.Join(dst, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				return err
			}
			// Bound the copy by the entry's declared size; a short read means a truncated archive.
			if n, err := io.CopyN(out, tr, h.Size); err != nil {
				out.Close()
				if err == io.EOF {
					return fmt.Errorf("truncated archive entry %q: got %d of %d bytes", h.Name, n, h.Size)
				}
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported entry type in archive: %q", h.Name)
		}
	}
}

// afterArchiveOpen is a test seam invoked once the archive file exists.
var afterArchiveOpen = func() error { return nil }

// writeArchive writes the archive and removes it again on any failure.
func writeArchive(path, src string) (err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			os.Remove(path)
		}
	}()
	if err = afterArchiveOpen(); err != nil {
		return err
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel, _ := filepath.Rel(src, p)
		if rel == "." || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		h, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		h.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(tw, in)
		return err
	})
	if err != nil {
		return err
	}
	if err = tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

// CompareVersions compares two semver strings (with optional prerelease): -1, 0 or 1.
func CompareVersions(a, b string) int {
	split := func(v string) ([]string, []string) {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		core, pre, hasPre := strings.Cut(v, "-")
		var p []string
		if hasPre {
			p = strings.Split(pre, ".")
		}
		return strings.Split(core, "."), p
	}
	ac, ap := split(a)
	bc, bp := split(b)
	for i := 0; i < len(ac) || i < len(bc); i++ {
		var x, y int
		if i < len(ac) {
			x, _ = strconv.Atoi(ac[i])
		}
		if i < len(bc) {
			y, _ = strconv.Atoi(bc[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	switch {
	case ap == nil && bp == nil:
		return 0
	case ap == nil:
		return 1
	case bp == nil:
		return -1
	}
	for i := 0; i < len(ap) || i < len(bp); i++ {
		if i >= len(ap) {
			return -1
		}
		if i >= len(bp) {
			return 1
		}
		x, xe := strconv.Atoi(ap[i])
		y, ye := strconv.Atoi(bp[i])
		switch {
		case xe == nil && ye == nil:
			if x != y {
				if x < y {
					return -1
				}
				return 1
			}
		case xe == nil:
			return -1
		case ye == nil:
			return 1
		default:
			if c := strings.Compare(ap[i], bp[i]); c != 0 {
				return c
			}
		}
	}
	return 0
}

func decodeB64(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return "", fmt.Errorf("decode secret value: %w", err)
	}
	return string(b), nil
}
