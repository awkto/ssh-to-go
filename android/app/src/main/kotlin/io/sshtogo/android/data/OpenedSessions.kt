package io.sshtogo.android.data

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.setValue

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

    /**
     * Bumped on every change. Reading it inside a Composable subscribes that
     * Composable to changes here, so dashboard "open" indicators recompose when
     * a session is opened or closed — even though [byHost] itself isn't a
     * snapshot-state collection.
     */
    private var revision by mutableIntStateOf(0)

    /** Record [session] on [host] as opened, appending in first-opened order. Idempotent. */
    @Synchronized
    fun markOpened(host: String, session: String) {
        val list = byHost.getOrPut(host) { mutableListOf() }
        if (!list.contains(session)) {
            list.add(session)
            revision++
        }
    }

    /** Snapshot of the sessions opened on [host] this run, in first-opened order. */
    @Synchronized
    fun forHost(host: String): List<String> {
        revision // subscribe Compose readers
        return byHost[host]?.toList() ?: emptyList()
    }

    /** Whether [session] on [host] is currently open in the app (reactive in Compose). */
    @Synchronized
    fun isOpen(host: String, session: String): Boolean {
        revision // subscribe Compose readers
        return byHost[host]?.contains(session) == true
    }

    /**
     * Drop [session] from [host]'s opened set — i.e. "close" it in the app so it
     * leaves the swipe carousel. The tmux session itself is untouched on the
     * server; re-opening it from the dashboard adds it back.
     */
    @Synchronized
    fun close(host: String, session: String) {
        if (byHost[host]?.remove(session) == true) revision++
    }
}
