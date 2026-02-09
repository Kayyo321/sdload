package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const (
	ModeSearch = iota
	ModeFindDownload
	ModeDownload
	ModeHelp
)

type ParsedInput struct {
	mode     int
	todo     []string
	settings map[string]any
	pkg_mgr  string
}

func parse_input(cmd_ln_args []string) (*ParsedInput, error) {
	mode_map := map[string]int{
		"-help":   ModeHelp,
		"-search": ModeSearch,
		"-finddl": ModeFindDownload,
		"-downl":  ModeDownload,
	}

	pi := new(ParsedInput)

	if mode, ok := mode_map[cmd_ln_args[0]]; !ok {
		return nil, fmt.Errorf("Unexpected mode: %s", cmd_ln_args[0])
	} else {
		pi.mode = mode
	}

	if pi.mode == ModeHelp { // No additional args needed
		return pi, nil
	}

	pi.todo = make([]string, 0)
	cmd_ln_args = cmd_ln_args[1:]

	idx := 0
	for {
		pi.todo = append(pi.todo, cmd_ln_args[idx])
		idx++
		if (idx >= len(cmd_ln_args)) {
			break
		}

		if cmd_ln_args[idx] != "+" {
			break
		}

		idx++ // skip the "+" separator
		if (idx >= len(cmd_ln_args)) {
			break
		}
	}

	if (idx >= len(cmd_ln_args)) { // No settings listed, we're done
		return pi, nil
	}

	pi.settings = make(map[string]any)
	cmd_ln_args = cmd_ln_args[(idx + 1):]

	for idx = 0; idx < len(cmd_ln_args); idx++ {
		arg := cmd_ln_args[idx]

		setting := strings.Split(arg, "=")
		if len(setting) != 2 {
			return nil, fmt.Errorf("Unexpected setting: `%s`, expected <setting>=<value> (no space between `=`)", arg)
		}

		setting_key := setting[0]
		setting_val := setting[1]

		pi.settings[setting_key] = setting_val
	}

	return pi, nil
}

func (pi *ParsedInput) detect_pkg_man() error {
	if _, err := exec.LookPath("apt-cache"); err == nil {
		pi.pkg_mgr = "apt"
		return nil
	}
	if _, err := exec.LookPath("pacman"); err == nil {
		pi.pkg_mgr = "pacman"
		return nil
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		pi.pkg_mgr = "dnf"
		return nil
	}
	if _, err := exec.LookPath("zypper"); err == nil {
		pi.pkg_mgr = "zypper"
		return nil
	}
	if _, err := exec.LookPath("apk"); err == nil {
		pi.pkg_mgr = "apk"
		return nil
	}
	if _, err := exec.LookPath("snap"); err == nil {
		pi.pkg_mgr = "snap"
		return nil
	}
	if _, err := exec.LookPath("flatpak"); err == nil {
		pi.pkg_mgr = "flatpak"
		return nil
	}

	pi.pkg_mgr = ""
	return fmt.Errorf("No supported package manager found on system")
}

func (pi *ParsedInput) search_setting(key string, prototype any) (any, bool, error) {
	raw, ok := pi.settings[key]
	if !ok {
		return nil, false, nil
	}

	switch prototype.(type) {
	case string:
		if v, ok := raw.(string); ok {
			return v, true, nil
		}
	case bool:
		if v, ok := raw.(bool); ok {
			return v, true, nil
		}
	case int:
		if v, ok := raw.(int); ok {
			return v, true, nil
		}
	case int64:
		if v, ok := raw.(int64); ok {
			return v, true, nil
		}
	case float64:
		if v, ok := raw.(float64); ok {
			return v, true, nil
		}
	}

	// Otherwise parse from string representation.
	s, isString := raw.(string)
	if !isString {
		s = fmt.Sprintf("%v", raw)
	}

	switch prototype.(type) {
	case string:
		return s, true, nil
	case bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, true, err
		}
		return b, true, nil
	case int:
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, true, err
		}
		return n, true, nil
	case int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, true, err
		}
		return n, true, nil
	case float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, true, err
		}
		return f, true, nil
	default:
		return nil, true, fmt.Errorf("unsupported target type for SearchSetting")
	}
}
