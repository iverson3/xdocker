package command

import (
	"fmt"
	"github.com/iverson3/xdocker/images"
)

func SearchImage(keyword string) error {
	list, err := images.DoSearchImageRequest(keyword)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Println("search result is empty")
		return nil
	}

	for _, item := range list {
		fmt.Printf("%s    ", item)
	}
	fmt.Println()
	return nil
}
