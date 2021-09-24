package pkg

import (
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/vdoctl/cmd"
	"net"
	"net/url"
	"os"
	"strings"
)

func CheckIfUrl(str string) bool {
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

//ds:///vmfs/volumes/6127d203-83712bb4-f4ae-02000c01829c/

func checkIPAddress(ip string) bool {
	return net.ParseIP(ip) != nil
}

func PromptGetInput(label string, err error, flag cmd.ValidationFlags) string {
	validate := func(input string) error {
		if len(input) <= 0 {
			return err
		}

		if flag == cmd.IsURL && !CheckIfUrl(input) {
			return errors.New("Please provide a valid URL")

		}

		if flag == cmd.IsIP && !checkIPAddress(input) {
			return errors.New("Please provide a valid IP address")
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
	if flag == cmd.IsPwd {
		prompt.Mask = '*'
	}

	res, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}
	return res
}
