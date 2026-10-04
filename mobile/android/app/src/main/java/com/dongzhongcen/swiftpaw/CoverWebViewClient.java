package com.dongzhongcen.swiftpaw;

import android.net.Uri;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebView;
import com.dongzhongcen.swiftpaw.core.gocore.BackgroundImage;
import com.dongzhongcen.swiftpaw.core.gocore.Picture;
import com.getcapacitor.Bridge;
import com.getcapacitor.BridgeWebViewClient;
import java.io.ByteArrayInputStream;
import java.io.FileInputStream;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

/**
 * 桌面版的封面和背景图片由 Go 的 /cover?path=...、/background?name=... 路由提供。Android 版没有这个 HTTP 服务，
 * 所以在 WebView 里拦下这两个地址，直接用 Go 内核读出音频文件里的封面、找到保存的背景图片。
 * 其他请求照常交给 Capacitor。
 */
public class CoverWebViewClient extends BridgeWebViewClient {

    private final Bridge bridge;

    public CoverWebViewClient(Bridge bridge) {
        super(bridge);
        this.bridge = bridge;
    }

    @Override
    public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest request) {
        Uri url = request.getUrl();
        if (isAppUrl(url) && "/cover".equals(url.getPath())) {
            return cover(url.getQueryParameter("path"));
        }
        if (isAppUrl(url) && "/background".equals(url.getPath())) {
            return background(url.getQueryParameter("name"));
        }
        return super.shouldInterceptRequest(view, request);
    }

    private boolean isAppUrl(Uri url) {
        return bridge.getHost().equals(url.getHost());
    }

    private WebResourceResponse cover(String path) {
        if (path == null || path.isEmpty()) {
            return notFound();
        }
        try {
            Picture picture = GoCore.get(bridge.getContext()).cover(path);
            WebResourceResponse response = new WebResourceResponse(
                picture.getMIME(),
                null,
                new ByteArrayInputStream(picture.getData())
            );
            response.setResponseHeaders(Collections.singletonMap("Cache-Control", "max-age=3600"));
            return response;
        } catch (Exception e) {
            return notFound(); // 没有内嵌封面，前端会显示默认图标
        }
    }

    private WebResourceResponse background(String name) {
        if (name == null || name.isEmpty()) {
            return notFound();
        }
        try {
            // Go 内核只认 SetBackgroundImage 返回的文件名，不能拿它读别的文件
            BackgroundImage image = GoCore.get(bridge.getContext()).background(name);
            Map<String, String> headers = new HashMap<>();
            headers.put("Cache-Control", "max-age=31536000, immutable"); // 文件名里带着内容的哈希
            return new WebResourceResponse(image.getMIME(), null, 200, "OK", headers, new FileInputStream(image.getPath()));
        } catch (Exception e) {
            return notFound();
        }
    }

    private static WebResourceResponse notFound() {
        return new WebResourceResponse(
            "text/plain",
            "utf-8",
            404,
            "Not Found",
            Collections.emptyMap(),
            new ByteArrayInputStream(new byte[0])
        );
    }
}
