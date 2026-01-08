package store

import (
	"errors"
	"strings"
)

var (
	ErrEmptyPath     = errors.New("path cannot be empty")
	ErrInvalidPath   = errors.New("path contains invalid characters")
	ErrRelativePath  = errors.New("path cannot contain .. segments")
	ErrAbsolutePath  = errors.New("path cannot start with /")
	ErrTrailingSlash = errors.New("path cannot end with /")
)

func NormalizePath(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	if strings.HasPrefix(path, "/") {
		return "", ErrAbsolutePath
	}

	if strings.HasSuffix(path, "/") {
		return "", ErrTrailingSlash
	}

	segments := strings.Split(path, "/")
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		if segment == ".." {
			return "", ErrRelativePath
		}
		if strings.ContainsAny(segment, "\x00") {
			return "", ErrInvalidPath
		}
	}

	normalized := strings.Join(segments, "/")
	normalized = strings.ReplaceAll(normalized, "//", "/")

	return normalized, nil
}

func SplitPath(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}
