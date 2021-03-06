package cc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jm33-m0/emp3r0r/emagent/internal/agent"
)

// Download download file using default http client
func Download(url, path string) (err error) {
	var (
		resp *http.Response
		data []byte
	)
	resp, err = http.Get(url)
	if err != nil {
		return
	}

	data, err = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return
	}

	return ioutil.WriteFile(path, data, 0600)
}

// SendCmd send command to agent
func SendCmd(cmd string, a *agent.SystemInfo) error {
	if a == nil {
		return errors.New("SendCmd: No such agent")
	}

	var cmdData agent.MsgTunData

	cmdData.Payload = fmt.Sprintf("cmd%s%s", agent.OpSep, cmd)
	cmdData.Tag = a.Tag

	return Send2Agent(&cmdData, a)
}

// IsCommandExist check if an executable is in $PATH
func IsCommandExist(exe string) bool {
	_, err := exec.LookPath(exe)
	return err == nil
}

// FileBaseName /path/to/foo -> foo
func FileBaseName(filepath string) (filename string) {
	// we only need the filename
	filepathSplit := strings.Split(filepath, "/")
	filename = filepathSplit[len(filepathSplit)-1]
	return
}

// VimEdit launch local vim to edit files
func VimEdit(filepath string) (err error) {
	if os.Getenv("TMUX") == "" ||
		!IsCommandExist("tmux") ||
		!IsCommandExist("vim") {

		return errors.New("You need to run emp3r0r under tmux, and make sure vim is installed")
	}

	// split tmux window, remember pane number
	vimjob := fmt.Sprintf("tmux split-window 'echo -n $TMUX_PANE>%svim.pane;vim %s'", Temp, filepath)
	cmd := exec.Command("/bin/sh", "-c", vimjob)
	err = cmd.Run()
	if err != nil {
		return
	}

	// index of our tmux pane
	for {
		if _, err = os.Stat(Temp + "vim.pane"); os.IsNotExist(err) {
			time.Sleep(200 * time.Millisecond)
		} else {
			break
		}
	}

	// remove vim.pane eventually
	defer func() {
		err = os.Remove(Temp + "vim.pane")
		if err != nil {
			CliPrintWarning(err.Error())
		}
	}()

	paneBytes, e := ioutil.ReadFile(Temp + "vim.pane")
	pane := string(paneBytes)
	if e != nil {
		return fmt.Errorf("cannot detect tmux pane number: %v", e)
	}

	// loop until vim exits
	for {
		time.Sleep(1 * time.Second)

		// check if our tmux pane exists, ie. the user hasn't done editing
		checkPaneCmd := exec.Command("tmux", "display-message", "-p", "-t", pane)
		out, err := checkPaneCmd.CombinedOutput()
		if err != nil {
			tmuxout := string(out)
			if strings.Contains(tmuxout, "can't find") {
				CliPrintSuccess("Vim has done editing")
				return nil
			}
			CliPrintError(err.Error())
			break
		}
	}

	return errors.New("don't know if vim has done editing")
}

// TmuxSplit split tmux window, and run command in the new pane
func TmuxSplit(hV, cmd string) error {
	if os.Getenv("TMUX") == "" ||
		!IsCommandExist("tmux") ||
		!IsCommandExist("less") {

		return errors.New("You need to run emp3r0r under `tmux`, and make sure `less` is installed")
	}

	job := fmt.Sprintf("tmux split-window -%s %s", hV, cmd)

	out, err := exec.Command("/bin/sh", "-c", job).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}

	return nil
}

// agentExists is agent already in target list?
func agentExists(agent *agent.SystemInfo) bool {
	for a := range Targets {
		if a.Tag == agent.Tag {
			return true
		}
	}

	return false
}

// TermClear clear screen
func TermClear() {
	os.Stdout.WriteString("\033[2J")
	err := CliBanner()
	if err != nil {
		CliPrintError("%v", err)
	}
}
