package com.dongzhongcen.swiftpaw;

import android.content.Context;
import com.dongzhongcen.swiftpaw.core.gocore.Core;

/**
 * Go 内核（internal/core，用 gomobile 编译）的唯一实例。
 * 界面（SwiftPawPlugin）和后台播放服务共用同一个，所以播放队列、歌单在两边是一致的。
 */
public final class GoCore {

    private static Core core;

    private GoCore() {}

    /** 第一次调用时创建内核：数据（配置、数据库、插件）保存在应用的私有文件夹里 */
    public static synchronized Core get(Context context) {
        if (core == null) {
            String dataDir = context.getApplicationContext().getFilesDir().getAbsolutePath();
            core = new Core(dataDir, BuildConfig.VERSION_NAME);
        }
        return core;
    }
}
