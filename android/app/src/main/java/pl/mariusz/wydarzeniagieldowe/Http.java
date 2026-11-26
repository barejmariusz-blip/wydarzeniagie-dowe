package pl.mariusz.wydarzeniagieldowe;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;

/** Prosty GET używany przez UI i synchronizację w tle. */
final class Http {
    interface Done { void done(int status, String body); }

    static void get(String url, Done cb) {
        HttpURLConnection c = null;
        try {
            c = (HttpURLConnection) new URL(url).openConnection();
            c.setConnectTimeout(15000);
            c.setReadTimeout(20000);
            c.setInstanceFollowRedirects(true);
            c.setRequestProperty("User-Agent", "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Mobile Safari/537.36 WydarzeniaGieldowe/0.1");
            c.setRequestProperty("Accept", "application/json, text/calendar, text/plain, */*");
            if (url.contains("cnn.io")) {   // CNN Fear & Greed odrzuca zapytania bez strony źródłowej
                c.setRequestProperty("Referer", "https://edition.cnn.com/");
                c.setRequestProperty("Origin", "https://edition.cnn.com");
            }
            int code = c.getResponseCode();
            InputStream in = code >= 400 ? c.getErrorStream() : c.getInputStream();
            ByteArrayOutputStream bo = new ByteArrayOutputStream();
            if (in != null) {
                byte[] buf = new byte[16384];
                int n;
                while ((n = in.read(buf)) > 0) bo.write(buf, 0, n);
                in.close();
            }
            cb.done(code, bo.toString(StandardCharsets.UTF_8.name()));
        } catch (Exception e) {
            cb.done(0, String.valueOf(e.getMessage()));
        } finally {
            if (c != null) c.disconnect();
        }
    }
}
