package io.sshtogo.android

import android.content.Context
import android.content.pm.PackageManager

/**
 * Resolves the running app's versionName via PackageManager. Falls back to
 * "?" if for some reason the package isn't found (shouldn't happen).
 */
fun Context.appVersionName(): String = try {
    packageManager.getPackageInfo(packageName, 0).versionName ?: "?"
} catch (_: PackageManager.NameNotFoundException) {
    "?"
}
