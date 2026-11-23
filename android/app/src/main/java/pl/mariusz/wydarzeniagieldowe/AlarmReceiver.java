package pl.mariusz.wydarzeniagieldowe;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;

public class AlarmReceiver extends BroadcastReceiver {
    @Override
    public void onReceive(Context c, Intent i) {
        Alarms.notify(c, i.getIntExtra("slot", 0), i.getStringExtra("title"), i.getStringExtra("body"));
    }
}
