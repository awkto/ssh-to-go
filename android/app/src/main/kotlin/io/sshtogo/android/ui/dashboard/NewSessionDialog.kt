package io.sshtogo.android.ui.dashboard

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.AssistChip
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import io.sshtogo.android.SshToGoApplication
import io.sshtogo.android.net.CreateSessionRequest

/**
 * The full New Session form, mirroring the web form and `stogo new`:
 * everything is available, nothing is required — a repeat create is
 * tap-+, tap-Create.
 *
 *  - Name: optional; blank uses the auto-name shown as the placeholder.
 *  - Directory: prefilled from the last-used value (persisted in
 *    AppPreferences), falling back to the server's new_session_dir.
 *    $name/$date are expanded SERVER-side; the preview line below the
 *    field makes the feature visible, exactly like the web form.
 *  - Command: optional; recent-command chips come from the server's
 *    shared list. Empty = plain shell.
 *  - Throwaway / incognito: off by default, reset every open (a sticky
 *    incognito default would defeat the point).
 *
 * On success the last-used directory/command/create-dir are persisted as
 * the next open's defaults (as typed — the $name template, not this
 * session's expansion) and [onCreated] fires with the server's sanitized
 * session name, which is the name to open the terminal with.
 */
@Composable
fun NewSessionDialog(
    hostName: String,
    recentCommands: List<String>,
    serverNewSessionDir: String,
    onCreate: (req: CreateSessionRequest, onResult: (error: String?, createdName: String?) -> Unit) -> Unit,
    onCreated: (sessionName: String) -> Unit,
    onDismiss: () -> Unit,
) {
    val prefs = SshToGoApplication.instance.prefs

    var name by remember { mutableStateOf("") }
    var dir by remember {
        mutableStateOf(prefs.newSessionDir.ifBlank { serverNewSessionDir.ifBlank { "~/sessions/" } })
    }
    var createDir by remember { mutableStateOf(prefs.newSessionCreateDir) }
    var command by remember { mutableStateOf(prefs.newSessionCommand) }
    var throwaway by remember { mutableStateOf(false) }
    var incognito by remember { mutableStateOf(false) }
    var creating by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    // Regenerated when throwaway toggles (tmp- vs session- prefix), same as
    // the web form's autoName.
    var autoName by remember { mutableStateOf(autoSessionName(false)) }

    // What $name will really expand to: the sanitized effective name.
    val effName = sanitizeSessionName(name).ifBlank { autoName }

    AlertDialog(
        onDismissRequest = { if (!creating) onDismiss() },
        title = { Text("New session on $hostName") },
        text = {
            Column(
                modifier = Modifier.verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it; error = null },
                    label = { Text("Name") },
                    placeholder = { Text(autoName) },
                    singleLine = true,
                    enabled = !creating,
                    modifier = Modifier.fillMaxWidth(),
                )

                OutlinedTextField(
                    value = dir,
                    onValueChange = { dir = it },
                    label = { Text("Directory") },
                    placeholder = { Text("~/") },
                    singleLine = true,
                    enabled = !creating,
                    modifier = Modifier.fillMaxWidth(),
                )
                // PREVIEW ONLY — the server does the real substitution. The
                // line exists because the feature is invisible otherwise: you
                // discover $name by watching it turn into the name you typed.
                val dirPreview = expandVarsPreview(dir, effName)
                if (dirPreview != dir.trim() && dirPreview.isNotBlank()) {
                    Text(
                        "→ $dirPreview",
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace,
                        color = MaterialTheme.colorScheme.outline,
                    )
                }
                FilterChip(
                    selected = createDir,
                    onClick = { createDir = !createDir },
                    label = { Text("Create directory if missing") },
                    enabled = !creating,
                )

                OutlinedTextField(
                    value = command,
                    onValueChange = { command = it },
                    label = { Text("Command (blank = shell)") },
                    placeholder = { Text("claude") },
                    singleLine = true,
                    enabled = !creating,
                    modifier = Modifier.fillMaxWidth(),
                )
                val cmdPreview = expandVarsPreview(command, effName)
                if (cmdPreview != command.trim() && cmdPreview.isNotBlank()) {
                    Text(
                        "→ $cmdPreview",
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace,
                        color = MaterialTheme.colorScheme.outline,
                    )
                }
                if (recentCommands.isNotEmpty()) {
                    Row(
                        modifier = Modifier.horizontalScroll(rememberScrollState()),
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        recentCommands.take(4).forEach { rc ->
                            AssistChip(
                                onClick = { command = rc },
                                label = { Text(rc, maxLines = 1) },
                                enabled = !creating,
                            )
                        }
                    }
                }

                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    FilterChip(
                        selected = throwaway,
                        onClick = {
                            throwaway = !throwaway
                            autoName = autoSessionName(throwaway)
                        },
                        label = { Text("throwaway") },
                        enabled = !creating,
                    )
                    FilterChip(
                        selected = incognito,
                        onClick = { incognito = !incognito },
                        label = { Text("incognito") },
                        enabled = !creating,
                    )
                }

                error?.let {
                    Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                }
            }
        },
        confirmButton = {
            TextButton(
                enabled = !creating,
                onClick = {
                    creating = true
                    error = null
                    val req = CreateSessionRequest(
                        name = name.trim().ifBlank { autoName },
                        cwd = dir.trim(),
                        createDir = createDir,
                        command = command.trim(),
                        throwaway = throwaway,
                        incognito = incognito,
                    )
                    onCreate(req) { err, createdName ->
                        creating = false
                        if (err == null && createdName != null) {
                            // Persist the form AS TYPED as next time's
                            // defaults ($name template, not the expansion).
                            prefs.newSessionDir = dir.trim()
                            prefs.newSessionCommand = command.trim()
                            prefs.newSessionCreateDir = createDir
                            onCreated(createdName)
                        } else {
                            error = err
                        }
                    }
                },
            ) { Text(if (creating) "Creating…" else "Create") }
        },
        dismissButton = {
            TextButton(enabled = !creating, onClick = onDismiss) { Text("Cancel") }
        },
    )
}

/** Web nsAutoName: 'session-'/'tmp-' + 4 random base36 chars. */
private fun autoSessionName(throwaway: Boolean): String {
    val alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
    val suffix = (1..4).map { alphabet.random() }.joinToString("")
    return (if (throwaway) "tmp-" else "session-") + suffix
}

/** Mirrors the server's sanitizeSessionName enough for the preview: trim, whitespace runs -> '-'. */
private fun sanitizeSessionName(s: String): String = s.trim().replace(Regex("[ \t]+"), "-")

// $name / $date preview — a deliberately small mirror of
// internal/sessionvars (Go), same as the web form keeps its own. PREVIEW
// ONLY: the server does the substitution that creates the session. Keep the
// rules in step: same two variables, same word boundary ($nameless is not
// $name), ${name} form supported, and every OTHER $-form left alone for the
// remote shell.
private val NS_VARS = listOf("name", "date")

private fun expandVarsPreview(s: String, sessionName: String): String {
    val src = s.trim()
    if ('$' !in src) return src
    fun valueOf(v: String): String =
        if (v == "name") sessionName else java.time.LocalDate.now().toString()
    fun isIdent(c: Char) = c.isLetterOrDigit() || c == '_'
    val out = StringBuilder(src.length)
    var i = 0
    while (i < src.length) {
        val c = src[i]
        if (c == '$' && i + 1 < src.length) {
            if (src[i + 1] == '{') {
                val end = src.indexOf('}', i + 2)
                if (end > 0 && src.substring(i + 2, end) in NS_VARS) {
                    out.append(valueOf(src.substring(i + 2, end)))
                    i = end + 1
                    continue
                }
            } else {
                val v = NS_VARS.firstOrNull { name ->
                    src.startsWith(name, i + 1) &&
                        (i + 1 + name.length >= src.length || !isIdent(src[i + 1 + name.length]))
                }
                if (v != null) {
                    out.append(valueOf(v))
                    i += 1 + v.length
                    continue
                }
            }
        }
        out.append(c)
        i++
    }
    return out.toString()
}

