package com.dongzhongcen.swiftpaw;

import android.app.PendingIntent;
import android.content.Intent;
import androidx.annotation.Nullable;
import androidx.media3.common.AudioAttributes;
import androidx.media3.common.C;
import androidx.media3.common.ForwardingPlayer;
import androidx.media3.common.Player;
import androidx.media3.datasource.DataSpec;
import androidx.media3.datasource.DefaultDataSource;
import androidx.media3.datasource.DefaultHttpDataSource;
import androidx.media3.datasource.ResolvingDataSource;
import androidx.media3.exoplayer.ExoPlayer;
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory;
import androidx.media3.session.MediaSession;
import androidx.media3.session.MediaSessionService;
import java.util.Map;

/**
 * 后台播放服务。音乐在这里放，所以切到别的应用、锁屏以后还能继续放；
 * 系统的媒体通知、锁屏控制、蓝牙耳机按键都通过 MediaSession 连到这里。
 */
public class PlaybackService extends MediaSessionService {

    private MediaSession session;

    @Override
    public void onCreate() {
        super.onCreate();
        Playback playback = Playback.get(this);

        // 在线歌曲的请求头（Referer 等）按地址加上去
        DefaultHttpDataSource.Factory http = new DefaultHttpDataSource.Factory()
            .setAllowCrossProtocolRedirects(true)
            .setConnectTimeoutMs(15000)
            .setReadTimeoutMs(20000);
        ResolvingDataSource.Factory dataSource = new ResolvingDataSource.Factory(
            new DefaultDataSource.Factory(this, http),
            (DataSpec spec) -> {
                Map<String, String> headers = playback.headersFor(spec.uri);
                return headers.isEmpty() ? spec : spec.withAdditionalHeaders(headers);
            }
        );

        ExoPlayer player = new ExoPlayer.Builder(this)
            .setMediaSourceFactory(new DefaultMediaSourceFactory(dataSource))
            .setAudioAttributes(
                new AudioAttributes.Builder().setUsage(C.USAGE_MEDIA).setContentType(C.AUDIO_CONTENT_TYPE_MUSIC).build(),
                true // 处理音频焦点：来电话、别的应用放声音时自动暂停
            )
            .setHandleAudioBecomingNoisy(true) // 拔耳机、断开蓝牙时暂停
            .setWakeMode(C.WAKE_MODE_NETWORK) // 锁屏后 CPU 和 Wi-Fi 不休眠
            .build();

        Intent open = new Intent(this, MainActivity.class).addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP);
        PendingIntent sessionActivity = PendingIntent.getActivity(
            this,
            0,
            open,
            PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT
        );
        session = new MediaSession.Builder(this, new QueuePlayer(player, playback))
            .setSessionActivity(sessionActivity)
            .build();
        playback.attach(player);
    }

    @Nullable
    @Override
    public MediaSession onGetSession(MediaSession.ControllerInfo controllerInfo) {
        return session;
    }

    @Override
    public void onDestroy() {
        Playback.get(this).detach();
        if (session != null) {
            session.getPlayer().release();
            session.release();
            session = null;
        }
        super.onDestroy();
    }

    /**
     * 播放器里每次只放一首歌，上一首 / 下一首由 Go 的播放队列决定。
     * 这里告诉系统“有上一首和下一首”，并把通知栏、锁屏、耳机上的按键转给 Playback。
     */
    private static final class QueuePlayer extends ForwardingPlayer {

        private final Playback playback;

        QueuePlayer(Player player, Playback playback) {
            super(player);
            this.playback = playback;
        }

        @Override
        public Player.Commands getAvailableCommands() {
            return super.getAvailableCommands()
                .buildUpon()
                .add(COMMAND_SEEK_TO_NEXT)
                .add(COMMAND_SEEK_TO_NEXT_MEDIA_ITEM)
                .add(COMMAND_SEEK_TO_PREVIOUS)
                .add(COMMAND_SEEK_TO_PREVIOUS_MEDIA_ITEM)
                .build();
        }

        @Override
        public boolean isCommandAvailable(int command) {
            switch (command) {
                case COMMAND_SEEK_TO_NEXT:
                case COMMAND_SEEK_TO_NEXT_MEDIA_ITEM:
                case COMMAND_SEEK_TO_PREVIOUS:
                case COMMAND_SEEK_TO_PREVIOUS_MEDIA_ITEM:
                    return true;
                default:
                    return super.isCommandAvailable(command);
            }
        }

        @Override
        public void seekToNext() {
            playback.next();
        }

        @Override
        public void seekToNextMediaItem() {
            playback.next();
        }

        @Override
        public void seekToPrevious() {
            playback.previous();
        }

        @Override
        public void seekToPreviousMediaItem() {
            playback.previous();
        }

        @Override
        public void play() {
            playback.play(); // 放完了或者出错停下时，从系统通知栏点播放也能重新开始
        }
    }
}
