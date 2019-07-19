package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/buger/goterm"
	. "github.com/logrusorgru/aurora"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v3"
)

var version = "v1.0"

func main() {
	app := cli.NewApp()
	app.Name = "deploy"
	app.Usage = "deploy application"
	app.Version = version
	app.Commands = []cli.Command{
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "list all of services",
			Action: func(c *cli.Context) (err error) {
				return lsService()
			},
		},
		{
			Name:  "app",
			Usage: "deploy microservice application",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "service, s",
					Value: "",
					Usage: "service to deploy",
				},
				cli.StringFlag{
					Name:  "branch, b",
					Value: "",
					Usage: "code branch to deploy",
				},
				cli.StringFlag{
					Name:  "env, e",
					Value: "",
					Usage: "environment to deploy, available env is: [dev, test, stage, production]",
				},
			},
			Action: func(c *cli.Context) (err error) {
				return deployMicroService(c.String("service"), c.String("branch"), c.String("env"))
			},
		},
		{
			Name:  "web",
			Usage: "deploy web application",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "service, s",
					Value: "",
					Usage: "service to deploy",
				},
				cli.StringFlag{
					Name:  "branch, b",
					Value: "",
					Usage: "code branch to deploy",
				},
				cli.StringFlag{
					Name:  "env, e",
					Value: "",
					Usage: "environment to deploy, available env is: [dev, test, stage, production]",
				},
			},
			Action: func(c *cli.Context) (err error) {
				fmt.Println("ToDO......")
				return nil
			},
		},
		{
			Name:    "lsbranch",
			Aliases: []string{"lsb"},
			Usage:   "list the code branches of service",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "service, s",
					Value: "",
					Usage: "service to list branches",
				},
			},
			Action: func(c *cli.Context) (err error) {
				return lsBranch(c.String("service"))
			},
		},
		{
			Name:      "upload-cdn",
			Aliases:   []string{"upcdn"},
			Usage:     "upload file or directory to gcs bucket",
			ArgsUsage: "<src> <dst>",
			Flags: []cli.Flag{
				cli.BoolTFlag{
					Name:  "cache, c",
					Usage: "the file or directory in gcs bucket cache or not",
				},
			},
			Action: func(c *cli.Context) (err error) {
				if len(c.Args()) < 2 {
					cli.ShowCommandHelp(c, "upload-cdn")
					return cli.NewExitError("Expected at least 2 parameters", 1)
				}
				fmt.Printf("src:%s\ndst:%s\nisCache:%v\n", c.Args()[0:len(c.Args())-1], c.Args()[len(c.Args())-1], c.Bool("cache"))
				return nil
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		if _, ok := err.(*cli.ExitError); !ok {
			// Exit errors are already printed
			fmt.Println(err)
		}
	}
}

func readConfig(conf string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(conf)
	if err != nil {
		return nil, err
	}
	result := make(map[string]interface{})
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func lsBranch(service string) error {
	serviceConf, err := readConfig(fmt.Sprintf("%s/.dpcfg/service.yaml", os.Getenv("HOME")))
	if err != nil {
		return err
	}
	var gitUrl string
	for _, v := range serviceConf {
		vAssert, _ := v.(map[interface{}]interface{})
		item, ok := vAssert[service]
		if ok {
			vItem, _ := item.(map[string]interface{})
			gitUrl = fmt.Sprintf("%s", vItem["gitUrl"])
			break
		}
	}
	if "" == gitUrl {
		return errors.New(fmt.Sprintf("Error: No such service: %s", service))
	}
	gitLsRemote := fmt.Sprintf("git ls-remote --head %s | awk -F '/' '{print $NF}'", gitUrl)
	cmd := exec.Command("bash", "-c", gitLsRemote)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ServiceName", "Branches"})
	table.SetHeaderColor(tablewriter.Colors{tablewriter.FgGreenColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgGreenColor, tablewriter.Bold})
	table.SetAutoMergeCells(true)
	var data [][]string
	for _, branch := range strings.Fields(string(output)) {
		data = append(data, []string{service, branch})
	}
	table.AppendBulk(data)
	table.Render()
	return nil
}

func lsService() error {
	serviceConf, err := readConfig(fmt.Sprintf("%s/.dpcfg/service.yaml", os.Getenv("HOME")))
	if err != nil {
		return err
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ServiceType", "ServiceName"})
	table.SetHeaderColor(tablewriter.Colors{tablewriter.FgGreenColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgGreenColor, tablewriter.Bold})
	table.SetAutoMergeCells(true)
	var data [][]string
	for k, v := range serviceConf {
		vAssert, _ := v.(map[interface{}]interface{})
		for svc, _ := range vAssert {
			data = append(data, []string{k, fmt.Sprintf("%s", svc)})
		}
	}
	table.AppendBulk(data)
	table.Render()
	return nil
}

func deployMicroService(service, branch, env string) error {
	if env != "dev" && env != "test" && env != "stage" && env != "production" {
		return errors.New(fmt.Sprintf("Error: No such env: %s, the available env is [dev, test, stage, production]", env))
	}
	serviceConf, err := readConfig(fmt.Sprintf("%s/.dpcfg/service.yaml", os.Getenv("HOME")))
	if err != nil {
		return err
	}
	conf, err := readConfig(fmt.Sprintf("%s/.dpcfg/conf.yaml", os.Getenv("HOME")))
	if err != nil {
		return err
	}

	var gitUrl string
	var buildScriptPath string
	for _, v := range serviceConf {
		vAssert, _ := v.(map[interface{}]interface{})
		item, ok := vAssert[service]
		if ok {
			vItem, _ := item.(map[string]interface{})
			gitUrl = fmt.Sprintf("%s", vItem["gitUrl"])
			buildScriptPath = fmt.Sprintf("%s", vItem["buildScriptPath"])
			break
		}
	}
	if "" == gitUrl || "" == buildScriptPath {
		return errors.New(fmt.Sprintf("Error: No such service: %s", service))
	}

	stepOutput("Pull Code...", 1)
	tmpDir, err := ioutil.TempDir("/tmp", "deploy")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	gitClone := fmt.Sprintf("git clone -b %s %s %s", branch, gitUrl, tmpDir)
	err = RunCommand("bash", "-c", "-x", gitClone)
	if _, ok := err.(*exec.ExitError); ok { // there is error code
		return err
	}

	stepOutput("Build Package...", 2)
	mvnBuild := fmt.Sprintf("cd %s/%s && sh build.sh", tmpDir, buildScriptPath)
	fmt.Println(mvnBuild)
	err = RunCommand("bash", "-c", "-x", mvnBuild)
	if _, ok := err.(*exec.ExitError); ok {
		return err
	}

	stepOutput("Build Docker Image...", 3)
	_, err = CopyFile(fmt.Sprintf("%s/%s/Dockerfile", tmpDir, buildScriptPath),
		fmt.Sprintf("%s/.dpcfg/%s", os.Getenv("HOME"), conf["MicroServiceDockerfile"]))
	if err != nil {
		return err
	}

	dockerImageName := fmt.Sprintf("%s/%s", conf["DockerRepo"], service)
	timestamp := time.Now().UTC().Format("20060102-150405")
	gitRevParse := fmt.Sprintf("cd %s && git rev-parse --short HEAD", tmpDir)
	commitId, err := exec.Command("bash", "-c", gitRevParse).Output()
	deployType := "cli"
	dockerImageTag := fmt.Sprintf("%s-%s-%s-%s", timestamp, strings.TrimRight(string(commitId), "\n"), deployType, branch)
	packgeName := parsePackageName(fmt.Sprintf("%s/%s/pom.xml", tmpDir, buildScriptPath))

	dockerBuild := fmt.Sprintf("cd %s/%s && docker build --build-arg JAR_FILE=%s -t %s:%s .", tmpDir, buildScriptPath,
		packgeName, dockerImageName, dockerImageTag)
	fmt.Println(dockerBuild)
	err = RunCommand("bash", "-c", "-x", dockerBuild)
	if err != nil {
		return err
	}

	stepOutput("Push Docker Image...", 4)
	dockerLogin := fmt.Sprintf("docker login -u %s -p \"$(cat %s)\" %s", conf["DockerRepoUser"],
		fmt.Sprintf("%s/.dpcfg/%s", os.Getenv("HOME"), conf["DockerRepoPassFile"]), conf["DockerRepoUrl"])
	err = RunCommand("bash", "-c", dockerLogin)
	if err != nil {
		return err
	}
	dockerPush := fmt.Sprintf("docker push %s:%s", dockerImageName, dockerImageTag)
	err = RunCommand("bash", "-c", "-x", dockerPush)
	if err != nil {
		return err
	}
	defer RemoveDockerImage(fmt.Sprintf("%s:%s", dockerImageName, dockerImageTag))

	stepOutput("Deploy to Kubernetes...", 5)
	helmRepoAdd := fmt.Sprintf("helm repo add deploy %s", conf["HelmRepo"])
	err = RunCommand("bash", "-c", "-x", helmRepoAdd)
	if err != nil {
		return err
	}
	var appName string
	if env == "dev" {
		appName = fmt.Sprintf("%s-dev", service)
	} else {
		appName = service
	}
	helmInstall := fmt.Sprintf(`helm upgrade --wait --timeout 600 --install -f %s/default-%s.yaml -f %s/values/%s/values-%s.yaml --set 'image.tag=%s,image.repository=%s' --kubeconfig %s --kube-context %s --namespace %s %s deploy/microservice`,
		conf["HelmValuesPath"], env, conf["HelmValuesPath"], service, env, dockerImageTag, service, fmt.Sprintf("%s/.dpcfg/kube-config", os.Getenv("HOME")), conf[fmt.Sprintf("kubeCtx%s", env)], env, appName)
	err = RunCommand("bash", "-c", "-x", helmInstall)
	if err != nil {
		return err
	}
	finishedSuccessOutput("Finished seuccess!")
	return nil
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.Create(dstName)
	if err != nil {
		return
	}
	defer dst.Close()

	return io.Copy(dst, src)
}

type Project struct {
	XMLName    xml.Name `xml:"project"`
	ArtifactId string   `xml:"artifactId"`
	Packaging  string   `xml:"packaging"`
	Version    string   `xml:"version"`
}

//
// 解析 pom.xml 获取构建 jar 包名称
//
func parsePackageName(pom string) string {
	data, _ := ioutil.ReadFile(pom)
	project := Project{}
	xml.Unmarshal(data, &project)
	return fmt.Sprintf("%s-%s.%s", project.ArtifactId, project.Version, project.Packaging)
}

func RunCommand(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)

	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	for {
		tmp := make([]byte, 1024)
		_, err := stdout.Read(tmp)
		fmt.Print(string(tmp))
		if err != nil {
			break
		}
	}

	if err = cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func RemoveDockerImage(image string) error {
	dockerRmi := fmt.Sprintf("docker rmi %s", image)
	return RunCommand("bash", "-c", dockerRmi)
}

func stepOutput(step string, n int) {
	terminalWidth := goterm.Width()
	for i := 0; i < terminalWidth; i++ {
		fmt.Print(Green("="))
	}
	fmt.Print("\n")
	step = fmt.Sprintf("\tStep%d: [ %s ] ", n, step)
	if len(step) < terminalWidth {
		for i := 0; i < (terminalWidth - len(step)); i++ {
			step += "*"
		}
	} else {
		step = Resize(step, uint(terminalWidth))
	}
	fmt.Println(Green(step))
	for i := 0; i < terminalWidth; i++ {
		fmt.Print(Green("="))
	}
	fmt.Print("\n")
}

func finishedSuccessOutput(step string) {
	terminalWidth := goterm.Width()
	for i := 0; i < terminalWidth; i++ {
		fmt.Print(Green("="))
	}
	fmt.Print("\n")
	step = fmt.Sprintf("\t%s  ", step)
	if len(step) < terminalWidth {
		for i := 0; i < (terminalWidth - len(step)); i++ {
			step += "*"
		}
	} else {
		step = Resize(step, uint(terminalWidth))
	}
	fmt.Println(Green(step))
	for i := 0; i < terminalWidth; i++ {
		fmt.Print(Green("="))
	}
	fmt.Print("\n")
}

// PadRight returns a new string of a specified length in which the end of the current string is padded with spaces or with a specified Unicode character.
func PadRight(str string, length int, pad byte) string {
	if len(str) >= length {
		return str
	}
	buf := bytes.NewBufferString(str)
	for i := 0; i < length-len(str); i++ {
		buf.WriteByte(pad)
	}
	return buf.String()
}

// Resize resizes the string with the given length. It ellipses with '...' when the string's length exceeds
// the desired length or pads spaces to the right of the string when length is smaller than desired
func Resize(s string, length uint) string {
	n := int(length)
	if len(s) == n {
		return s
	}
	// Pads only when length of the string smaller than len needed
	s = PadRight(s, n, ' ')
	if len(s) > n {
		b := []byte(s)
		var buf bytes.Buffer
		for i := 0; i < n-3; i++ {
			buf.WriteByte(b[i])
		}
		buf.WriteString("...")
		s = buf.String()
	}
	return s
}
