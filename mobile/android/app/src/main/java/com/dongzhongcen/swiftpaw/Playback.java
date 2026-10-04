package com.dongzhongcen.swiftpaw;

import android.content.Context;
import android.net.Uri;
import android.os.Handler;
import android.os.Looper;
import androidx.annotation.NonNull;
import androidx.media3.common.C;
import androidx.media3.common.MediaItem;
import androidx.media3.common.MediaMetadata;
import androidx.media3.common.PlaybackException;
import androidx.media3.common.Player;
import androidx.media3.exoplayer.ExoPlayer;
import com.dongzhongcen.swiftpaw.core.gocore.Picture;
import java.io.File;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

/**
 * 原生播放的逻辑。播放顺序和桌面版一样由 Go 内核的播放队列决定，这里负责：
 * 加载队列里正在播放的那首歌、一首放完后自动接下一首、放不了时跳过、记“最近播放”，
 * 以及响应通知栏和锁屏上的 上一首 / 下一首。
 *
 * 播放器（ExoPlayer）属于 PlaybackService，所以界面关掉以后音乐也能继续放。
 * 界面（SwiftPawPlugin）通过 Listener 收到状态变化，再转给前端。
 * 所有方法都要在主线程调用；调用 Go 内核（可能要等插件返回地址）放在后台线程。
 */
final class Playback {

    /** 前端关心的事件 */
    interface Listener {
        /** 播放状态变了：在不在播放、进度、时长 */
        void onState(JSONObject state);

        /** 原生这边换了歌（自动下一首、通知栏上一首/下一首），把新的队列快照给前端 */
        void onQueueChanged(String queueJson);

        /** 某首歌放不了 */
        void onError(String message);

        /** 记了一次“最近播放” */
        void onRecentChanged();
    }

    private static Playback instance;

    static synchronized Playback get(Context context) {
        if (instance == null) {
            instance = new Playback(context.getApplicationContext());
        }
        return instance;
    }

    private final Context context;
    private final Handler main = new Handler(Looper.getMainLooper());
    // 播放相关的 Go 调用按顺序一个一个来，免得快速连点“下一首”时乱序
    private final ExecutorService worker = Executors.newSingleThreadExecutor();
    private final List<Runnable> waitingForPlayer = new ArrayList<>();

    private ExoPlayer player;
    private Listener listener;

    private String loadedKey = ""; // 播放器里现在是哪首歌（和前端的 songKey 规则一样）
    private JSONObject loadedSong;
    private String recordedKey = ""; // 已经记过“最近播放”的歌
    private int loadToken = 0; // 每次加载加一，后台线程回来时用它判断是不是已经过时了
    private boolean loading = false;
    private boolean playWhenLoaded = false;
    private int errorStreak = 0; // 连续放不了的次数，整个队列都放不了时停下来

    // 在线歌曲要带的请求头（插件给的 Referer、User-Agent 等），按播放地址查
    private volatile Map<String, Map<String, String>> requestHeaders = Collections.emptyMap();

    private Playback(Context context) {
        this.context = context;
    }

    // ---- PlaybackService 调用 ----

    void attach(ExoPlayer player) {
        this.player = player;
        player.addListener(playerListener);
        List<Runnable> tasks = new ArrayList<>(waitingForPlayer);
        waitingForPlayer.clear();
        for (Runnable task : tasks) {
            task.run();
        }
    }

    void detach() {
        if (player != null) {
            player.removeListener(playerListener);
        }
        player = null;
        loadedKey = "";
        loadedSong = null;
    }

    Map<String, String> headersFor(Uri uri) {
        Map<String, String> headers = requestHeaders.get(uri.toString());
        return headers != null ? headers : Collections.emptyMap();
    }

    // ---- SwiftPawPlugin 调用 ----

    void setListener(Listener listener) {
        this.listener = listener;
        if (listener != null && player != null) {
            emitState();
        }
    }

    /** whenReady 在播放器准备好以后执行（服务刚启动时播放器还没建好） */
    void whenReady(Runnable task) {
        if (player != null) {
            task.run();
        } else {
            waitingForPlayer.add(task);
        }
    }

    /**
     * 加载 Go 播放队列里 key 对应的歌（一般是正在播放的那首）。key 和现在的一样时什么都不做；
     * key 为空表示队列空了，停止播放。
     */
    void load(String key, boolean play) {
        errorStreak = 0;
        loadInternal(key, play);
    }

    void play() {
        if (player == null) {
            return;
        }
        if (loading) {
            playWhenLoaded = true;
            return;
        }
        if (player.getPlaybackState() == Player.STATE_ENDED) {
            player.seekTo(0);
        }
        if (player.getPlaybackState() == Player.STATE_IDLE && player.getMediaItemCount() > 0) {
            player.prepare(); // 出错停下以后再点播放，重新试一次
        }
        player.play();
    }

    void pause() {
        playWhenLoaded = false;
        if (player != null) {
            player.pause();
        }
    }

    void seek(long positionMs) {
        if (player != null && !loading) {
            player.seekTo(Math.max(0, positionMs));
        }
    }

    void setVolume(float volume) {
        if (player != null) {
            player.setVolume(Math.max(0f, Math.min(1f, volume)));
        }
    }

    /** 通知栏、锁屏、耳机上的“下一首” */
    void next() {
        advance("QueueNext", "[false]");
    }

    /** 通知栏、锁屏、耳机上的“上一首” */
    void previous() {
        advance("QueuePrevious", "[]");
    }

    // ---- 内部 ----

    private void loadInternal(String key, boolean play) {
        if (player == null) {
            return;
        }
        if (key == null || key.isEmpty()) {
            loadToken++;
            loading = false;
            loadedKey = "";
            loadedSong = null;
            player.stop();
            player.clearMediaItems();
            emitState();
            return;
        }
        if (key.equals(loadedKey)) {
            if (play) {
                play();
            }
            return;
        }
        loadedKey = key;
        recordedKey = "";
        loading = true;
        playWhenLoaded = play;
        int token = ++loadToken;
        emitState();
        worker.execute(() -> {
            JSONObject[] found = { null };
            try {
                JSONObject song = findSong(key);
                found[0] = song;
                MediaItem item = mediaItemOf(song);
                main.post(() -> {
                    if (token != loadToken || player == null) {
                        return;
                    }
                    loading = false;
                    loadedSong = song;
                    player.setMediaItem(item);
                    player.prepare();
                    player.setPlayWhenReady(playWhenLoaded);
                    emitState();
                });
            } catch (Exception e) {
                main.post(() -> {
                    if (token != loadToken) {
                        return;
                    }
                    loading = false;
                    loadedSong = found[0];
                    skipBroken(key, e.getMessage());
                });
            }
        });
    }

    // findSong 在 Go 的播放队列里找到 key 对应的歌
    private JSONObject findSong(String key) throws Exception {
        JSONObject state = new JSONObject(GoCore.get(context).call("QueueState", "[]"));
        JSONArray songs = state.optJSONArray("songs");
        if (songs == null) {
            throw new Exception("播放队列是空的");
        }
        int current = state.optInt("current", -1);
        if (current >= 0 && current < songs.length() && key.equals(keyOf(songs.getJSONObject(current)))) {
            return songs.getJSONObject(current);
        }
        for (int i = 0; i < songs.length(); i++) {
            if (key.equals(keyOf(songs.getJSONObject(i)))) {
                return songs.getJSONObject(i);
            }
        }
        throw new Exception("播放队列里没有这首歌");
    }

    // mediaItemOf 生成播放器要的 MediaItem：本地歌曲直接读文件，在线歌曲先问插件要地址
    private MediaItem mediaItemOf(JSONObject song) throws Exception {
        MediaMetadata.Builder metadata = new MediaMetadata.Builder()
            .setTitle(displayName(song))
            .setArtist(song.optString("artist", ""))
            .setAlbumTitle(song.optString("album", ""));
        Uri uri;
        if (isOnline(song)) {
            JSONObject source = new JSONObject(GoCore.get(context).call("MediaSource", "[" + song + "]"));
            String url = source.optString("url", "");
            if (url.isEmpty()) {
                throw new Exception("插件没有返回播放地址");
            }
            uri = Uri.parse(url);
            Map<String, String> headers = new HashMap<>();
            JSONObject h = source.optJSONObject("headers");
            if (h != null) {
                for (java.util.Iterator<String> it = h.keys(); it.hasNext();) {
                    String name = it.next();
                    headers.put(name, h.optString(name));
                }
            }
            requestHeaders = Collections.singletonMap(uri.toString(), headers);
            String artwork = song.optString("artwork", "");
            if (!artwork.isEmpty()) {
                metadata.setArtworkUri(Uri.parse(artwork));
            }
        } else {
            String path = song.optString("path", "");
            uri = Uri.fromFile(new File(path));
            try {
                Picture cover = GoCore.get(context).cover(path);
                metadata.setArtworkData(cover.getData(), MediaMetadata.PICTURE_TYPE_FRONT_COVER);
            } catch (Exception ignored) {
                // 没有内嵌封面，通知栏显示默认图标
            }
        }
        return new MediaItem.Builder().setMediaId(keyOf(song)).setUri(uri).setMediaMetadata(metadata.build()).build();
    }

    // advance 调用 Go 队列的方法（下一首、上一首）并播放新的当前歌曲
    private void advance(String method, String args) {
        worker.execute(() -> {
            try {
                String queueJson = GoCore.get(context).call(method, args);
                main.post(() -> applyQueue(queueJson));
            } catch (Exception e) {
                main.post(() -> emitError("切换歌曲失败：" + e.getMessage()));
            }
        });
    }

    private void applyQueue(String queueJson) {
        String key = "";
        try {
            JSONObject state = new JSONObject(queueJson);
            JSONArray songs = state.optJSONArray("songs");
            int current = state.optInt("current", -1);
            if (songs != null && current >= 0 && current < songs.length()) {
                key = keyOf(songs.getJSONObject(current));
            }
        } catch (JSONException ignored) {
            // 解析不了就当作队列空了
        }
        if (listener != null) {
            listener.onQueueChanged(queueJson);
        }
        if (!key.isEmpty() && key.equals(loadedKey) && player != null) {
            // 同一首歌再放一次（单曲循环，或者队列里只有这一首）
            player.seekTo(0);
            play();
            return;
        }
        loadInternal(key, true);
    }

    // skipBroken 某首歌放不了：提示一下并跳到下一首；整个队列都放不了时停下来，防止无限跳歌
    private void skipBroken(String key, String reason) {
        errorStreak++;
        String name = loadedSong != null && key.equals(keyOf(loadedSong)) ? displayName(loadedSong) : "这首歌";
        String detail = reason != null && !reason.isEmpty() ? "（" + reason + "）" : "";
        int size = queueSize();
        if (errorStreak >= Math.max(1, size)) {
            emitError("无法播放：" + name + detail + "，队列里的歌都放不了，已停止");
            if (player != null) {
                player.stop();
            }
            emitState();
            return;
        }
        emitError("无法播放：" + name + detail + "，已跳到下一首");
        advance("QueueNext", "[false]");
    }

    private int queueSize() {
        try {
            JSONObject state = new JSONObject(GoCore.get(context).call("QueueState", "[]"));
            JSONArray songs = state.optJSONArray("songs");
            return songs != null ? songs.length() : 0;
        } catch (Exception e) {
            return 0;
        }
    }

    private void recordPlayOnce() {
        JSONObject song = loadedSong;
        if (song == null || loadedKey.equals(recordedKey)) {
            return;
        }
        recordedKey = loadedKey;
        worker.execute(() -> {
            try {
                GoCore.get(context).call("RecordPlay", "[" + song + "]");
                main.post(() -> {
                    if (listener != null) {
                        listener.onRecentChanged();
                    }
                });
            } catch (Exception ignored) {
                // 记不上最近播放不影响播放
            }
        });
    }

    private void emitError(String message) {
        if (listener != null) {
            listener.onError(message);
        }
    }

    private void emitState() {
        if (listener == null || player == null) {
            return;
        }
        try {
            JSONObject state = new JSONObject();
            int playbackState = player.getPlaybackState();
            boolean wantsToPlay = loading ? playWhenLoaded : player.getPlayWhenReady() && playbackState != Player.STATE_ENDED;
            long duration = player.getDuration();
            state.put("key", loadedKey);
            state.put("playing", wantsToPlay);
            state.put("buffering", loading || playbackState == Player.STATE_BUFFERING);
            state.put("position", loading ? 0 : player.getCurrentPosition() / 1000.0);
            state.put("duration", loading || duration == C.TIME_UNSET ? -1 : duration / 1000.0);
            listener.onState(state);
        } catch (JSONException ignored) {
            // 不会发生
        }
    }

    private final Player.Listener playerListener = new Player.Listener() {
        @Override
        public void onPlaybackStateChanged(int playbackState) {
            emitState();
            if (playbackState == Player.STATE_ENDED && !loadedKey.isEmpty()) {
                advance("QueueNext", "[true]"); // 自动下一首（单曲循环时 Go 会返回同一首）
            }
        }

        @Override
        public void onIsPlayingChanged(boolean isPlaying) {
            emitState();
            if (isPlaying) {
                errorStreak = 0;
                recordPlayOnce();
            }
        }

        @Override
        public void onPlayWhenReadyChanged(boolean playWhenReady, int reason) {
            emitState();
        }

        @Override
        public void onPositionDiscontinuity(
            @NonNull Player.PositionInfo oldPosition,
            @NonNull Player.PositionInfo newPosition,
            int reason
        ) {
            emitState();
        }

        @Override
        public void onPlayerError(@NonNull PlaybackException error) {
            String key = loadedKey;
            if (key.isEmpty()) {
                return;
            }
            skipBroken(key, errorText(error));
        }
    };

    private static String errorText(PlaybackException error) {
        switch (error.errorCode) {
            case PlaybackException.ERROR_CODE_IO_FILE_NOT_FOUND:
                return "文件不存在";
            case PlaybackException.ERROR_CODE_IO_NO_PERMISSION:
                return "没有读取权限";
            case PlaybackException.ERROR_CODE_IO_NETWORK_CONNECTION_FAILED:
            case PlaybackException.ERROR_CODE_IO_NETWORK_CONNECTION_TIMEOUT:
                return "网络连接失败";
            case PlaybackException.ERROR_CODE_IO_BAD_HTTP_STATUS:
                return "服务器拒绝了请求";
            case PlaybackException.ERROR_CODE_PARSING_CONTAINER_UNSUPPORTED:
            case PlaybackException.ERROR_CODE_DECODING_FORMAT_UNSUPPORTED:
                return "不支持这种格式";
            default:
                return "";
        }
    }

    // ---- 歌曲信息，规则和前端 util.js 一样 ----

    static boolean isOnline(JSONObject song) {
        String source = song.optString("source", "");
        return !source.isEmpty() && !"local".equals(source);
    }

    static String keyOf(JSONObject song) {
        return isOnline(song) ? song.optString("source") + ":" + song.optString("id") : song.optString("path", "");
    }

    static String displayName(JSONObject song) {
        String title = song.optString("title", "");
        if (!title.isEmpty()) {
            return title;
        }
        String name = song.optString("name", "");
        return name.isEmpty() ? "未知歌曲" : name;
    }
}
