package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// Image paste for native SSH sessions. A program in a remote tmux pane
// (Claude Code, say) reads the clipboard of the host it runs on, not the
// one on this desktop, so pasting a screenshot there finds nothing. This
// uploads the image to the session's host and puts the saved file's path
// on the local clipboard instead: pasting it into the terminal hands the
// app a path, which it attaches as an image. Same mechanism as the web
// terminal's image paste, done by hand.

const pasteImageUsage = `usage: stogo paste-image <session> [file]
  uploads the clipboard image (or file) to the session's host, then puts
  the saved path on the clipboard — paste it into the session`

func cmdPasteImage(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("%s", pasteImageUsage)
	}
	var img []byte
	var err error
	if len(args) == 2 {
		img, err = os.ReadFile(args[1])
	} else {
		img, err = clipboardImage()
	}
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)
	host, _, err := c.resolveSession(args[0])
	if err != nil {
		return err
	}
	path, err := uploadImage(c, host, img)
	if err != nil {
		return err
	}

	if err := clipboardSetText(path + " "); err != nil {
		fmt.Println(path)
		fmt.Fprintf(os.Stderr, "stogo: couldn't put the path on the clipboard (%v); paste it by hand\n", err)
		return nil
	}
	fmt.Printf("%s\non the clipboard — paste it into the session\n", path)
	return nil
}

// uploadImage stores img on host and returns the path it was saved at.
func uploadImage(c *apiClient, host string, img []byte) (string, error) {
	var resp struct {
		Path string `json:"path"`
	}
	err := c.do("POST", "/api/hosts/"+url.PathEscape(host)+"/paste-image", img, &resp)
	return resp.Path, err
}

// clipboardTool is the program stogo reads the desktop clipboard with:
// wl-paste on Wayland, xclip on X11, or "" when there is neither.
func clipboardTool() string {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if _, err := exec.LookPath("wl-paste"); err == nil {
			return "wl-paste"
		}
	}
	if os.Getenv("DISPLAY") != "" {
		if _, err := exec.LookPath("xclip"); err == nil {
			return "xclip"
		}
	}
	return ""
}

// clipboardImage reads an image from the desktop clipboard with wl-paste
// (Wayland) or xclip (X11), whichever the session has.
func clipboardImage() ([]byte, error) {
	switch clipboardTool() {
	case "wl-paste":
		types, _ := exec.Command("wl-paste", "--list-types").Output()
		if t := imageType(string(types)); t != "" {
			return exec.Command("wl-paste", "--no-newline", "--type", t).Output()
		}
	case "xclip":
		types, _ := exec.Command("xclip", "-selection", "clipboard", "-t", "TARGETS", "-o").Output()
		if t := imageType(string(types)); t != "" {
			return exec.Command("xclip", "-selection", "clipboard", "-t", t, "-o").Output()
		}
	default:
		return nil, fmt.Errorf("can't read the clipboard: install wl-clipboard (Wayland) or xclip (X11), or pass an image file")
	}
	return nil, fmt.Errorf("no image on the clipboard")
}

// imageType picks the first image MIME type from a newline-separated list
// of clipboard targets, preferring PNG.
func imageType(list string) string {
	var first string
	for _, t := range strings.Fields(list) {
		if t == "image/png" {
			return t
		}
		if first == "" && strings.HasPrefix(t, "image/") {
			first = t
		}
	}
	return first
}

func clipboardSetText(s string) error {
	var cmd *exec.Cmd
	switch {
	case os.Getenv("WAYLAND_DISPLAY") != "":
		cmd = exec.Command("wl-copy")
	case os.Getenv("DISPLAY") != "":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("no desktop session")
	}
	cmd.Stdin = bytes.NewReader([]byte(s))
	return cmd.Run()
}
