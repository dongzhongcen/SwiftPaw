package com.dongzhongcen.swiftpaw;

import android.content.ActivityNotFoundException;
import android.content.Intent;
import android.net.Uri;
import androidx.activity.OnBackPressedCallback;
import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * 前端和 Go 内核之间的桥。前端（frontend/src/js/platform）把桌面版里调用 Wails 绑定的地方
 * 换成调用这个插件，方法名和参数完全一样。
 */
@CapacitorPlugin(name = "SwiftPaw")
public class SwiftPawPlugin extends Plugin {

    // 插件搜索、下载可能要好几秒，放到线程池里做，不挡住其它调用
    private final ExecutorService executor = Executors.newCachedThreadPool();

    @Override
    public void load() {
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

    @Override
    protected void handleOnDestroy() {
        executor.shutdown();
    }
}
