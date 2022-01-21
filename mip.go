package main

import (
	"flag"
	"fmt"
	"mik/mic_/compiler_util"
	"mik/mip_/mip_util"
	"os"
)

func main() {
	// arg parse arguments
	installPtr := flag.String("install", "", "install a package from git")
	listPtr := flag.Bool("list", false, "list all packages")
	removePtr := flag.String("remove", "", "remove a package")
	addPkgPtr := flag.String("add_pkg", "", "adds a package at the specified location")

	flag.Parse()

	// Checks if any flag was specified
	if *installPtr != "" || *listPtr != false || *removePtr != "" || *addPkgPtr != "" {
		// Check which is active and if more than one is active
		if *installPtr != "" && *listPtr == false && *removePtr == "" && *addPkgPtr == "" {
			// installs a specified github url
			mip_util.InstallGit(installPtr)
			os.Exit(0)
		} else if *listPtr && *installPtr == "" && *removePtr == "" && *addPkgPtr == "" {
			// lists all the installed packages
			mip_util.ListAll()
			os.Exit(0)
		} else if *removePtr != "" && *installPtr == "" && *listPtr == false && *addPkgPtr == "" {
			// removes a specified package by name
			mip_util.RemovePkg(removePtr)
		} else if *addPkgPtr != "" && *installPtr == "" && *listPtr == false && *removePtr == "" {
			// adds a package (at specified path) to source
			ditch_nil := ""
			mip_util.AddPkg(addPkgPtr, &ditch_nil)
			os.Exit(0)
		} else {
			// Too many args
			compiler_util.NewError("To many arguments specified", "", "", false)
		}
	} else {
		// No args specified
		fmt.Println("Usage:\n\t-install <github_link> to install a pkg from github\n\t-list to list all the installed packages\n\t-remove <pkg_name> to remove a pkg\n\t-add_pkg <dir_path> to add a dir to mik-srd")
	}
}
