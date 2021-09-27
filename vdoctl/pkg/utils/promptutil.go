package utils

import (
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"net"
	"net/url"
	"os"
	"strings"
)

type ValidationFlags string

const (
	IsPwd    ValidationFlags = "isPwd"
	IsURL    ValidationFlags = "isURL"
	IsIP     ValidationFlags = "isIP"
	IsString ValidationFlags = "isString"
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

func checkIPAddress(ip string) bool {
	return net.ParseIP(ip) != nil
}

func PromptGetInput(label string, err error, flag ValidationFlags) string {
	validate := func(input string) error {
		if len(input) <= 0 {
			return err
		}

		if flag == IsURL && !CheckIfUrl(input) {
			return errors.New("Please provide a valid URL")

		}

		if flag == IsIP && !checkIPAddress(input) {
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
	if flag == IsPwd {
		prompt.Mask = '*'
	}

	res, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}
	return res
}

func PromptGetSelect(items []string, label string) string {
	index := -1
	var result string
	var err error

	for index < 0 {
		prompt := promptui.SelectWithAdd{
			Label: label,
			Items: items,
		}

		index, result, err = prompt.Run()

		if index == -1 {
			items = append(items, result)
		}
	}

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	return result
}
