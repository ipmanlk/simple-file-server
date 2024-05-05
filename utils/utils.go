package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/google/uuid"
)

// Generate a unique UUID for each file
func GenerateUUID() string {
	return uuid.New().String()
}

// Calculate SHA-256 hash of a file
func CalculateFileHash(file io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ReusableReader wraps an io.Reader and provides a way to reset the reader to the beginning of the stream
// after reaching the end of the stream.
// This is useful when you need to read the same stream multiple times.
// Credits: https://blog.flexicondev.com/read-go-http-request-body-multiple-times
type reusableReader struct {
	io.Reader
	readBuf *bytes.Buffer
	backBuf *bytes.Buffer
}

func ReusableReader(r io.Reader) io.Reader {
	readBuf := bytes.Buffer{}
	readBuf.ReadFrom(r) // TODO: handle error properly
	backBuf := bytes.Buffer{}

	return reusableReader{
		io.TeeReader(&readBuf, &backBuf),
		&readBuf,
		&backBuf,
	}
}

func (r reusableReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == io.EOF {
		r.reset()
	}
	return n, err
}

func (r reusableReader) reset() {
	io.Copy(r.readBuf, r.backBuf)
}

func (r reusableReader) Close() error { return nil }
