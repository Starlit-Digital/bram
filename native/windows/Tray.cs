using System.Diagnostics;
using System.Text.Json.Nodes;

namespace Bram;

sealed class Tray : ApplicationContext
{
    readonly Store store;
    readonly Relay relay;
    readonly NotifyIcon icon;
    readonly ContextMenuStrip menu = new();
    readonly Control dispatcher = new();
    public bool Exiting { get; private set; }
    public Tray(Store store, Relay relay)
    {
        this.store = store; this.relay = relay; dispatcher.CreateControl();
        icon = new NotifyIcon { Icon = SystemIcons.Application, Text = "Bram", ContextMenuStrip = menu, Visible = true };
        menu.Opening += (_, _) => BuildMenu(); BuildMenu();
        icon.DoubleClick += (_, _) => OpenLog();
    }
    void BuildMenu()
    {
        menu.Items.Clear();
        menu.Items.Add("Bram on " + store.Name).Enabled = false;
        menu.Items.Add("Show Pairing Code…", null, (_, _) => ShowCode());
        var nearbyMenu = new ToolStripMenuItem("Pair with nearby device");
        foreach (var pair in relay.Discovery.Nearby().Where(p => !store.Peers().ContainsKey(p.Key)))
        {
            var id = pair.Key; var endpoint = pair.Value;
            nearbyMenu.DropDownItems.Add(id, null, async (_, _) =>
            {
                var value = Prompt("Pair with " + id, "Enter the code displayed by Bram on that device.", "", true);
                if (value == null) return;
                try { await relay.Pair(endpoint.Address.ToString(), endpoint.Port, value.Trim(), id); MessageBox.Show("Paired with " + id, "Bram"); }
                catch (Exception ex) { MessageBox.Show("Pairing failed: " + ex.Message, "Bram"); }
            });
        }
        if (nearbyMenu.DropDownItems.Count == 0) nearbyMenu.DropDownItems.Add("No unpaired devices found").Enabled = false;
        menu.Items.Add(nearbyMenu);
        var peers = new ToolStripMenuItem("Paired peers");
        var nearby = relay.Discovery.Nearby();
        foreach (var pair in store.Peers())
        {
            var peer = pair.Value!.AsObject(); peers.DropDownItems.Add(Protocol.Text(peer, "name") + " (" + pair.Key + ") • " + (nearby.ContainsKey(pair.Key) ? "online" : "offline")).Enabled = false;
        }
        if (peers.DropDownItems.Count == 0) peers.DropDownItems.Add("No paired peers").Enabled = false;
        menu.Items.Add(peers); menu.Items.Add(new ToolStripSeparator());
        var recent = new ToolStripMenuItem("Recent activity");
        var path = Path.Combine(store.Hub, "bram.log");
        foreach (var line in File.Exists(path) ? File.ReadLines(path).TakeLast(15).Reverse() : []) recent.DropDownItems.Add(line.Length > 120 ? line[..120] + "…" : line).Enabled = false;
        menu.Items.Add(recent); menu.Items.Add("Open Log…", null, (_, _) => OpenLog());
        menu.Items.Add("Show Messages Folder", null, (_, _) => Process.Start(new ProcessStartInfo("explorer.exe", store.Root) { UseShellExecute = true }));
        menu.Items.Add(new ToolStripSeparator()); menu.Items.Add("Quit Bram", null, (_, _) => RequestExit());
    }
    void OpenLog() => Process.Start(new ProcessStartInfo("notepad.exe", Path.Combine(store.Hub, "bram.log")) { UseShellExecute = true });
    void ShowCode() => Prompt("Pair another device", "Enter this code on the other device within two minutes. It can be used once.", relay.BeginPairing(), false);
    static string? Prompt(string title, string text, string value, bool editable)
    {
        using var form = new Form { Text = title, ClientSize = new Size(440, 160), FormBorderStyle = FormBorderStyle.FixedDialog, StartPosition = FormStartPosition.CenterScreen, MinimizeBox = false, MaximizeBox = false, TopMost = true, AutoScaleMode = AutoScaleMode.Dpi };
        var label = new Label { Text = text, Location = new Point(14, 14), Size = new Size(410, 45) };
        var input = new TextBox { Text = value, ReadOnly = !editable, Location = new Point(14, 64), Size = new Size(410, 30) };
        var ok = new Button { Text = editable ? "Pair" : "Done", DialogResult = DialogResult.OK, Location = new Point(338, 112), Size = new Size(86, 28) };
        var cancel = new Button { Text = "Cancel", DialogResult = DialogResult.Cancel, Location = new Point(242, 112), Size = new Size(86, 28), Visible = editable };
        form.Controls.AddRange([label, input, ok, cancel]); form.AcceptButton = ok; form.CancelButton = cancel;
        return form.ShowDialog() == DialogResult.OK ? input.Text : null;
    }
    public void RequestExit() { if (dispatcher.InvokeRequired) { dispatcher.BeginInvoke(RequestExit); return; } Exiting = true; icon.Visible = false; ExitThread(); }
    protected override void Dispose(bool disposing) { if (disposing) { icon.Dispose(); menu.Dispose(); dispatcher.Dispose(); } base.Dispose(disposing); }
}
