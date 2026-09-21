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
VirtualDub.audio.SetSource(1);
VirtualDub.audio.SetMode(1);
VirtualDub.audio.SetInterleave(1,500,1,0,0);
VirtualDub.audio.SetClipMode(1,1);
VirtualDub.audio.SetEditMode(1);
VirtualDub.audio.SetConversion(0,0,0,0,0);
VirtualDub.audio.SetVolume();
{{.AudioScript}}
VirtualDub.audio.EnableFilterGraph(0);
VirtualDub.video.SetInputFormat(0);
VirtualDub.video.SetOutputFormat(7);
VirtualDub.video.SetMode(3);
VirtualDub.video.SetSmartRendering(0);
VirtualDub.video.SetPreserveEmptyFrames(0);
VirtualDub.video.SetFrameRate2(0,0,1);
VirtualDub.video.SetIVTC(0,0,0,0);
{{.VideoScript}}
{{.ContainerScript}}
{{.AnalysisScript}}
VirtualDub.audio.filters.Clear();
VirtualDub.video.SetRange();
VirtualDub.project.ClearTextInfo();
// -- $reloadstop --
VirtualDub.RunNullVideoPass();
VirtualDub.audio.SetSource(1);
VirtualDub.Close();

// $endjob
{{end}}
