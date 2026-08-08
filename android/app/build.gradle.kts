plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
    id("org.jetbrains.kotlin.plugin.serialization")
}

android {
    namespace = "io.sshtogo.android"
    compileSdk = 34

    defaultConfig {
        applicationId = "io.sshtogo.android"
        minSdk = 26
        targetSdk = 34
        versionCode = 24
        versionName = project.findProperty("versionName")?.toString() ?: "0.7.0"
        vectorDrawables.useSupportLibrary = true
    }

    // Release signing keystore, supplied by CI (env) or a local gradle.properties.
    // Resolution order per field: env var first, then gradle property. When no
    // keystore path is provided the release build falls back to debug signing so
    // local `assembleRelease` keeps working without the secrets.
    val releaseStorePath = System.getenv("ANDROID_KEYSTORE_PATH")
        ?: (project.findProperty("releaseStoreFile") as String?)
    val hasReleaseKeystore = !releaseStorePath.isNullOrBlank()

    signingConfigs {
        create("release") {
            if (hasReleaseKeystore) {
                storeFile = file(releaseStorePath!!)
                storePassword = System.getenv("ANDROID_KEYSTORE_PASSWORD")
                    ?: project.findProperty("releaseStorePassword") as String?
                keyAlias = System.getenv("ANDROID_KEY_ALIAS")
                    ?: project.findProperty("releaseKeyAlias") as String?
                keyPassword = System.getenv("ANDROID_KEY_PASSWORD")
                    ?: project.findProperty("releaseKeyPassword") as String?
            }
        }
    }

    buildTypes {
        debug {
            isMinifyEnabled = false
            applicationIdSuffix = ".debug"
            versionNameSuffix = "-debug"
        }
        release {
            // Sign with the real release keystore when one is available
            // (CI with secrets set); otherwise fall back to the debug keystore
            // so local release builds and forks without secrets still produce
            // an installable APK.
            signingConfig = if (hasReleaseKeystore)
                signingConfigs.getByName("release")
            else
                signingConfigs.getByName("debug")
            isMinifyEnabled = false
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        compose = true
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
    }
}

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2024.09.03")
    implementation(composeBom)
    androidTestImplementation(composeBom)

    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.activity:activity-compose:1.9.2")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.6")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.8.6")
    implementation("androidx.navigation:navigation-compose:2.8.2")
    implementation("androidx.security:security-crypto:1.1.0-alpha06")

    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-extended")

    // Vendored terminal emulator + view (modified copy of Termux's libraries
    // with the local-PTY/JNI plumbing replaced by a remote WebSocket transport).
    implementation(project(":libraries:terminal-view"))

    // Networking
    implementation("com.squareup.retrofit2:retrofit:2.11.0")
    implementation("com.squareup.retrofit2:converter-kotlinx-serialization:2.11.0")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("com.squareup.okhttp3:logging-interceptor:4.12.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.9.0")

    debugImplementation("androidx.compose.ui:ui-tooling")
}
