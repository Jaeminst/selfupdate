package selfupdate

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
)

// ErrNotSupported is returned by `Manage` when it is not possible to manage the current application.
var ErrNotSupported = errors.New("operating system not supported")

// Source define an interface that is able to get an update
type Source interface {
	Get() (io.ReadCloser, int64, error) // Get the executable to be updated to
}

// Config define extra parameter necessary to manage the updating process
type Config struct {
	FetchOnStart           bool
	Zip                    bool
	Source                 Source            // Necessary Source for update
	RestartConfirmCallback func() bool       // if present will ask for user acceptance before restarting app
	UpgradeConfirmCallback func(string) bool // if present will ask for user acceptance, it can present the message passed
	ExitCallback           func(error)       // if present will be expected to handle app exit procedure
}

// Updater is managing update for your application in the background
type Updater struct {
	lock       sync.Mutex
	conf       *Config
	executable string
}

// CheckNow will manually trigger a check of an update and if one is present will start the update process
func (u *Updater) CheckNow() error {
	u.lock.Lock()
	defer u.lock.Unlock()

	if ask := u.conf.UpgradeConfirmCallback; ask != nil {
		if !ask("New version found") {
			log.Println("The user didn't confirm the upgrade.")
			return nil
		}
	}

	r, contentLength, err := u.conf.Source.Get()
	if err != nil {
		return err
	}
	defer r.Close()

	// ZIP 파일인지 여부에 따른 분기
	if u.conf.Zip {
		u.executable, err = applyUpdateForZip(r)
		if err != nil {
			return err
		}
	} else {
		pr := &progressReader{
			Reader: r,
			bar:    progressbar.DefaultBytes(contentLength),
		}
		u.executable, err = applyUpdate(pr)
		if err != nil {
			return err
		}
	}

	if ask := u.conf.RestartConfirmCallback; ask != nil {
		ask()
	}
	return u.Restart()
}

// Restart once an update is done can trigger a restart of the binary. This is useful to implement a restart later policy.
func (u *Updater) Restart() error {
	return restart(u.conf.ExitCallback, u.executable)
}

// Manage sets up an Updater and runs it to manage the current executable.
func Manage(conf *Config) (*Updater, error) {
	updater := &Updater{conf: conf}

	go func() {
		if updater.conf.FetchOnStart {
			err := updater.CheckNow()
			if err != nil {
				log.Println("Upgrade error: ", err)
			}
		}
	}()

	// TODO check if we can support the current app!
	return updater, nil
}

func applyUpdate(reader io.Reader) (string, error) {
	opts := &Options{}

	err := apply(reader, opts)
	if err != nil {
		return "", err
	}
	return opts.TargetPath, nil
}

// applyUpdateForZip extracts the .app folder or executable from the ZIP file and applies the update.
func applyUpdateForZip(body io.Reader) (string, error) {
	extractedPath, err := unzip(body)
	if err != nil {
		return "", fmt.Errorf("failed to unzip: %v", err)
	}

	opts := &Options{}

	if strings.HasSuffix(extractedPath, ".app") {
		err = applyAppDirectory(extractedPath, opts)
		if err != nil {
			return "", err
		}
	} else {
		file, err := getFileReader(extractedPath)
		if err != nil {
			return "", fmt.Errorf("failed to open extracted file: %v", err)
		}
		err = apply(file, opts)
		if err != nil {
			return "", err
		}
	}

	return opts.TargetPath, nil
}

func unzip(body io.Reader) (string, error) {
	// Body를 메모리에 읽어오기
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}

	// []byte를 io.ReaderAt로 변환
	reader := bytes.NewReader(data)

	// zip.Reader 생성
	r, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return "", err
	}

	// .app 디렉토리와 .exe 파일 존재 여부를 체크할 변수
	var rootFileName string

	// 시스템 임시 디렉토리 가져오기
	destDir := os.TempDir()

	// zip 파일 안의 모든 파일을 순회
	for _, f := range r.File {
		// 파일 경로 설정
		fPath := filepath.Join(destDir, f.Name)

		// 디렉토리일 경우
		if f.FileInfo().IsDir() {
			// 디렉토리 이름이 .app으로 끝나는지 확인
			if strings.HasSuffix(f.Name, ".app/") {
				rootFileName = fPath
			}
			// 디렉토리 생성
			if err := os.MkdirAll(fPath, os.ModePerm); err != nil {
				log.Printf("error make file dir: %s", fPath)
				return "", err
			}
		} else {
			// 파일일 경우 해당 파일을 압축 해제
			if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
				return "", err
			}
			outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return "", err
			}
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			_, err = io.Copy(outFile, rc)

			// 파일 해제 후 리소스 해제
			outFile.Close()
			rc.Close()

			if err != nil {
				return "", err
			}

			// .exe 파일인지 확인
			if strings.HasSuffix(f.Name, ".exe") {
				rootFileName = fPath
			}

			// 루트 경로에 있는 파일의 경우 이름 저장
			if filepath.Dir(f.Name) == "." {
				rootFileName = fPath
			}
		}
	}

	return rootFileName, nil
}

// getFileReader는 압축 해제된 파일의 경로에서 io.Reader를 반환합니다.
func getFileReader(filePath string) (io.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// 파일 내용을 메모리로 읽어들임
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// bytes.Reader로 변환하여 io.Reader로 반환
	return bytes.NewReader(buffer.Bytes()), nil
}
