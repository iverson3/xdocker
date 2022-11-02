package command

import (
	"fmt"
	"os"
	"reflect"
	"github.com/iverson3/xdocker/util"
	"text/tabwriter"
)

func InspectContainer(container string) error {
	exists, containerName, err := util.ContainerIsExists(container)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("container not exists: %s", container)
	}

	info, err := util.GetContainerInfoByName(containerName)
	if err != nil {
		return err
	}

	// 使用反射遍历结构体字段
	t := reflect.TypeOf(*info)
	values := reflect.ValueOf(*info)

	// 格式化输出容器详细信息
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	for k := 0; k < t.NumField(); k++ {
		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t\n",
			t.Field(k).Tag.Get("json"),
			values.Field(k).Interface(),
		)
	}

	err = w.Flush()
	if err != nil {
		return err
	}
	return nil
}