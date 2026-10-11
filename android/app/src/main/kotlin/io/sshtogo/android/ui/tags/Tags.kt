package io.sshtogo.android.ui.tags

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Checkbox
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlin.math.cos
import kotlin.math.pow
import kotlin.math.sin

/**
 * Session tags, mirroring the web Sessions page: free-form labels stored in
 * each session's icon entry, a shared colour per tag (an OKLCH hue from
 * /api/tag-colors), and an "all / one tag / untagged" filter.
 */

/** Filter sentinel for "sessions with no tags"; a real tag never starts with ':'. */
const val UNTAGGED = ":untagged"

/** Swatches for tag colours. Mirrors keystore.TagHues on the server. */
val TAG_HUES = listOf(25, 55, 75, 95, 125, 150, 180, 210, 245, 275, 305, 345)

/** Same rules as keystore.NormalizeTags: lowercase, trimmed, inner whitespace to '-', max 32. */
fun normalizeTag(raw: String): String =
    raw.lowercase().trim().split(Regex("\\s+")).filter { it.isNotEmpty() }
        .joinToString("-").take(32)

fun normalizeTags(raw: Collection<String>): List<String> =
    raw.map(::normalizeTag).filter { it.isNotEmpty() }.distinct().sorted()

/**
 * A tag's hue: the stored one, else the web UI's name hash so tags from
 * before colours were stored look the same on both clients.
 */
fun tagHue(tag: String, colors: Map<String, Int>): Int {
    colors[tag]?.let { return it }
    var h = 0L
    for (c in tag) h = (h * 31 + c.code) and 0xFFFFFFFFL
    return (h % 360).toInt()
}

/** OKLCH -> sRGB, gamut-clipped. Enough precision for chips and dots. */
fun oklch(l: Double, c: Double, hueDeg: Int, alpha: Float = 1f): Color {
    val hr = Math.toRadians(hueDeg.toDouble())
    val a = c * cos(hr)
    val b = c * sin(hr)
    val l_ = l + 0.3963377774 * a + 0.2158037573 * b
    val m_ = l - 0.1055613458 * a - 0.0638541728 * b
    val s_ = l - 0.0894841775 * a - 1.2914855480 * b
    val ll = l_ * l_ * l_
    val mm = m_ * m_ * m_
    val ss = s_ * s_ * s_
    fun enc(x: Double): Float {
        val v = x.coerceIn(0.0, 1.0)
        return (if (v <= 0.0031308) 12.92 * v else 1.055 * v.pow(1 / 2.4) - 0.055).toFloat()
    }
    return Color(
        red = enc(4.0767416621 * ll - 3.3077115913 * mm + 0.2309699292 * ss),
        green = enc(-1.2684380046 * ll + 2.6097574011 * mm - 0.3413193965 * ss),
        blue = enc(-0.0041960863 * ll - 0.7034186147 * mm + 1.7076147010 * ss),
        alpha = alpha,
    )
}

/** The solid swatch colour for a hue (dots, swatches). */
fun tagDotColor(hue: Int): Color = oklch(0.7, 0.14, hue)

@Composable
private fun isDark(): Boolean = MaterialTheme.colorScheme.surface.luminance() < 0.5f

@Composable
fun TagDot(hue: Int, size: Int = 8) {
    Box(Modifier.size(size.dp).clip(CircleShape).background(tagDotColor(hue)))
}

/** Small pill showing a tag in its colour. */
@Composable
fun TagChip(tag: String, hue: Int, modifier: Modifier = Modifier) {
    val dark = isDark()
    val fg = if (dark) oklch(0.78, 0.12, hue) else oklch(0.45, 0.14, hue)
    val bg = oklch(0.65, 0.12, hue, alpha = if (dark) 0.18f else 0.16f)
    Text(
        tag,
        color = fg,
        fontSize = 11.sp,
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
        modifier = modifier
            .clip(RoundedCornerShape(50))
            .background(bg)
            .padding(horizontal = 7.dp, vertical = 1.dp),
    )
}

@Composable
fun TagChips(tags: List<String>, colors: Map<String, Int>, modifier: Modifier = Modifier) {
    if (tags.isEmpty()) return
    Row(modifier = modifier, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
        tags.forEach { TagChip(it, tagHue(it, colors)) }
    }
}

/**
 * Horizontal row of filter chips: All, each tag (with its count), Untagged.
 * [selected] is "" for all, [UNTAGGED], or a tag.
 */
@Composable
fun TagFilterRow(
    counts: List<Pair<String, Int>>,
    untaggedCount: Int,
    colors: Map<String, Int>,
    selected: String,
    onSelect: (String) -> Unit,
) {
    Row(
        modifier = Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        FilterChip(selected = selected.isEmpty(), onClick = { onSelect("") }, label = { Text("All") })
        counts.forEach { (tag, n) ->
            FilterChip(
                selected = selected == tag,
                onClick = { onSelect(if (selected == tag) "" else tag) },
                leadingIcon = { TagDot(tagHue(tag, colors)) },
                label = { Text("$tag  $n") },
            )
        }
        FilterChip(
            selected = selected == UNTAGGED,
            onClick = { onSelect(if (selected == UNTAGGED) "" else UNTAGGED) },
            label = { Text("Untagged  $untaggedCount") },
        )
    }
}

/**
 * Add/remove a session's tags. Toggles are local until Save; tapping a
 * tag's colour dot opens the swatches, and a colour pick applies at once
 * (it recolours the tag everywhere, not just on this session).
 */
@Composable
fun TagEditorDialog(
    sessionName: String,
    current: List<String>,
    allTags: List<String>,
    colors: Map<String, Int>,
    onSave: (List<String>) -> Unit,
    onSetColor: (tag: String, hue: Int) -> Unit,
    onDismiss: () -> Unit,
) {
    var chosen by remember { mutableStateOf(current.toSet()) }
    var known by remember { mutableStateOf((allTags + current).distinct().sorted()) }
    var draft by remember { mutableStateOf("") }
    var coloring by remember { mutableStateOf<String?>(null) }

    fun addDraft() {
        val t = normalizeTag(draft)
        draft = ""
        if (t.isEmpty()) return
        chosen = chosen + t
        if (t !in known) known = (known + t).sorted()
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Tags for $sessionName", maxLines = 1, overflow = TextOverflow.Ellipsis) },
        text = {
            Column(Modifier.heightIn(max = 420.dp).verticalScroll(rememberScrollState())) {
                OutlinedTextField(
                    value = draft,
                    onValueChange = { draft = it.take(32) },
                    label = { Text("New tag") },
                    placeholder = { Text("work, personal…") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { addDraft() }),
                    trailingIcon = {
                        TextButton(onClick = { addDraft() }, enabled = draft.isNotBlank()) { Text("Add") }
                    },
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.size(8.dp))
                if (known.isEmpty()) {
                    Text(
                        "No tags yet. Try \"work\" or \"personal\".",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.outline,
                    )
                }
                known.forEach { tag ->
                    val hue = tagHue(tag, colors)
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { chosen = if (tag in chosen) chosen - tag else chosen + tag }
                            .padding(vertical = 2.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Checkbox(
                            checked = tag in chosen,
                            onCheckedChange = { chosen = if (it) chosen + tag else chosen - tag },
                        )
                        Box(Modifier.weight(1f)) { TagChip(tag, hue) }
                        // Colour dot: opens the swatches for this tag.
                        Box(
                            Modifier
                                .size(40.dp)
                                .clickable { coloring = if (coloring == tag) null else tag },
                            contentAlignment = Alignment.Center,
                        ) {
                            Box(
                                Modifier
                                    .size(18.dp)
                                    .clip(CircleShape)
                                    .background(tagDotColor(hue))
                                    .then(
                                        if (coloring == tag)
                                            Modifier.border(2.dp, MaterialTheme.colorScheme.onSurface, CircleShape)
                                        else Modifier
                                    ),
                            )
                        }
                    }
                    if (coloring == tag) {
                        TAG_HUES.chunked(6).forEach { rowHues ->
                            Row(
                                Modifier.fillMaxWidth().padding(start = 48.dp, bottom = 6.dp),
                                horizontalArrangement = Arrangement.spacedBy(8.dp),
                            ) {
                                rowHues.forEach { h ->
                                    Box(
                                        Modifier
                                            .size(26.dp)
                                            .clip(CircleShape)
                                            .background(tagDotColor(h))
                                            .then(
                                                if (h == hue)
                                                    Modifier.border(2.dp, MaterialTheme.colorScheme.onSurface, CircleShape)
                                                else Modifier
                                            )
                                            .clickable { onSetColor(tag, h) },
                                    )
                                }
                            }
                        }
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                // A typed-but-not-added tag counts: Save is the obvious tap.
                val pending = normalizeTag(draft)
                onSave(normalizeTags(if (pending.isEmpty()) chosen else chosen + pending))
            }) { Text("Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
