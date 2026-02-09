package main

import (
	"fmt"
	"os"
)

func fail(e error) {
	fmt.Printf("Error: %s\n", e.Error())
	os.Exit(1)
}

func main() {
	args_without_proc := os.Args[1:]
	if len(args_without_proc) < 1 {
		fail(fmt.Errorf("Not enough command line arguments to execute (try -help)"))
	}

	parsed_input, error_from_parsing := parse_input(args_without_proc)
	if error_from_parsing != nil {
		fail(error_from_parsing)
	}

	if could_not_find_pack_man := parsed_input.detect_pkg_man(); could_not_find_pack_man != nil {
		fail(could_not_find_pack_man)
	}

	sdload_error := execute_based_on_mode(parsed_input)
	if sdload_error != nil {
		fail(sdload_error)
	}
}
