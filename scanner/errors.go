package scanner

import "runtime"

type ErrTrufflehogNotInstalled struct {
	installInstructions string
	installCommand      string
}

func NewTrufflehogNotInstalledError() *ErrTrufflehogNotInstalled {
	switch runtime.GOOS {
	case "darwin":
		fallthrough
	case "linux":
		return &ErrTrufflehogNotInstalled{
			installInstructions: "Important dependency missing. Please reinstall axi. You can use the following command to install: ",
			installCommand:      "~/.axi/bin/axi install",
		}
	}

	return &ErrTrufflehogNotInstalled{installInstructions: "Please install trufflehog. Use the following command to install it: "}
}

func (e *ErrTrufflehogNotInstalled) Error() string {
	return "Trufflehog not installed. \n" +
		e.installInstructions + "\n" +
		e.installCommand
}
