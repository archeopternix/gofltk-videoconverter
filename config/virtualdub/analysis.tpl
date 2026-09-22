{{define "VDubFirst"}}// $job "Job {{.Num}}"
// $data ""
// $input "{{.InFile}}"
// $output ""
// $state 0
// $id c7556892247894db
// $start_time 00000000 00000000
// $end_time 00000000 00000000
// $script

VirtualDub.Open("{{.InFile}}","",0);
{{.AudioScript}}
{{.VideoScript}}
VirtualDub.SaveFormatAVI();
VirtualDub.SaveAudioFormat("");
VirtualDub.video.filters.BeginUpdate();
VirtualDub.video.filters.Clear();
VirtualDub.video.filters.Add("Deshaker v3.1");
VirtualDub.video.filters.instance[0].Config("19|1|16|4|1|0|1|0|640|480|1|2|1000|1000|1000|1000|4|1|0|2|8|30|300|4|{{.LogFile}}|0|0|0|0|0|0|0|0|0|0|0|0|0|1|15|15|5|15|1|1|30|30|0|0|0|0|1|0|0|10|1000|1|88|1|1|20|5000|100|20|1|0|ff00ff");
VirtualDub.video.filters.EndUpdate();
VirtualDub.audio.filters.Clear();
VirtualDub.video.SetRange();
VirtualDub.project.ClearTextInfo();
// -- $reloadstop --
VirtualDub.RunNullVideoPass();
VirtualDub.audio.SetSource(1);
VirtualDub.Close();

// $endjob
{{end}}
