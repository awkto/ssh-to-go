package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/awkto/ssh-to-go/internal/sshutil"
)

// Image paste. A terminal only carries text, so a screenshot pasted in the
// browser can't reach a TUI like Claude Code through the pty, and the
// program can't read the clipboard itself: it runs on the host, whose
// clipboard (if it has one) isn't the user's. Instead the browser uploads
// the image here, the server writes it to a file on the host, and the
// browser pastes the file's path into the terminal as ordinary text.
// Claude Code, and most tools that take images, attach an image given its
// path. The server never touches the session itself.

const (
	// maxPasteImageBytes caps one upload. Screenshots are well under this
	// even on a 4K display.
	maxPasteImageBytes = 20 << 20
	// pasteTTLMinutes is how long a pasted image stays on the host. The
	// tool reading the path does so within seconds; a day leaves room to
	// scroll back and reuse one.
	pasteTTLMinutes = 24 * 60
)

// pasteImageExt maps the sniffed content type to a file extension. Only
// image types the common TUIs accept are let through.
var pasteImageExt = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// pasteImageScript writes stdin to a new file under ~/.ssh-to-go/paste on
// the host, prunes pastes older than the TTL, and prints the file's
// absolute path. name is generated server-side and contains only
// [0-9a-z.-], so it needs no quoting beyond the double quotes here.
func pasteImageScript(name string) string {
	return strings.Join([]string{
		"umask 077",
		`D="$HOME/.ssh-to-go/paste"`,
		`mkdir -p "$D"`,
		fmt.Sprintf(`find "$D" -type f -mmin +%d -delete 2>/dev/null`, pasteTTLMinutes),
		fmt.Sprintf(`cat > "$D/%s" && printf '%%s' "$D/%s"`, name, name),
	}, "\n")
}

// pasteImageName is a sortable, collision-free file name for one paste.
func pasteImageName(ext string, now time.Time) string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%s-%s.%s", now.Format("20060102-150405"), hex.EncodeToString(b), ext)
}

// PasteImage stores the request body (one image) on the host and returns
// its path: POST /api/hosts/{host}/paste-image -> {"path": "/home/u/..."}.
// The type is sniffed from the bytes, not taken from the request.
func (h *Handlers) PasteImage(w http.ResponseWriter, r *http.Request) {
	hostCfg, ok := h.Hub.GetHostConfig(r.PathValue("host"))
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxPasteImageBytes))
	if err != nil {
		http.Error(w, fmt.Sprintf("image too large (max %d MB)", maxPasteImageBytes>>20), http.StatusRequestEntityTooLarge)
		return
	}
	ext, ok := pasteImageExt[http.DetectContentType(body)]
	if !ok {
		http.Error(w, "not a PNG, JPEG, GIF or WebP image", http.StatusUnsupportedMediaType)
		return
	}

	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	out, err := sshutil.ExecInput(client, pasteImageScript(pasteImageName(ext, time.Now())), body)
	path := strings.TrimSpace(out)
	if err != nil || !strings.HasPrefix(path, "/") {
		http.Error(w, fmt.Sprintf("saving image on host failed: %v", err), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]string{"path": path})
}
