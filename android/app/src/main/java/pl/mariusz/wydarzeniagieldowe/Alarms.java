package pl.mariusz.wydarzeniagieldowe;

import android.app.AlarmManager;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.os.Build;

import org.json.JSONArray;
import org.json.JSONObject;

/** Planowanie przypomnień. Strona decyduje CO i KIEDY, ta klasa tylko wykonuje. */
final class Alarms {

    static final String PREFS = "wg-native";
    static final String CH_REMIND = "remind";
    static final int MAX_SLOTS = 150;

    private Alarms() { }

    static SharedPreferences prefs(Context c) {
        return c.getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }

    static void channels(Context c) {
        NotificationManager nm = c.getSystemService(NotificationManager.class);
        if (nm == null) return;
        NotificationChannel r = new NotificationChannel(CH_REMIND, "Przypomnienia o wydarzeniach giełdowych", NotificationManager.IMPORTANCE_HIGH);
        r.setDescription("Dane makro, decyzje banków, wygasanie, święta");
        nm.createNotificationChannel(r);
    }

    static void scheduleAll(Context c, String json) {
        prefs(c).edit().putString("alarms", json == null ? "[]" : json).apply();
        AlarmManager am = (AlarmManager) c.getSystemService(Context.ALARM_SERVICE);
        if (am == null) return;
        for (int i = 0; i < MAX_SLOTS; i++) am.cancel(pending(c, i, null));
        try {
            JSONArray arr = new JSONArray(json == null ? "[]" : json);
            long now = System.currentTimeMillis();
            int slot = 0;
            for (int i = 0; i < arr.length() && slot < MAX_SLOTS; i++) {
                JSONObject a = arr.getJSONObject(i);
                long at = a.getLong("at");
                if (at <= now) continue;
                PendingIntent pi = pending(c, slot++, a);
                if (Build.VERSION.SDK_INT >= 31 && !am.canScheduleExactAlarms()) {
                    am.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pi);
                } else {
                    am.setExactAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pi);
                }
            }
        } catch (Exception ignored) { }
    }

    private static PendingIntent pending(Context c, int slot, JSONObject a) {
        Intent i = new Intent(c, AlarmReceiver.class);
        i.setAction("wg.alarm." + slot);
        if (a != null) {
            i.putExtra("title", a.optString("title"));
            i.putExtra("body", a.optString("body"));
            i.putExtra("slot", slot);
        }
        return PendingIntent.getBroadcast(c, slot, i, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
    }

    static void notify(Context c, int slot, String title, String body) {
        channels(c);
        NotificationManager nm = (NotificationManager) c.getSystemService(Context.NOTIFICATION_SERVICE);
        if (nm == null) return;
        Intent open = new Intent(c, MainActivity.class).setFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        PendingIntent pi = PendingIntent.getActivity(c, 100 + slot, open, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
        Notification n = new Notification.Builder(c, CH_REMIND)
                .setSmallIcon(R.drawable.ic_stat)
                .setColor(0xFF4C9BFF)
                .setContentTitle(title)
                .setContentText(body)
                .setStyle(new Notification.BigTextStyle().bigText(body))
                .setAutoCancel(true)
                .setContentIntent(pi)
                .build();
        nm.notify(1000 + slot, n);
    }

    /** Po restarcie telefonu / aktualizacji apki odtwarza ostatnio zlecone przypomnienia. */
    static void restore(Context c) {
        scheduleAll(c, prefs(c).getString("alarms", "[]"));
    }
}
