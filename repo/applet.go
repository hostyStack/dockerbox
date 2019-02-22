package repo

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"golang.org/x/crypto/ssh/terminal"
)

const dockerExe = "docker"

type Applet struct {
	Name       string `yaml:"name"`
	WorkDir    string `yaml:"work_dir"`
	Entrypoint string `yaml:"entrypoint"`
	Restart    string `yaml:"restart"`
	Network    string `yaml:"network"`
	EnvFilter  string `yaml:"env_filter"`

	RM          bool `yaml:"rm"`
	TTY         bool `yaml:"tty"`
	Interactive bool `yaml:"interactive"`
	Privileged  bool `yaml:"privileged"`
	Detach      bool `yaml:"detach"`
	Kill        bool `yaml:"kill"`
	AllEnvs     bool `yaml:"all_envs"`

	Env          []string `yaml:"environment"`
	Volumes      []string `yaml:"volumes"`
	Ports        []string `yaml:"ports"`
	EnvFile      []string `yaml:"env_file"`
	Dependencies []string `yaml:"dependencies"`
	Links        []string `yaml:"links"`

	Image string `yaml:"image"`
	Tag   string `yaml:"image_tag"`

	Command []string `yaml:"command"`
}

func (a *Applet) Exec(extra ...string) error {
	cmd := a.RunCmd(extra)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error runnng applet %s: %v", a.Name, err)
	}
	return nil
}

func (a *Applet) PreExec() {
	cmd := a.KillCmd()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Printf("error killing %s: %v", a.Name, err)
	}
}

func isTTY() bool {
	return terminal.IsTerminal(int(os.Stdin.Fd()))
}

func (a *Applet) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawApplet Applet
	raw := rawApplet{
		RM:          true,
		Interactive: true,
		TTY:         true,
		Tag:         "latest",
	}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*a = Applet(raw)
	return nil
}

func (a *Applet) KillCmd() *exec.Cmd {
	args := []string{
		"kill",
		a.Name,
	}

	return exec.Command(dockerExe, args...)
}

func (a *Applet) RunCmd(extra []string) *exec.Cmd {
	args := []string{
		"run",
	}

	if a.Name != "" {
		args = append(args, "--name", a.Name)
	}
	if a.WorkDir != "" {
		args = append(args, "--workdir", os.ExpandEnv(a.WorkDir))
	}
	if a.Entrypoint != "" {
		args = append(args, "--entrypoint", a.Entrypoint)
	}
	if a.Restart != "" {
		args = append(args, "--restart", a.Restart)
	}
	if a.Network != "" {
		args = append(args, "--network", a.Network)
	}

	if a.RM {
		args = append(args, "--rm")
	}
	if a.Interactive {
		args = append(args, "--interactive")
	}
	if a.Privileged {
		args = append(args, "--privileged")
	}
	if a.Detach {
		args = append(args, "--detach")
	}
	if isTTY() && a.TTY {
		args = append(args, "--tty")
	}
	if a.AllEnvs {
		for _, f := range os.Environ() {
			if matched, _ := regexp.MatchString(a.EnvFilter, f); matched {
				args = append(args, "-e", f)
			}
		}
	}
	for _, f := range a.Env {
		args = append(args, "-e", os.ExpandEnv(f))
	}
	for _, f := range a.Volumes {
		args = append(args, "-v", os.ExpandEnv(f))
	}
	for _, f := range a.Ports {
		args = append(args, "-p", f)
	}
	for _, f := range a.EnvFile {
		args = append(args, "--env-file", f)
	}
	for _, f := range a.Links {
		args = append(args, "--link", f)
	}

	args = append(args, fmt.Sprintf("%s:%s", a.Image, a.Tag))

	if len(a.Command) != 0 && len(extra) == 0 {
		args = append(args, a.Command...)
	} else {
		args = append(args, extra...)
	}

	return exec.Command(dockerExe, args...)
}
