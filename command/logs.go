package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/util"
)

func LogContainer(container string) error {
	exists, containerName, err := util.ContainerIsExists(container)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("container not exists: %s", container)
	}

	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, containerName)
	logPath := dirUrl + model.ContainerLogFileName

	file, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprint(os.Stdout, string(content))
	return nil
}