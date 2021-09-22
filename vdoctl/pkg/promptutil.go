package pkg

import (
	"fmt"
	"github.com/manifoldco/promptui"
	"net"
	"net/url"
	"os"
	"strings"
)

func IsUrl(str string) bool {
	url, err := url.ParseRequestURI(str)
	if err != nil {
		return false
	}

	address := net.ParseIP(url.Host)

	if address == nil {

		return strings.Contains(url.Host, ".")
	}

	return true
}

//func checkIPAddress(ip string) bool {
//	return net.ParseIP(ip) != nil
//}

func PromptGetInput(label string, err error) string {
	validate := func(input string) error {
		if len(input) <= 0 {
			return err
		}
		return nil
	}

	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: templates,
		Validate:  validate,
	}

	res, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}
	return res
}
