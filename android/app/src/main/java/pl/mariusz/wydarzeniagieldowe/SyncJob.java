package pl.mariusz.wydarzeniagieldowe;

import android.app.job.JobInfo;
import android.app.job.JobParameters;
import android.app.job.JobScheduler;
import android.app.job.JobService;
import android.content.ComponentName;
import android.content.Context;
import android.os.Handler;
import android.os.Looper;
import android.webkit.JavascriptInterface;
import android.webkit.WebSettings;
import android.webkit.WebView;

import org.json.JSONObject;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Synchronizacja w tle co N minut (JobScheduler, minimum 15 min).
 * Uruchamia UI bez okna (WebView z #bg): ta sama logika co w aplikacji pobiera dane,
 * zapisuje je w localStorage i przeplanowuje przypomnienia, po czym woła Android.bgDone().
 */
public class SyncJob extends JobService {

    static final int JOB_ID = 7001;
    private static final long TIMEOUT_MS = 120_000;

    private final Handler main = new Handler(Looper.getMainLooper());
    private final ExecutorService net = Executors.newFixedThreadPool(3);
    private WebView web;
    private JobParameters params;
    private boolean finished;

    /** minutes = 0 wyłącza. Ponowne ustawienie tego samego interwału nic nie zmienia. */
    static void schedule(Context ctx, int minutes) {
        JobScheduler js = (JobScheduler) ctx.getSystemService(Context.JOB_SCHEDULER_SERVICE);
        if (js == null) return;
        if (minutes <= 0) { js.cancel(JOB_ID); return; }
        long period = Math.max(15, minutes) * 60_000L;
        JobInfo cur = js.getPendingJob(JOB_ID);
        if (cur != null && cur.getIntervalMillis() == period) return;
        JobInfo job = new JobInfo.Builder(JOB_ID, new ComponentName(ctx, SyncJob.class))
                .setRequiredNetworkType(JobInfo.NETWORK_TYPE_ANY)
                .setPeriodic(period)
                .setPersisted(true)
                .build();
        js.schedule(job);
    }

    @Override
    public boolean onStartJob(JobParameters p) {
        if (MainActivity.visible) return false;   // aplikacja otwarta: synchronizuje sama
        params = p;
        finished = false;
        Alarms.channels(this);
        try {
            web = new WebView(getApplicationContext());
            WebSettings s = web.getSettings();
            s.setJavaScriptEnabled(true);
            s.setDomStorageEnabled(true);
            s.setAllowFileAccess(true);
            web.addJavascriptInterface(new BgBridge(), "Android");
            web.loadUrl("file:///android_asset/index.html#bg");
        } catch (Exception e) {
            return false;
        }
        main.postDelayed(() -> finish(true), TIMEOUT_MS);
        return true;
    }

    @Override
    public boolean onStopJob(JobParameters p) {
        cleanup();
        return true;   // spróbuj ponownie później
    }

    private void finish(boolean retry) {
        if (finished) return;
        finished = true;
        cleanup();
        if (params != null) jobFinished(params, retry);
    }

    private void cleanup() {
        main.removeCallbacksAndMessages(null);
        if (web != null) { try { web.destroy(); } catch (Exception ignored) { } web = null; }
    }

    private void deliver(int id, int status, String body) {
        final String js = "window.onHttp&&window.onHttp(" + id + "," + status + "," + JSONObject.quote(body == null ? "" : body) + ")";
        main.post(() -> { if (web != null) web.evaluateJavascript(js, null); });
    }

    /** Podzbiór mostu z MainActivity potrzebny do synchronizacji. */
    class BgBridge {
        @JavascriptInterface public void http(final int id, final String url) { net.execute(() -> Http.get(url, (st, b) -> deliver(id, st, b))); }
        @JavascriptInterface public void schedule(String json) { Alarms.scheduleAll(SyncJob.this, json); }
        @JavascriptInterface public void bgDone() { main.post(() -> finish(false)); }
        @JavascriptInterface public void setBgSync(int minutes) { }
        @JavascriptInterface public String notifyState() { return "granted"; }
        @JavascriptInterface public void askNotify() { }
        @JavascriptInterface public void test(String t, String b) { }
        @JavascriptInterface public void toCalendar(String a, double b, double c, String d, String e) { }
        @JavascriptInterface public String saveFile(String a, String b, String c) { return "fail"; }
    }
}
