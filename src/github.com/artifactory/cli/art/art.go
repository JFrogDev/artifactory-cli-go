package main

import (
  "os"
  "strings"
  "regexp"
  "strconv"
  "github.com/codegangsta/cli"
  "encoding/json"
)

var LocalPath string

var DryRun bool
var Url string
var User string
var Password string
var TargetPath string
var Flat bool

func main() {
    app := cli.NewApp()
    app.Name = "art"
    app.Usage = "Artifactory CLI"

    app.Commands = []cli.Command{
        {
            Name: "upload",
            Flags: GetUploadFlags(),
            Aliases: []string{"u"},
            Usage: "upload <local path> <repo name:repo path>",
            Action: func(c *cli.Context) {
                Upload(c)
            },
        },
        {
            Name: "download",
            Flags: GetDownloadFlags(),
            Aliases: []string{"d"},
            Usage: "download <repo path>",
            Action: func(c *cli.Context) {
                Download(c)
            },
        },
    }

    app.Run(os.Args)
}

func GetFlags() []cli.Flag {
    return []cli.Flag{
        cli.StringFlag{
         Name:  "url",
         Usage: "Artifactory URL",
        },
        cli.StringFlag{
         Name:  "user",
         Usage: "Artifactory user",
        },
        cli.StringFlag{
         Name:  "password",
         Usage: "Artifactory password",
        },
    }
}

func GetUploadFlags() []cli.Flag {
    flags := []cli.Flag{
        nil,nil,nil,nil,
    }
    copy(flags[0:3], GetFlags())
    flags[3] = cli.BoolFlag{
         Name:  "dry-run",
         Usage: "Set to true to disable communication with Artifactory",
    }
    return flags
}

func GetDownloadFlags() []cli.Flag {
    flags := []cli.Flag{
        nil,nil,nil,nil,
    }
    copy(flags[0:3], GetFlags())
    flags[3] = cli.BoolFlag{
        Name:  "flat",
        Usage: "Set to true if you do not wish to have the Artifactory repository path structure created locally for your downloaded files",
    }
    return flags
}

func InitFlags(c *cli.Context) {
    Url = GetMandatoryFlag(c, "url")
    if !strings.HasSuffix(Url, "/") {
        Url += "/"
    }

    User = c.String("user")
    Password = c.String("password")
    DryRun = c.Bool("dry-run")
    Flat = c.Bool("flat")
}

func GetFilesToUpload() []Artifact {
    rootPath := GetRootPath(LocalPath)
    if !IsPathExists(rootPath) {
        Exit("Path does not exist: " + rootPath)
    }
    artifacts := []Artifact{}
    if !IsDir(rootPath) {
        artifacts = append(artifacts, Artifact{rootPath, TargetPath})
        return artifacts
    }
    r, err := regexp.Compile(LocalPath)
    CheckError(err)

    paths := ListFiles(rootPath)
    for _, path := range paths {
        groups := r.FindStringSubmatch(path)
        size := len(groups)
        target := TargetPath
        for i := 1; i < size; i++ {
            target = strings.Replace(target, "$" + strconv.Itoa(i), groups[i], -1)
        }
        if ( size > 0) {
            artifacts = append(artifacts, Artifact{path, target})
        }
    }
    return artifacts
}

func Download(c *cli.Context) {
    InitFlags(c)
    size := len(c.Args())
    if size != 1 {
        Exit("Wrong number of arguments")
    }

    CheckAndGetRepoPathFromArg(c.Args()[0])
    repo := strings.Split(c.Args()[0], ":")[0]
    url := Url + "api/search/pattern?pattern=" + c.Args()[0]
    json := SendGet(url, User, Password)
    files := ParsePatternSearchResponse(json)
    for _, file := range files {
        downloadPath := Url + repo + "/" + file
        DownloadFile(downloadPath, file, Flat)
    }
}

func ParsePatternSearchResponse(resp []byte) []string {
    var f Files
    err := json.Unmarshal(resp, &f)
    CheckError(err)
    return f.Files
}

func Upload(c *cli.Context) {
    InitFlags(c)
    size := len(c.Args())
    if size != 2 {
        Exit("Wrong number of arguments")
    }
    LocalPath = c.Args()[0]
    TargetPath = CheckAndGetRepoPathFromArg(c.Args()[1])
    artifacts := GetFilesToUpload()

    for _, artifact := range artifacts {
        target := Url + artifact.targetPath
        PutFile(artifact.localPath, target, User, Password, DryRun)
    }
}

// Get a CLI flagg. If the flag does not exist, exit with a message.
func GetMandatoryFlag(c *cli.Context, flag string) string {
    value := c.String(flag)
    if value == "" {
        Exit("The --" + flag + " flag is mandatory")
    }
    return value
}

// Get the local root path, from which to start collecting artifacts to be uploaded to Artifactory.
func GetRootPath(path string) string {
    index := strings.Index(path, "(")
    if index == -1 {
        return path
    }
    return path[0:index]
}

func CheckAndGetRepoPathFromArg(arg string) string {
    if strings.Index(arg, ":") == -1 {
        Exit("Invalid repo path format: '" + arg + "'. Should be [repo:path].")
    }
    return strings.Replace(arg, ":", "/", -1)
}

type Artifact struct {
    localPath string
    targetPath string
}

type Files struct {
    Files []string
}