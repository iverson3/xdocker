package command

import (
	"fmt"
	"os"
	"github.com/iverson3/xdocker/images"
	"text/tabwriter"
)

func ListImages() error {
	images, err := images.GetAllImages()
	if err != nil {
		return err
	}

	// 格式化输出镜像信息列表
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, _ = fmt.Fprint(w, "Image ID\tNAME\tTAG\tSIZE\tCREATED\n")
	for _, item := range images {
		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Name,
			item.TAG,
			item.Size,
			item.CreateTime)
	}
	err = w.Flush()
	if err != nil {
		return err
	}

	return nil
}
