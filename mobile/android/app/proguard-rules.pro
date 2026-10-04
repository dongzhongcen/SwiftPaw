# 正式版目前没有开启代码压缩（minifyEnabled false）。以后开启时需要保留下面这些类：
# gomobile 生成的 Java 代码通过 JNI 调用，Capacitor 插件通过注解和反射找到。
-keep class go.** { *; }
-keep class com.dongzhongcen.swiftpaw.core.** { *; }
-keep @com.getcapacitor.annotation.CapacitorPlugin class * { *; }
