package command

func ReStartContainer(containerFlag string) error {
	err := StopContainer(containerFlag)
	if err != nil {
		return err
	}

	err = StartContainer(containerFlag)
	if err != nil {
		return err
	}
	return nil
}
