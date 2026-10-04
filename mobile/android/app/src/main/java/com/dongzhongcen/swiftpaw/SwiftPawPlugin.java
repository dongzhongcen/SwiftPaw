package com.dongzhongcen.swiftpaw;

import android.Manifest;
import android.app.Activity;
import android.content.ActivityNotFoundException;
import android.content.ComponentName;
import android.content.Intent;
import android.database.Cursor;
import android.net.Uri;
import android.os.Build;
import android.provider.OpenableColumns;
import androidx.activity.result.ActivityResult;
import androidx.activity.OnBackPressedCallback;
import androidx.media3.session.MediaController;
import androidx.media3.session.SessionToken;
import com.getcapacitor.JSObject;
import com.getcapacitor.PermissionState;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.ActivityCallback;
import com.getcapacitor.annotation.CapacitorPlugin;
import com.getcapacitor.annotation.Permission;
import com.getcapacitor.annotation.PermissionCallback;
import com.google.common.util.concurrent.ListenableFuture;
import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import org.json.JSONArray;
import org.json.JSONObject;

/**
 * 前端和 Go 内核之间的桥。前端（frontend/src/js/platform）把桌面版里调用 Wails 绑定的地方
 * 换成调用这个插件，方法名和参数完全一样。
 */
@CapacitorPlugin(
    name = "SwiftPaw",
    permissions = {
        // 读取手机里的音乐：Android 13 起是 READ_MEDIA_AUDIO，之前是 READ_EXTERNAL_STORAGE
        @Permission(alias = SwiftPawPlugin.AUDIO, strings = { Manifest.permission.READ_MEDIA_AUDIO }),
        @Permission(alias = SwiftPawPlugin.STORAGE, strings = { Manifest.permission.READ_EXTERNAL_STORAGE }),
    }
)
public class SwiftPawPlugin extends Plugin {

    static final String AUDIO = "audio";
    static final String STORAGE = "storage";

    // 连着后台播放服务，服务才会启动；界面销毁时断开
    private ListenableFuture<MediaController> controller;

    // 插件搜索、下载可能要好几秒，放到线程池里做，不挡住其它调用
    private final ExecutorService executor = Executors.newCachedThreadPool();

    @Override
    public void load() {
        // 拦下 /cover 请求，用 Go 内核读封面（桌面版是 Go 的 HTTP 服务提供的）
        getBridge().setWebViewClient(new CoverWebViewClient(getBridge()));
        connectPlayback();

        // 返回键交给前端处理（关对话框、收起播放页、回上一级页面），前端还没准备好时直接退到后台
        getActivity()
            .getOnBackPressedDispatcher()
            .addCallback(
                getActivity(),
                new OnBackPressedCallback(true) {
                    @Override
                    public void handleOnBackPressed() {
                        if (hasListeners("backButton")) {
                            notifyListeners("backButton", new JSObject());
                        } else {
                            getActivity().moveTaskToBack(true);
                        }
                    }
                }
            );
    }

    /**
     * 调用内核的一个方法。参数：method（方法名），args（参数数组的 JSON 字符串）。
     * 返回 { json: 结果的 JSON 字符串 }，内核返回错误时 reject，错误信息就是 Go 的错误文字。
     */
    @PluginMethod
    public void call(PluginCall call) {
        String method = call.getString("method", "");
        String args = call.getString("args", "[]");
        executor.execute(() -> {
            try {
                String result = GoCore.get(getContext()).call(method, args);
                JSObject ret = new JSObject();
                ret.put("json", result);
                call.resolve(ret);
            } catch (Exception e) {
                call.reject(e.getMessage() != null ? e.getMessage() : e.toString());
            }
        });
    }

    /** 退到后台，和按 Home 键一样（音乐继续放），不关掉应用。 */
    @PluginMethod
    public void minimize(PluginCall call) {
        getActivity().runOnUiThread(() -> {
            getActivity().moveTaskToBack(true);
            call.resolve();
        });
    }

    /** 用系统浏览器打开网址（只允许 http 和 https）。 */
    @PluginMethod
    public void openUrl(PluginCall call) {
        String url = call.getString("url", "");
        Uri uri = Uri.parse(url);
        String scheme = uri.getScheme();
        if (!"http".equals(scheme) && !"https".equals(scheme)) {
            call.reject("只能打开 http 或 https 网址");
            return;
        }
        try {
            Intent intent = new Intent(Intent.ACTION_VIEW, uri);
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            getContext().startActivity(intent);
            call.resolve();
        } catch (ActivityNotFoundException e) {
            call.reject("没有找到可以打开网址的应用");
        }
    }

    // ---- 本机音乐 ----

    /** 扫描手机里的音乐，交给 Go 内核作为歌曲库。返回 { json: 歌曲列表 } */
    @PluginMethod
    public void scanLibrary(PluginCall call) {
        String alias = audioPermission();
        if (getPermissionState(alias) != PermissionState.GRANTED) {
            requestPermissionForAlias(alias, call, "onAudioPermission");
            return;
        }
        doScan(call);
    }

    @PermissionCallback
    private void onAudioPermission(PluginCall call) {
        if (getPermissionState(audioPermission()) == PermissionState.GRANTED) {
            doScan(call);
        } else {
            call.reject("没有读取音乐的权限。请在系统设置的“应用 → 极拍 → 权限”里允许访问音乐和音频，然后再扫描");
        }
    }

    private static String audioPermission() {
        return Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU ? AUDIO : STORAGE;
    }

    private void doScan(PluginCall call) {
        executor.execute(() -> {
            try {
                JSONArray songs = MediaLibrary.scan(getContext());
                String kept = GoCore.get(getContext()).call("SetLibrary", "[" + songs + "]");
                JSObject ret = new JSObject();
                ret.put("json", kept);
                call.resolve(ret);
            } catch (Exception e) {
                call.reject("扫描失败：" + e.getMessage());
            }
        });
    }

    // ---- 插件 ----

    // 插件文件最大 2 MB（和 Go 里读插件文件的上限一样）
    private static final int MAX_PLUGIN_SIZE = 2 * 1024 * 1024;

    /** 用系统的文件选择器选一个 .js 插件并安装。返回 { json: 插件信息 }，取消时返回 {} */
    @PluginMethod
    public void installPluginFile(PluginCall call) {
        Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT);
        intent.addCategory(Intent.CATEGORY_OPENABLE);
        intent.setType("*/*"); // .js 在不同手机上的 MIME 类型不一样，不按类型过滤，安装时再检查扩展名
        startActivityForResult(call, intent, "onPluginFilePicked");
    }

    @ActivityCallback
    private void onPluginFilePicked(PluginCall call, ActivityResult result) {
        if (call == null) {
            return;
        }
        Intent data = result.getData();
        if (result.getResultCode() != Activity.RESULT_OK || data == null || data.getData() == null) {
            call.resolve(); // 用户取消了
            return;
        }
        Uri uri = data.getData();
        executor.execute(() -> {
            File copy = null;
            try {
                // Go 内核按文件路径安装，插件的 id 是文件名，所以先用原来的文件名复制到缓存文件夹
                File dir = new File(getContext().getCacheDir(), "plugin-import");
                if (!dir.isDirectory() && !dir.mkdirs()) {
                    throw new IOException("无法创建临时文件夹");
                }
                copy = new File(dir, safeFileName(displayName(uri)));
                copyLimited(uri, copy);
                String info = GoCore.get(getContext()).call("InstallPluginFile", "[" + org.json.JSONObject.quote(copy.getAbsolutePath()) + "]");
                JSObject ret = new JSObject();
                ret.put("json", info);
                call.resolve(ret);
            } catch (Exception e) {
                call.reject(e.getMessage() != null ? e.getMessage() : e.toString());
            } finally {
                if (copy != null) {
                    //noinspection ResultOfMethodCallIgnored
                    copy.delete();
                }
            }
        });
    }

    private String displayName(Uri uri) {
        try (Cursor cursor = getContext().getContentResolver().query(uri, new String[] { OpenableColumns.DISPLAY_NAME }, null, null, null)) {
            if (cursor != null && cursor.moveToFirst()) {
                String name = cursor.getString(0);
                if (name != null && !name.isEmpty()) {
                    return name;
                }
            }
        }
        String last = uri.getLastPathSegment();
        return last != null ? last : "plugin.js";
    }

    // 文件名里只留下安全的部分（不能带路径）
    private static String safeFileName(String name) {
        String base = name.substring(name.lastIndexOf('/') + 1).replace('\\', '_').trim();
        return base.isEmpty() || base.startsWith(".") ? "plugin" + base : base;
    }

    private void copyLimited(Uri uri, File target) throws IOException {
        try (InputStream in = getContext().getContentResolver().openInputStream(uri); OutputStream out = new FileOutputStream(target)) {
            if (in == null) {
                throw new IOException("读不了这个文件");
            }
            byte[] buffer = new byte[16 * 1024];
            int total = 0;
            int n;
            while ((n = in.read(buffer)) != -1) {
                total += n;
                if (total > MAX_PLUGIN_SIZE) {
                    throw new IOException("插件文件太大了（超过 2 MB）");
                }
                out.write(buffer, 0, n);
            }
        }
    }

    // ---- 关于 ----

    /** 读出安装包里带的许可证文件（LICENSE 和 THIRD_PARTY_NOTICES.md）。返回 { text } */
    @PluginMethod
    public void readNotices(PluginCall call) {
        executor.execute(() -> {
            try {
                String text = readAsset("licenses/LICENSE") + "\n\n" + readAsset("licenses/THIRD_PARTY_NOTICES.md");
                JSObject ret = new JSObject();
                ret.put("text", text);
                call.resolve(ret);
            } catch (IOException e) {
                call.reject("读不到许可证文件：" + e.getMessage());
            }
        });
    }

    private String readAsset(String name) throws IOException {
        try (InputStream in = getContext().getAssets().open(name)) {
            ByteArrayOutputStream out = new ByteArrayOutputStream();
            byte[] buffer = new byte[16 * 1024];
            int n;
            while ((n = in.read(buffer)) != -1) {
                out.write(buffer, 0, n);
            }
            return out.toString(StandardCharsets.UTF_8.name());
        }
    }

    // ---- 播放：前端的“播放器”在 Android 上换成了原生的，见 frontend/src/js/native/audio.js ----

    private void connectPlayback() {
        SessionToken token = new SessionToken(getContext(), new ComponentName(getContext(), PlaybackService.class));
        controller = new MediaController.Builder(getContext(), token).buildAsync();
        Playback.get(getContext()).setListener(
            new Playback.Listener() {
                @Override
                public void onState(JSONObject state) {
                    try {
                        notifyListeners("playerState", JSObject.fromJSONObject(state));
                    } catch (org.json.JSONException ignored) {
                        // 不会发生
                    }
                }

                @Override
                public void onQueueChanged(String queueJson) {
                    JSObject data = new JSObject();
                    data.put("json", queueJson);
                    notifyListeners("queueChanged", data);
                }

                @Override
                public void onError(String message) {
                    JSObject data = new JSObject();
                    data.put("message", message);
                    notifyListeners("playerError", data);
                }

                @Override
                public void onRecentChanged() {
                    notifyListeners("recentChanged", new JSObject());
                }
            }
        );
    }

    // onPlayer 在主线程、播放服务准备好以后执行
    private void onPlayer(PluginCall call, java.util.function.Consumer<Playback> task) {
        getActivity().runOnUiThread(() -> {
            Playback playback = Playback.get(getContext());
            playback.whenReady(() -> {
                task.accept(playback);
                call.resolve();
            });
        });
    }

    /** 加载 Go 播放队列里的一首歌。参数：key（为空表示停止），play */
    @PluginMethod
    public void playerLoad(PluginCall call) {
        String key = call.getString("key", "");
        boolean play = Boolean.TRUE.equals(call.getBoolean("play", false));
        onPlayer(call, (p) -> p.load(key, play));
    }

    @PluginMethod
    public void playerPlay(PluginCall call) {
        onPlayer(call, Playback::play);
    }

    @PluginMethod
    public void playerPause(PluginCall call) {
        onPlayer(call, Playback::pause);
    }

    /** 跳到某个位置。参数：position（秒） */
    @PluginMethod
    public void playerSeek(PluginCall call) {
        double position = call.getDouble("position", 0.0);
        onPlayer(call, (p) -> p.seek(Math.round(position * 1000)));
    }

    /** 音量。参数：volume（0 到 1） */
    @PluginMethod
    public void playerSetVolume(PluginCall call) {
        double volume = call.getDouble("volume", 1.0);
        onPlayer(call, (p) -> p.setVolume((float) volume));
    }

    @Override
    protected void handleOnDestroy() {
        Playback.get(getContext()).setListener(null);
        if (controller != null) {
            MediaController.releaseFuture(controller);
            controller = null;
        }
        executor.shutdown();
    }
}
