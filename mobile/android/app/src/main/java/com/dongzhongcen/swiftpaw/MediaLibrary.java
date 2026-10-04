package com.dongzhongcen.swiftpaw;

import android.content.ContentResolver;
import android.content.Context;
import android.database.Cursor;
import android.net.Uri;
import android.os.Build;
import android.provider.MediaStore;
import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

/**
 * 从系统的媒体库（MediaStore）里读出手机上所有的音乐。
 * 系统已经扫描过存储里的音频文件并读好了标签，所以比自己遍历文件夹快得多，也不需要“所有文件”权限。
 * 铃声、通知音、录音不算音乐（IS_MUSIC = 0），不会出现在列表里。
 */
final class MediaLibrary {

    private MediaLibrary() {}

    /** 返回歌曲列表，格式和 Go 里的 music.Song 一样，可以直接交给 core 的 SetLibrary */
    static JSONArray scan(Context context) throws JSONException {
        Uri collection = Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q
            ? MediaStore.Audio.Media.getContentUri(MediaStore.VOLUME_EXTERNAL)
            : MediaStore.Audio.Media.EXTERNAL_CONTENT_URI;
        String[] columns = {
            MediaStore.Audio.Media.DATA,
            MediaStore.Audio.Media.TITLE,
            MediaStore.Audio.Media.ARTIST,
            MediaStore.Audio.Media.ALBUM,
            MediaStore.Audio.Media.DURATION,
        };
        String selection = MediaStore.Audio.Media.IS_MUSIC + " != 0";
        String order = MediaStore.Audio.Media.DATA + " ASC"; // 和桌面版一样按文件路径排

        JSONArray songs = new JSONArray();
        ContentResolver resolver = context.getContentResolver();
        try (Cursor cursor = resolver.query(collection, columns, selection, null, order)) {
            if (cursor == null) {
                return songs;
            }
            int pathColumn = cursor.getColumnIndexOrThrow(MediaStore.Audio.Media.DATA);
            int titleColumn = cursor.getColumnIndexOrThrow(MediaStore.Audio.Media.TITLE);
            int artistColumn = cursor.getColumnIndexOrThrow(MediaStore.Audio.Media.ARTIST);
            int albumColumn = cursor.getColumnIndexOrThrow(MediaStore.Audio.Media.ALBUM);
            int durationColumn = cursor.getColumnIndexOrThrow(MediaStore.Audio.Media.DURATION);
            while (cursor.moveToNext()) {
                String path = cursor.getString(pathColumn);
                if (path == null || path.isEmpty()) {
                    continue;
                }
                String fileName = path.substring(path.lastIndexOf('/') + 1);
                int dot = fileName.lastIndexOf('.');
                String baseName = dot > 0 ? fileName.substring(0, dot) : fileName;
                String format = dot > 0 ? fileName.substring(dot + 1).toUpperCase(java.util.Locale.ROOT) : "";

                JSONObject song = new JSONObject();
                song.put("path", path);
                song.put("name", baseName);
                song.put("title", textOr(cursor.getString(titleColumn), baseName));
                song.put("artist", textOr(cursor.getString(artistColumn), "未知歌手"));
                song.put("album", textOr(cursor.getString(albumColumn), ""));
                song.put("format", format);
                song.put("source", "local");
                song.put("duration", cursor.getLong(durationColumn) / 1000.0);
                songs.put(song);
            }
        }
        return songs;
    }

    // 没有标签时系统会填 "<unknown>"，当作没有
    private static String textOr(String value, String fallback) {
        if (value == null) {
            return fallback;
        }
        String text = value.trim();
        if (text.isEmpty() || "<unknown>".equals(text)) {
            return fallback;
        }
        return text;
    }
}
