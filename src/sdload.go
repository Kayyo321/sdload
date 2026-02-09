package main

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	DefaultSearchMax = 10
)

func execute_based_on_mode(pi *ParsedInput) error {
	out := fmt.Errorf("Unexpected mode: %d", pi.mode)

	switch pi.mode {
	case ModeHelp:
		print_help()
		out = nil

	case ModeSearch:
		fmt.Println("Searching for:", pi.todo[0])
		lines, _out := search(pi)
		out = _out

		if out == nil {
			fmt.Println(strings.Join(lines, "\n"))
		}

	case ModeFindDownload:
		cmdName, args, _out := find_download(pi)
		out = _out

		if out == nil {
			fmt.Printf("Recommended download query: %s %s\n", cmdName, strings.Join(args, " "))
		}

	case ModeDownload:
		out = download(pi)

	default:
		break
	}

	if out == nil && len(pi.todo) > 1 {
		pi.todo = pi.todo[1:] // chop off last thing done
		out = execute_based_on_mode(pi)
	}

	return out
}

func print_help() {
	fmt.Println("sdload - A simple package search and download tool")
	fmt.Println()
	fmt.Println("Usage: ")
	fmt.Println("  sdload -help")
	fmt.Println("\t*  Show this help message.")
	fmt.Println()
	fmt.Println("  sdload -search <query> [`and` <query> ...] [settings]")
	fmt.Println("\t*  Search for packages matching <query>.")
	fmt.Println()
	fmt.Println("  sdload -finddl <query> [`and` <query> ...] [settings]")
	fmt.Println("\t*  Find the download command for <package_name>.")
	fmt.Println()
	fmt.Println("  sdload -downl <query> [`and` <query> ...] [settings]")
	fmt.Println("\t*  Download and install <package_name>.")
	fmt.Println()
	fmt.Println("Settings:")
	fmt.Println("\tsearch_max=<number>  - Maximum number of search results to display (default 10).")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("\tsdload -search vim search_max=5")
	fmt.Println("\tsdload -finddl curl")
	fmt.Println("\tsdload -downl git")
}

func search(pi *ParsedInput) ([]string, error) {
	if len(pi.todo) == 0 {
		return nil, fmt.Errorf("Nothing to search for")
	}

	search_max := DefaultSearchMax
	if v, ok, err := pi.search_setting("search_max", int(0)); err != nil {
		return nil, err
	} else if ok {
		search_max = v.(int)
	}

	query := pi.todo[0]

	var cmd *exec.Cmd

	switch pi.pkg_mgr {
	case "apt":
		cmd = exec.Command("apt-cache", "search", query)
	case "pacman":
		cmd = exec.Command("pacman", "-Ss", query)
	case "dnf":
		cmd = exec.Command("dnf", "search", query)
	case "zypper":
		cmd = exec.Command("zypper", "se", query)
	case "apk":
		cmd = exec.Command("apk", "search", query)
	case "snap":
		cmd = exec.Command("snap", "find", query)
	case "flatpak":
		cmd = exec.Command("flatpak", "search", query)
	default:
		return nil, fmt.Errorf("no supported package manager detected")
	}

	out, _ := cmd.CombinedOutput()
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if search_max > 0 && len(lines) > search_max {
		lines = lines[:search_max]
	}

	return lines, nil
}

func extract_pkg_name(entry string, pkg_mgr string) string {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return ""
	}

	switch pkg_mgr {
	case "apt":
		// e.g. "bash - GNU Bourne Again SHell"
		// Take everything before the first " - "
		if idx := strings.Index(entry, " - "); idx != -1 {
			return strings.TrimSpace(entry[:idx])
		}
		// Fallback: first field
		fields := strings.Fields(entry)
		if len(fields) > 0 {
			return fields[0]
		}
		return ""

	case "pacman":
		// e.g. "extra/bash 5.2.026-2 (base)"
		// Package name is after "repo/"
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			return ""
		}
		// First field typically "repo/name"
		parts := strings.SplitN(fields[0], "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		// Fallback: first field
		return parts[0]

	case "dnf":
		// e.g. "bash.x86_64 : The GNU Bourne Again shell"
		// Name (including arch) is before first space, but base name is before first dot
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			return ""
		}
		name := fields[0] // "bash.x86_64"
		if dot := strings.Index(name, "."); dot != -1 {
			name = name[:dot]
		}
		return name

	case "zypper":
		// `zypper se` output can be tabular, e.g.:
		// "i+ | bash          | The GNU Bourne Again Shell"
		// or "  | bash-completion | Programmable completion for bash"
		// Strategy: split on '|' and take the 2nd column, trimmed.
		if strings.Contains(entry, "|") {
			parts := strings.Split(entry, "|")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1])
			}
		}
		// Fallback: second "word-ish" token
		fields := strings.Fields(entry)
		if len(fields) >= 2 {
			return fields[1]
		}
		if len(fields) == 1 {
			return fields[0]
		}
		return ""

	case "apk":
		// e.g. "bash-5.2.15-r0"
		// Package name is before last '-' followed by version-ish
		// Simple approach: first field, then strip trailing "-<digits...>"
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			return ""
		}
		name := fields[0]
		// apk names can contain '-', so to be safe, we can just take the field as-is
		// If you really want to strip version, use a heuristic:
		// find the last '-' and check if the suffix starts with a digit
		if idx := strings.LastIndex(name, "-"); idx != -1 && idx+1 < len(name) &&
			name[idx+1] >= '0' && name[idx+1] <= '9' {
			return name[:idx]
		}
		return name

	case "snap":
		// `snap find` often prints a table where first column is name.
		// e.g. "core         Canonical, Ubuntu, and others ..."
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			return ""
		}
		return fields[0]

	case "flatpak":
		// `flatpak search` output is tabular; first column is usually application ID or name.
		// e.g. "org.gnome.Builder   GNOME Builder IDE   org.gnome.Platform ..."
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			return ""
		}
		return fields[0]

	default:
		// Unknown package manager: best effort, first token.
		fields := strings.Fields(entry)
		if len(fields) > 0 {
			return fields[0]
		}
		return ""
	}
}

func find_download(pi *ParsedInput) (string, []string, error) {
	possibilities, err := search(pi)
	if err != nil {
		return "", nil, err
	}

	pkg_names := make([]string, 0, len(possibilities))
	for _, pkg := range possibilities {
		pkg_name := extract_pkg_name(pkg, pi.pkg_mgr)
		if pkg_name == pi.todo[0] {
			return build_download_command(pi, pkg_name)
		}
		pkg_names = append(pkg_names, pkg_name)
	}

	fmt.Print("Multiple possibilities found, did you mean")
	for i, pkg := range pkg_names {
		if i >= len(pkg_names)-1 {
			fmt.Printf(" or '%s': ", pkg)
		} else {
			fmt.Printf(" '%s',", pkg)
		}
	}

	var pkg_intended string
	n, read_err := fmt.Scanln(&pkg_intended)
	if n != 1 || read_err != nil {
		return "", nil, fmt.Errorf("Failed to read user input: %s", read_err.Error())
	}

	for _, pkg := range possibilities {
		if extract_pkg_name(pkg, pi.pkg_mgr) == pkg_intended {
			return build_download_command(pi, pkg_intended)
		}
	}

	return "", nil, fmt.Errorf("No package found matching '%s'", pkg_intended)
}

func build_download_command(pi *ParsedInput, pkg_name string) (string, []string, error) {
	switch pi.pkg_mgr {
	case "apt":
		return "apt-get", []string{"install", pkg_name}, nil
	case "pacman":
		return "pacman", []string{"-S", pkg_name}, nil
	case "dnf":
		return "dnf", []string{"install", pkg_name}, nil
	case "zypper":
		return "zypper", []string{"install", pkg_name}, nil
	case "apk":
		return "apk", []string{"add", pkg_name}, nil
	case "snap":
		return "snap", []string{"install", pkg_name}, nil
	case "flatpak":
		return "flatpak", []string{"install", pkg_name}, nil
	default:
		return "", nil, fmt.Errorf("unsupported package manager: %s", pi.pkg_mgr)
	}
}

func download(pi *ParsedInput) error {
	cmdName, args, err := find_download(pi)
	if err != nil {
		return err
	}

	fmt.Printf("Running: %s %s\n", cmdName, strings.Join(args, " "))

	cmd := exec.Command(cmdName, args...)
	out, _ := cmd.CombinedOutput()
	fmt.Printf("Output of download command:\n%s\n", string(out))

	return nil
}
