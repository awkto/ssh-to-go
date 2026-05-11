package io.sshtogo.android

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import io.sshtogo.android.ui.SshToGoApp
import io.sshtogo.android.ui.theme.SshToGoTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            SshToGoTheme {
                SshToGoApp()
            }
        }
    }
}
