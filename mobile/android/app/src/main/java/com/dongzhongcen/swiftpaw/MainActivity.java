package com.dongzhongcen.swiftpaw;

import android.os.Bundle;
import com.getcapacitor.BridgeActivity;

/** 主界面：一个全屏的 WebView，加载和桌面版相同的前端（frontend/dist） */
public class MainActivity extends BridgeActivity {

    @Override
    public void onCreate(Bundle savedInstanceState) {
        // 自己写的插件要在 super.onCreate 之前注册
        registerPlugin(SwiftPawPlugin.class);
        super.onCreate(savedInstanceState);
    }
}
