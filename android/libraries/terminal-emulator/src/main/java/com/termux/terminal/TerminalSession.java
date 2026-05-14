package com.termux.terminal;

import android.annotation.SuppressLint;
import android.os.Handler;
import android.os.Message;

import java.nio.charset.StandardCharsets;
import java.util.UUID;

/**
 * A terminal session.
 *
 * Upstream Termux uses this class to drive a local PTY via a forked shell
 * subprocess. ssh-to-go always talks to a remote tmux over WebSocket, so
 * the JNI/PTY plumbing has been stripped out. Subclasses are expected to
 * provide their own transport by overriding {@link #initializeEmulator}
 * and {@link #write(byte[], int, int)} — see RelayTerminalSession.
 *
 * Bytes received from the transport should be pushed to
 * {@link #emulatorAppend(byte[], int)} on the main thread (or via a
 * MainThreadHandler post) so the emulator can parse them.
 */
public class TerminalSession extends TerminalOutput {

    private static final int MSG_NEW_INPUT = 1;
    protected static final int MSG_PROCESS_EXITED = 4;

    public final String mHandle = UUID.randomUUID().toString();

    protected TerminalEmulator mEmulator;

    /** Process-to-terminal queue, retained so legacy subclasses can push bytes. */
    protected final ByteQueue mProcessToTerminalIOQueue = new ByteQueue(64 * 1024);
    /** Terminal-to-process queue, retained for parity. Subclasses override write() to use a transport. */
    protected final ByteQueue mTerminalToProcessIOQueue = new ByteQueue(4096);
    /** Buffer to write translate code points into utf8 before sending. */
    private final byte[] mUtf8InputBuffer = new byte[5];

    /** Callback which gets notified when a session finishes or changes title. */
    protected TerminalSessionClient mClient;

    /** Logical "alive" flag. 0 = not started, 1 = running, -1 = finished. */
    protected int mShellPid;
    protected int mShellExitStatus;

    /** Set by the application for user identification of session. */
    public String mSessionName;

    protected final Integer mTranscriptRows;

    final Handler mMainThreadHandler = new MainThreadHandler();

    private static final String LOG_TAG = "TerminalSession";

    public TerminalSession(Integer transcriptRows, TerminalSessionClient client) {
        this.mTranscriptRows = transcriptRows;
        this.mClient = client;
    }

    /**
     * Legacy ctor signature kept for source compatibility with upstream
     * TerminalView code. shellPath/cwd/args/env are ignored — subclasses
     * are responsible for their own transport.
     */
    public TerminalSession(String shellPath, String cwd, String[] args, String[] env, Integer transcriptRows, TerminalSessionClient client) {
        this(transcriptRows, client);
    }

    public void updateTerminalSessionClient(TerminalSessionClient client) {
        mClient = client;
        if (mEmulator != null) mEmulator.updateTerminalSessionClient(client);
    }

    /** Inform the attached transport of the new size and reflow or initialize the emulator. */
    public void updateSize(int columns, int rows, int cellWidthPixels, int cellHeightPixels) {
        if (mEmulator == null) {
            initializeEmulator(columns, rows, cellWidthPixels, cellHeightPixels);
        } else {
            mEmulator.resize(columns, rows, cellWidthPixels, cellHeightPixels);
            onSizeChanged(columns, rows, cellWidthPixels, cellHeightPixels);
        }
    }

    public String getTitle() {
        return (mEmulator == null) ? null : mEmulator.getTitle();
    }

    /**
     * Create the {@link TerminalEmulator} and start any transport threads.
     * Subclasses should call super.initializeEmulator(...) to set up the
     * emulator, then start their own I/O.
     */
    public void initializeEmulator(int columns, int rows, int cellWidthPixels, int cellHeightPixels) {
        mEmulator = new TerminalEmulator(this, columns, rows, cellWidthPixels, cellHeightPixels, mTranscriptRows, mClient);
        mShellPid = 1;
        mClient.setTerminalShellPid(this, mShellPid);
        onSizeChanged(columns, rows, cellWidthPixels, cellHeightPixels);
    }

    /** Hook for subclasses to forward resize events to their transport. Default: no-op. */
    protected void onSizeChanged(int columns, int rows, int cellWidthPixels, int cellHeightPixels) {
    }

    /** Write data from user input. Default queues into the legacy buffer; subclasses should override to dispatch to a transport. */
    @Override
    public void write(byte[] data, int offset, int count) {
        if (mShellPid > 0) mTerminalToProcessIOQueue.write(data, offset, count);
    }

    public void writeCodePoint(boolean prependEscape, int codePoint) {
        if (codePoint > 1114111 || (codePoint >= 0xD800 && codePoint <= 0xDFFF)) {
            throw new IllegalArgumentException("Invalid code point: " + codePoint);
        }

        int bufferPosition = 0;
        if (prependEscape) mUtf8InputBuffer[bufferPosition++] = 27;

        if (codePoint <= /* 7 bits */0b1111111) {
            mUtf8InputBuffer[bufferPosition++] = (byte) codePoint;
        } else if (codePoint <= /* 11 bits */0b11111111111) {
            mUtf8InputBuffer[bufferPosition++] = (byte) (0b11000000 | (codePoint >> 6));
            mUtf8InputBuffer[bufferPosition++] = (byte) (0b10000000 | (codePoint & 0b111111));
        } else if (codePoint <= /* 16 bits */0b1111111111111111) {
            mUtf8InputBuffer[bufferPosition++] = (byte) (0b11100000 | (codePoint >> 12));
            mUtf8InputBuffer[bufferPosition++] = (byte) (0b10000000 | ((codePoint >> 6) & 0b111111));
            mUtf8InputBuffer[bufferPosition++] = (byte) (0b10000000 | (codePoint & 0b111111));
        } else {
            mUtf8InputBuffer[bufferPosition++] = (byte) (0b11110000 | (codePoint >> 18));
            mUtf8InputBuffer[bufferPosition++] = (byte) (0b10000000 | ((codePoint >> 12) & 0b111111));
            mUtf8InputBuffer[bufferPosition++] = (byte) (0b10000000 | ((codePoint >> 6) & 0b111111));
            mUtf8InputBuffer[bufferPosition++] = (byte) (0b10000000 | (codePoint & 0b111111));
        }
        write(mUtf8InputBuffer, 0, bufferPosition);
    }

    public TerminalEmulator getEmulator() {
        return mEmulator;
    }

    /** Notify the {@link #mClient} that the screen has changed. */
    protected void notifyScreenUpdate() {
        mClient.onTextChanged(this);
    }

    /**
     * Push bytes received from the transport into the emulator on the main thread.
     * Safe to call from any thread.
     */
    public void emulatorAppend(byte[] bytes, int length) {
        if (length <= 0) return;
        mProcessToTerminalIOQueue.write(bytes, 0, length);
        mMainThreadHandler.sendEmptyMessage(MSG_NEW_INPUT);
    }

    public void reset() {
        if (mEmulator != null) mEmulator.reset();
        notifyScreenUpdate();
    }

    /** Signal that the session is finished. Subclasses should override to close their transport. */
    public void finishIfRunning() {
        // No local PTY to kill. Subclasses tear down their transport here.
    }

    protected void cleanupResources(int exitStatus) {
        synchronized (this) {
            mShellPid = -1;
            mShellExitStatus = exitStatus;
        }
        mTerminalToProcessIOQueue.close();
        mProcessToTerminalIOQueue.close();
    }

    @Override
    public void titleChanged(String oldTitle, String newTitle) {
        mClient.onTitleChanged(this);
    }

    public synchronized boolean isRunning() {
        return mShellPid != -1;
    }

    public synchronized int getExitStatus() {
        return mShellExitStatus;
    }

    @Override
    public void onCopyTextToClipboard(String text) {
        mClient.onCopyTextToClipboard(this, text);
    }

    @Override
    public void onPasteTextFromClipboard() {
        mClient.onPasteTextFromClipboard(this);
    }

    @Override
    public void onBell() {
        mClient.onBell(this);
    }

    @Override
    public void onColorsChanged() {
        mClient.onColorsChanged(this);
    }

    public int getPid() {
        return mShellPid;
    }

    /** Local-pty-only cwd lookup is unavailable for remote sessions. */
    public String getCwd() {
        return null;
    }

    @SuppressLint("HandlerLeak")
    class MainThreadHandler extends Handler {

        final byte[] mReceiveBuffer = new byte[64 * 1024];

        @Override
        public void handleMessage(Message msg) {
            int bytesRead = mProcessToTerminalIOQueue.read(mReceiveBuffer, false);
            if (bytesRead > 0 && mEmulator != null) {
                mEmulator.append(mReceiveBuffer, bytesRead);
                notifyScreenUpdate();
            }

            if (msg.what == MSG_PROCESS_EXITED) {
                int exitCode = (Integer) msg.obj;
                cleanupResources(exitCode);

                String exitDescription = "\r\n[Connection closed";
                if (exitCode != 0) exitDescription += " (code " + exitCode + ")";
                exitDescription += " - press Enter]";

                byte[] bytesToWrite = exitDescription.getBytes(StandardCharsets.UTF_8);
                if (mEmulator != null) {
                    mEmulator.append(bytesToWrite, bytesToWrite.length);
                    notifyScreenUpdate();
                }

                mClient.onSessionFinished(TerminalSession.this);
            }
        }

    }

}
