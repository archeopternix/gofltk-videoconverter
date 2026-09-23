{{define "Avisynth"}}
# AviSynth profile: DV
LoadPlugin("{{.AvisynthPath}}\ffms2.dll")
LoadPlugin("{{.AvisynthPath}}\mvtools2.dll")
LoadPlugin("{{.AvisynthPath}}\RgTools.dll")
LoadPlugin("{{.AvisynthPath}}\MaskTools2.dll")
LoadPlugin("{{.AvisynthPath}}\avsresize.dll")
Import("{{.AvisynthPath}}\QTGMC.avsi")

video = FFVideoSource("{{.InFile}}")
audio = FFAudioSource("{{.InFile}}")
source = AudioDub(video, audio)
{{if .ConvertYV}}converted = source.ConvertToYV12()
{{else}}converted = source
{{end}}
progressive = QTGMC(converted, Preset="{{.Preset}}")
cropped = progressive.Crop(0, 2, 0, -2)
resized = cropped.z_Spline64Resize({{.ResizeX}}, {{.ResizeY}})
return resized
{{end}}
