package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"studygolang/docker/xdocker/model"
	"studygolang/docker/xdocker/util"
	"text/tabwriter"
)

func ListContainer() error {
	// 遍历 /var/run/xdocker 便可以得到所有的容器目录，读取其下的config.json便可以得到容器信息
	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, "")
	dirUrl = dirUrl[:len(dirUrl)-1]

	dirs, err := ioutil.ReadDir(dirUrl)
	if err != nil {
		return err
	}

	var containers []*model.ContainerInfo
	for _, dir := range dirs {
		info, err := util.GetContainerInfo(dir)
		if err != nil {
			return err
		}

		containers = append(containers, info)
	}

	// 格式化输出容器信息列表
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, _ = fmt.Fprint(w, "ID\tNAME\tPID\tImage\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containers {
		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Name,
			item.Pid,
			item.Image,
			item.Status,
			item.Command,
			item.CreateTime)
	}
	err = w.Flush()
	if err != nil {
		return err
	}

	return nil
}