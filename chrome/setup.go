package chrome

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var DriverPath = filepath.Join(os.TempDir(), "chromedriver")

func latestRelease() (version string) {
	res, err := http.Get("http://chromedriver.storage.googleapis.com/LATEST_RELEASE")
	if err != nil {
		return ""
	}
	defer res.Body.Close()

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return ""
	}
	return string(buf[:len(buf)-1])
}

func targetArch() (target string, err error) {
	var arch string
	switch runtime.GOARCH {
	case "386":
		arch = "32"
	case "amd64":
		arch = "64"
	default:
		return "", errors.New("not supported")
	}

	switch runtime.GOOS {
	case "darwin":
		return "mac32", nil
	case "linux":
		return "linux" + arch, nil
	case "windows":
		return "win32", nil
	default:
		return "", errors.New("not supported")
	}
}

func SetupDriver() error {
	target, err := targetArch()
	if err != nil {
		return err
	}

	// version := latestRelease()
	version := "2.37"

	_, err = os.Stat(DriverPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if !os.IsNotExist(err) {
		buf, err := exec.Command(DriverPath, "--version").CombinedOutput()
		if err != nil {
			return err
		}
		infos := bytes.Split(buf, []byte(" "))
		if len(infos) != 3 {
			return errors.New(fmt.Sprintf("unexpected version string: %s", string(buf)))
		}
		current := string(infos[1])

		if strings.HasPrefix(current, version) {
			return nil
		}
	}

	url := fmt.Sprintf("http://chromedriver.storage.googleapis.com/%s/chromedriver_%s.zip", version, target)
	log.Printf("download from: %s", url)

	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	r, err := zip.NewReader(bytes.NewReader(body), res.ContentLength)
	if err != nil {
		return err
	}

	for _, f := range r.File {
		savepath := filepath.Join(os.TempDir(), f.Name)
		dst, err := os.OpenFile(savepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer dst.Close()
		src, err := f.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		io.Copy(dst, src)

		log.Printf("saved: %s", savepath)
	}
	return nil
}
