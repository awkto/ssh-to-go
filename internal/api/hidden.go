package api

// refreshHidden re-publishes a host's incognito set to the hub.
//
// The hub filters listings by session NAME (see hub.visibleLocked), not by a
// registry lookup, so the set goes stale the moment a name changes hands:
// rename an incognito session and the hub keeps hiding a name that no longer
// exists, while the live session — same session, new name — renders for
// everyone until the process restarts.
//
// So every path that adds, renames or drops a registry entry has to call this.
// It is a map rebuild over one host's entries; cheap enough to call
// unconditionally rather than trying to work out whether the set can have
// changed.
func (h *Handlers) refreshHidden(hostName string) {
	if h.Registry == nil || h.Hub == nil {
		return
	}
	h.Hub.SetHidden(hostName, h.Registry.HiddenNames(hostName))
}
