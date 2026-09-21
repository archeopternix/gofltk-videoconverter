{{define "Avisynth"}}
# AviSynth profile: HDV
LoadPlugin("{{.AvisynthPath}}\ffms2.dll")
LoadPlugin("{{.AvisynthPath}}\mvtools2.dll")
LoadPlugin("{{.AvisynthPath}}\RgTools.dll")
LoadPlugin("{{.AvisynthPath}}\MaskTools2.dll")
LoadPlugin("{{.AvisynthPath}}\avsresize.dll")
Import("{{.AvisynthPath}}\FineDehalo.avsi")
Import("{{.AvisynthPath}}\LSFmod.avsi")
{{if .Deinterlace}}Import("{{.AvisynthPath}}\QTGMC.avsi"){{end}}

video = FFVideoSource("{{.InFile}}")
audio = FFAudioSource("{{.InFile}}")
source = AudioDub(video, audio)
{{if .ConvertYV}}converted = source.ConvertToYV12()
{{else}}converted = source
{{end}}
{{if .Deinterlace}}progressive = QTGMC(converted, Preset="{{.Preset}}")
{{else}}{{if .ConvertYV}}progressive = converted.FineDehalo()
{{else}}progressive = converted
{{end}}{{end}}
resized = progressive.z_Spline64Resize({{.ResizeX}}, {{.ResizeY}})
{{if .ConvertYV}}sharpened = resized.LSFmod(defaults="slow", screenW={{.ResizeX}}, screenH={{.ResizeY}})
return sharpened
{{else}}return resized
{{end}}
{{end}}
