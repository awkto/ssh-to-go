package io.sshtogo.android.data

/**
 * In-memory registry of the tmux sessions the user has actually opened during
 * the current app run, recorded in first-opened order, keyed by host.
 *
 * The terminal swipe carousel is built from this instead of the host's full
 * session list: swiping then only moves between sessions we've deliberately
 * opened, rather than connecting through every session that happens to exist
 * on the host just to step past it.
 *
 * Deliberately NOT persisted — it models the live working set for this run and
 * resets when the app process is killed.
 */
class OpenedSessions {
    private val byHost = mutableMapOf<String, MutableList<String>>()

    /** Record [session] on [host] as opened, appending in first-opened order. Idempotent. */
    @Synchronized
    fun markOpened(host: String, session: String) {
        val list = byHost.getOrPut(host) { mutableListOf() }
        if (!list.contains(session)) list.add(session)
    }

    /** Snapshot of the sessions opened on [host] this run, in first-opened order. */
    @Synchronized
    fun forHost(host: String): List<String> = byHost[host]?.toList() ?: emptyList()

    /**
     * Drop [session] from [host]'s opened set — i.e. "close" it in the app so it
     * leaves the swipe carousel. The tmux session itself is untouched on the
     * server; re-opening it from the dashboard adds it back.
     */
    @Synchronized
    fun close(host: String, session: String) {
        byHost[host]?.remove(session)
    }
}
