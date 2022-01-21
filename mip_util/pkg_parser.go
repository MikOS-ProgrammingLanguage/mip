package mip_util

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"mik/mic_/compiler_util"
	"os"
	"os/exec"
	"strconv"
	"strings"

	cp "github.com/otiai10/copy"
)

var src_path string = strings.ReplaceAll(strings.ReplaceAll(readConf(), "\\", "/"), "\n", "")
var mik_src_path string = src_path + "mik-src/"
var req_satisfied_loc string = mik_src_path + "req_satisfied.conf"
var TEMP_CNT int = -1
var DEPENDS bool = false

func strip(str string) string {
	return strings.Join(strings.Fields(str), "")
}
func mk_str(arr []string) string {
	ret_str := ""
	for i, val := range arr {
		if i < 1 {
			continue
		}
		ret_str += val
	}
	return ret_str
}

func Clear() {
	// clear git
	checkErr(os.RemoveAll(mik_src_path + "git"))
	checkErr(os.Mkdir(mik_src_path+"git", 0755))

	// clear temp
	checkErr(os.RemoveAll(mik_src_path + "temp"))
	checkErr(os.Mkdir(mik_src_path+"temp", 0755))
}

func checkErr(err error) {
	if err != nil {
		Clear()
		panic(err)
	}
}

func readConf() string {
	cntnt, err := os.ReadFile("mik.conf")
	if err != nil {
		panic(err)
	}
	return string(cntnt)
}

// Lists all the packages in mik-src
func ListAll() {
	// Loads the content of the req_satisfied
	content, err := os.ReadFile(req_satisfied_loc)
	if err != nil {
		panic(err)
	}

	split_ := strings.Split(string(content), "\n")

	var split []string
	for _, val := range split_ {
		if val == "" {
			continue
		}
		split = append(split, val)
	}

	if len(split) == 1 && split[0] == "" {
		compiler_util.NewInfo("No packages found", "", "", true)
	} else {
		if len(split) == 1 {
			fmt.Println("You currently have 1 package installed")
		} else {
			fmt.Println("You currently have " + strconv.Itoa(len(split)) + " packages installed")
		}
		for i := 0; i < len(split); i++ {
			fmt.Println("\t|____ " + strings.Split(split[i], ":::")[1])
		}
	}
}

// add_pkg adds a pkg to mik-src with a specified path
func AddPkg(path, url *string) bool {
	existing_files, err := os.ReadFile(req_satisfied_loc)
	checkErr(err)
	existing_ := strings.Split(string(existing_files), "\n")
	var existing []string
	if len(existing_) == 1 {
		existing = append(existing, existing_[0])
	} else {
		for _, val := range existing_ {
			if val == "" {
				continue
			}
			existing = append(existing, strings.Split(val, ":::")[1])
		}
	}

	TEMP_CNT++
	var PKG_NAME string
	var CURRENT_TEMP string = fmt.Sprintf("tmp%d", TEMP_CNT)
	var IGNORE_FILES []string
	var DEPENDENCIES []string

	*path = strings.ReplaceAll(strings.ReplaceAll(*path, "\\", "/"), "\n", "")

	// copy dir to to temp
	checkErr(cp.Copy(*path, fmt.Sprintf("%stemp/%s", mik_src_path, CURRENT_TEMP), cp.Options{AddPermission: 0777}))
	out, err := os.ReadFile(fmt.Sprintf("%stemp/%s/milk.pkg", mik_src_path, CURRENT_TEMP))
	if err != nil {
		Clear()
		compiler_util.NewCritical("No milk.pkg found", "At "+fmt.Sprintf("%s", *path), "", true)
	} else {
		pkg_txt := string(out)
		pkg_args := strings.Split(pkg_txt, "\n")
		for _, val := range pkg_args {
			val = strip(val)
			val_2 := strings.SplitN(val, ":", 2)

			if val == "" {
				continue
			}

			if len(val_2) < 2 {
				compiler_util.NewError("No vars specified after <arg>:", "", "", true)
			}
			switch val_2[0] {
			case "package-name":
				PKG_NAME = mk_str(val_2)
				if compiler_util.StringInSlice(PKG_NAME, existing) {
					if !DEPENDS {
						Clear()
						compiler_util.NewCritical("Package: "+PKG_NAME+". Allready esxists", "Aborted...", "", true)
					} else {
						compiler_util.NewInfo("Requirement: "+PKG_NAME+". Allready satisfied", "", "", false)
						// clear git
						checkErr(os.RemoveAll(mik_src_path + "git"))
						checkErr(os.Mkdir(mik_src_path+"git", 0755))
						return true
					}
				}
				if *url == "" {
					*url = PKG_NAME
				}
			case "ignore-file":
				IGNORE_FILES = append(IGNORE_FILES, mk_str(val_2))
			case "depends":
				DEPENDENCIES = append(DEPENDENCIES, mk_str(val_2))
			default:
				compiler_util.NewError("Invalid Argument: "+val_2[0], "", "", true)
			}
		}
		// remove all files in ignore
		for _, val := range IGNORE_FILES {
			checkErr(os.Remove(fmt.Sprintf("%stemp/%s/%s", mik_src_path, CURRENT_TEMP, val)))
		}
		compiler_util.NewInfo("Done Ignoring", "", "", false)

		// make dependencies
		for _, val := range DEPENDENCIES {
			DEPENDS = true
			InstallGit(&val)
			DEPENDS = false
		}

		// make a pkg structure
		checkErr(os.MkdirAll(mik_src_path+"pkg/"+PKG_NAME, 0755))
		checkErr(cp.Copy(fmt.Sprintf("%stemp/%s", mik_src_path, CURRENT_TEMP), fmt.Sprintf("%spkg/%s", mik_src_path, PKG_NAME)))

		// clear temp
		checkErr(os.RemoveAll(fmt.Sprintf("%stemp/%s/", mik_src_path, CURRENT_TEMP)))

		// preprocess all not ignored files ending in .milk
		files_, err := ioutil.ReadDir(fmt.Sprintf("%spkg/%s/", mik_src_path, PKG_NAME))
		checkErr(err)
		var files []string

		// get all files that end in .milk
		for _, val := range files_ {
			name := val.Name()
			if strings.HasSuffix(name, ".milk") {
				files = append(files, name)
			}
		}

		// make a yoink string
		var yoink_str string
		for _, val := range files {
			yoink_str += fmt.Sprintf("#yoink <%s>\n", val)
		}

		// preprocess yoink string and write it
		pth := fmt.Sprintf("%spkg/%s/", mik_src_path, PKG_NAME)
		preprocessed_txt := compiler_util.Preprocess(&yoink_str, &pth)
		checkErr(os.WriteFile(fmt.Sprintf("%spkg/%s/main_%s.milk", mik_src_path, PKG_NAME, PKG_NAME), []byte(*preprocessed_txt), 0755))
		exst, err := os.ReadFile(req_satisfied_loc)

		var exst2 string = ""
		checkErr(err)
		if string(exst) == "" {
			exst2 += fmt.Sprintf("%s:::%s", *url, PKG_NAME)
		} else {
			exst2 += fmt.Sprintf("%s\n%s:::%s", string(existing_files), *url, PKG_NAME)
		}

		checkErr(os.WriteFile(req_satisfied_loc, []byte(exst2), os.ModeAppend))
		compiler_util.NewSuccess(fmt.Sprintf("Sucessfully added package: %s. You can now use it with '#yoink-src<%s>", PKG_NAME, PKG_NAME), "", "", false)
	}
	return true
}

// install installs to mik-src via. a github link
func InstallGit(url *string) {
	// clear git cash
	checkErr(os.RemoveAll(fmt.Sprintf("%sgit/", mik_src_path)))
	checkErr(os.Mkdir(fmt.Sprintf("%sgit/", mik_src_path), 0755))

	// clone the actual repo
	clone_link := exec.Command("git", "clone", *url, mik_src_path+"git")
	checkErr(clone_link.Run())

	// call Addpkg with git
	pth := mik_src_path + "git"
	AddPkg(&pth, url)
	checkErr(os.RemoveAll(mik_src_path + "git"))
	checkErr(os.Mkdir(mik_src_path+"git", 0755))
	compiler_util.NewSuccess("Succesfully downloaded", "", "", false)
}

// remove removes a pkg from mik-src
func RemovePkg(pkg_name *string) {
	if *pkg_name == "." {
		compiler_util.NewWarning("Do you want to remove all packages? (Y/N)", "", "", false)
		reader := bufio.NewReader(os.Stdin)

		answer, _, err := reader.ReadLine()
		if err != nil {
			panic(err)
		}

		if string(answer) == "Y" || string(answer) == "y" {
			compiler_util.NewInfo("Deleting all packages...", "", "", false)
			// delete all pkgs
		} else {
			compiler_util.NewInfo("Aborted.", "", "", false)
		}
	} else {
		// check if pkg is in req_satisfied.conf
		cntnt, err := os.ReadFile(req_satisfied_loc)
		checkErr(err)

		var pkg_exists bool = false
		var new_cntnt string = ""

		for _, val := range strings.Split(string(cntnt), "\n") {
			if val == "" {
				continue
			}
			if *pkg_name == strings.Split(string(val), ":::")[1] {
				pkg_exists = true
				new_cntnt = strings.ReplaceAll(string(cntnt), *pkg_name+":::"+strings.Split(string(val), ":::")[1], "")
				break
			}
		}

		if pkg_exists {
			checkErr(os.RemoveAll(mik_src_path + "pkg/" + *pkg_name))
			checkErr(os.WriteFile(req_satisfied_loc, []byte(new_cntnt), 0755))
			compiler_util.NewSuccess("Succesfully removed package: "+*pkg_name, "", "", true)
		} else {
			compiler_util.NewCritical("The package you tried to remove was not found", "Try to use -list to see all installed packages", "", true)
		}
	}
}
