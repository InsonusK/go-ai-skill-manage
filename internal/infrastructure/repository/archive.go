package repository

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var httpsGitHub = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)
var sshGitHub = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?/?$`)

func GitHubURL(raw string) (string, string, error) {
	for _, re := range []*regexp.Regexp{httpsGitHub, sshGitHub} {
		m := re.FindStringSubmatch(raw)
		if m != nil {
			return m[1], m[2], nil
		}
	}
	return "", "", fmt.Errorf("not a GitHub repository URL")
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}
type Archive struct {
	Client  HTTPClient
	BaseURL string
}

func (a Archive) Fetch(ctx context.Context, raw, tree, dest string) (string, error) {
	owner, repo, err := GitHubURL(raw)
	if err != nil {
		return "", err
	}
	base := a.BaseURL
	if base == "" {
		base = "https://github.com"
	}
	address := base + "/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/archive/" + url.PathEscape(tree) + ".tar.gz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return "", err
	}
	response, err := a.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("archive HTTP status %d", response.StatusCode)
	}
	gz, err := gzip.NewReader(response.Body)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	if err := os.MkdirAll(dest, 0755); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(dest)
	if err != nil {
		return "", err
	}
	defer root.Close()
	reader := tar.NewReader(gz)
	top := ""
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		name := strings.TrimSuffix(strings.TrimPrefix(header.Name, "./"), "/")
		if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\:") {
			return "", fmt.Errorf("unsafe archive path %q", header.Name)
		}
		first := strings.Split(name, "/")[0]
		if top == "" {
			top = first
		}
		if first != top {
			return "", fmt.Errorf("archive must contain one root directory")
		}
		mode := os.FileMode(header.Mode) & 0777
		switch header.Typeflag {
		case tar.TypeDir:
			if err := root.MkdirAll(filepath.FromSlash(name), 0755); err != nil {
				return "", err
			}
		case tar.TypeReg, tar.TypeRegA:
			total += header.Size
			if header.Size < 0 || total > 512<<20 {
				return "", fmt.Errorf("archive exceeds 512 MiB")
			}
			if err := root.MkdirAll(filepath.FromSlash(path.Dir(name)), 0755); err != nil {
				return "", err
			}
			file, err := root.OpenFile(filepath.FromSlash(name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return "", err
			}
			_, copyErr := io.Copy(file, reader)
			closeErr := file.Close()
			if copyErr != nil {
				return "", copyErr
			}
			if closeErr != nil {
				return "", closeErr
			}
		default:
			return "", fmt.Errorf("unsafe archive entry type %d", header.Typeflag)
		}
	}
	if top == "" {
		return "", fmt.Errorf("empty archive")
	}
	info, err := root.Stat(top)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("archive root must be a directory")
	}
	return filepath.Join(dest, top), nil
}
