package selfupdate

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/Jaeminst/selfupdate/internal/osext"
)

func restart(exiter func(error), executable string) error {
	// .app 디렉토리일 경우 open 명령을 사용하여 실행
	if filepath.Ext(executable) == ".app" {
		cmd := exec.Command("open", executable)

		// 실행할 때 부모 프로세스의 입출력 사용
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		// cmd.SysProcAttr는 MacOS에서 새 프로세스를 시작할 때 필요한 설정을 담을 수 있음
		cmd.SysProcAttr = &syscall.SysProcAttr{}

		err := cmd.Start() // open 명령 실행
		if exiter != nil {
			exiter(err)
		} else if err == nil {
			os.Exit(0) // 성공 시 현재 프로세스 종료
		}
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if executable == "" {
		executable, err = osext.Executable()
		if err != nil {
			return err
		}
	}

	_, err = os.StartProcess(executable, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   os.Environ(),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Sys:   &syscall.SysProcAttr{},
	})

	if exiter != nil {
		exiter(err)
	} else if err == nil {
		os.Exit(0)
	}
	return err
}
