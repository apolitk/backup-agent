package archiver

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ProgressCallback func(processed int64)

type Archiver interface {
	CreateArchive(sourcePath, destFile string, progress ProgressCallback) error
	ExtractArchive(archivePath, destPath string, progress ProgressCallback) error
}

type tarArchiver struct{}

func New() Archiver {
	return &tarArchiver{}
}

func (t *tarArchiver) CreateArchive(sourcePath, destFile string, progress ProgressCallback) error {
	file, err := os.Create(destFile)
	if err != nil {
		return fmt.Errorf("create dest file: %w", err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	var processed int64

	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(path, sourcePath+string(os.PathSeparator))
		if relPath == "" {
			return nil
		}

		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			n, err := io.Copy(tw, f)
			if err != nil {
				return err
			}

			processed += n
			if progress != nil {
				progress(processed)
			}
		}

		return nil
	})
}

func (t *tarArchiver) ExtractArchive(archivePath, destPath string, progress ProgressCallback) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var processed int64

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read header: %w", err)
		}

		target := filepath.Join(destPath, header.Name)

		// Защита от path traversal
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destPath)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}

			n, err := io.CopyN(outFile, tr, header.Size)
			outFile.Close()

			if err != nil && err != io.EOF {
				return err
			}

			processed += n
			if progress != nil {
				progress(processed)
			}
		}
	}

	return nil
}