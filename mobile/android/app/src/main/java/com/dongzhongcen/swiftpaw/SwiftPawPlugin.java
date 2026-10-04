package com.dongzhongcen.swiftpaw;

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

    @Override
    protected void handleOnDestroy() {
        executor.shutdown();
    }
}
