package udpchat

import (
	"fmt"
)

func PrintHelpInfo() {
	fmt.Println(
		"Usage:\n" +
			"		>>> help ------------------ get help information\n" +
			"		>>> history --------------- get chat history\n" +
			"		>>> send: <msg> ----------- send message to the server\n" +
			"		>>> sendfile: <filename> -- request for file transfer\n" +
			"		>>> quit ------------------ exit from udpchat")
}
