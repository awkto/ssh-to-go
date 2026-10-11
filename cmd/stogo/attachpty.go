package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// Image paste for native terminals. A program in the remote pane (Claude
// Code, say) answers Ctrl+V by reading the clipboard of the host it runs
// on, which isn't this desktop's, so a screenshot never arrives. When this
// desktop has a clipboard stogo can read, the attach runs under a pty that
// stogo relays instead of being exec'd: a Ctrl+V with an image on the
// clipboard uploads the image to the session's host (the same endpoint the
// web terminal uses) and pastes the saved path instead, which the app
// attaches as an image. Any other Ctrl+V, and every other byte, passes
// through untouched. STOGO_IMAGE_PASTE=0 turns it off.

// imagePasteAvailable reports whether the attach should run under the
// relaying pty: both ends are terminals and there is a clipboard to read.
func imagePasteAvailable() bool {
	if os.Getenv("STOGO_IMAGE_PASTE") == "0" {
		return false
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	return clipboardTool() != ""
}

// runAttachWithImagePaste runs the handoff command under a pty, relaying
// the terminal both ways, and exits with the command's status.
func runAttachWithImagePaste(c *apiClient, host, command string) error {
	cmd := exec.Command("/bin/sh", "-c", command)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	defer ptmx.Close()

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	go func() {
		for range winch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	_ = pty.InheritSize(os.Stdin, ptmx)

	stdin := int(os.Stdin.Fd())
	old, err := term.MakeRaw(stdin)
	if err != nil {
		return err
	}

	outDone := make(chan struct{})
	go func() {
		io.Copy(os.Stdout, ptmx)
		close(outDone)
	}()
	go relayInput(os.Stdin, ptmx, func() (string, bool) {
		img, err := clipboardImage()
		if err != nil {
			return "", false
		}
		path, err := uploadImage(c, host, img)
		return path, err == nil
	})

	werr := cmd.Wait()
	select { // let the last of the output through
	case <-outDone:
	case <-time.After(time.Second):
	}
	term.Restore(stdin, old)

	var exitErr *exec.ExitError
	if errors.As(werr, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	return werr
}

// ctrlVKeys are the ways a terminal sends Ctrl+V: the control byte, and
// the CSI u / modifyOtherKeys encodings a terminal switches to when tmux
// asks for extended keys.
var ctrlVKeys = [][]byte{
	{0x16},
	[]byte("\x1b[118;5u"),
	[]byte("\x1b[27;5;118~"),
}

// findCtrlV returns the offset and length of the first Ctrl+V in b, or -1.
func findCtrlV(b []byte) (int, int) {
	at, n := -1, 0
	for _, k := range ctrlVKeys {
		if i := bytes.Index(b, k); i >= 0 && (at < 0 || i < at) {
			at, n = i, len(k)
		}
	}
	return at, n
}

// relayInput copies keystrokes from in to out. Each Ctrl+V asks
// pasteImage for an image path; if it gets one, the path goes out as a
// bracketed paste (the remote tmux strips the markers and re-adds them
// only if the pane's app enabled bracketed paste), and otherwise the key
// goes out as typed.
func relayInput(in io.Reader, out io.Writer, pasteImage func() (string, bool)) {
	buf := make([]byte, 4096)
	for {
		n, err := in.Read(buf)
		data := buf[:n]
		for len(data) > 0 {
			i, l := findCtrlV(data)
			if i < 0 {
				out.Write(data)
				break
			}
			out.Write(data[:i])
			if path, ok := pasteImage(); ok {
				out.Write([]byte("\x1b[200~" + path + " \x1b[201~"))
			} else {
				out.Write(data[i : i+l])
			}
			data = data[i+l:]
		}
		if err != nil {
			return
		}
	}
}
