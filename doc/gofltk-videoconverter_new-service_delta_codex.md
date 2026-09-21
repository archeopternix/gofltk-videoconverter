# Delta-Spezifikation für Codex – `gofltk-videoconverter` / Branch `new-service`

**Repository:** `archeopternix/gofltk-videoconverter`  
**Branch:** `new-service`  
**Analysierter Stand:** `eea97c7e98c7ee89fc97d0a363996d7c9e7750cc`  
**Zweck:** Diese Spezifikation beschreibt ausschließlich die Änderungen gegenüber dem aktuellen Stand des Branches `new-service`.

---

## 1. Zielbild

Die bestehende Architektur bleibt grundsätzlich erhalten:

```text
Input files
    ↓
ffprobe / source identification
    ↓
workflow selection
    ↓
AviSynth preparation per file
    ↓
ONE VirtualDub2 batch for all applicable files
    ↓
FFmpeg zscale per file, falls im Workflow konfiguriert
    ↓
final output verification
    ↓
temporary-file cleanup
```

Wesentliche neue Architekturregel:

```text
Skalierung findet niemals vor dem Deshaker statt.

Wenn Skalierung benötigt wird:

AviSynth
    ↓
VirtualDub2 / Deshaker
    ↓
temporäres verlustarmes Video
    ↓
FFmpeg zscale
    ↓
finales Output-Video
```

Damit wird `scale.avs.tpl` nicht mehr für die eigentliche Skalierung verwendet.

---

# 2. AviSynth: DLL-/Import-Pfade und YV12

## 2.1 Neue App-Konfiguration

`medialang/config/config.go` erweitern.

Empfohlene Felder:

```go
type Paths struct {
    Workflows        string `yaml:"workflows"`
    AviSynthProfiles string `yaml:"avisynth_profiles"`

    AviSynthDLLs     string `yaml:"avisynth_dlls"`
    AviSynthImports  string `yaml:"avisynth_imports"`

    VirtualDubCodecs string `yaml:"virtualdub_codecs"`
    Deshaker         string `yaml:"deshaker"`
}
```

Beide neuen Pfade werden in `App.resolvePaths()` relativ zum Verzeichnis der jeweiligen `app.xx.yaml` aufgelöst.

Beispiel Windows:

```yaml
paths:
  workflows: workflows
  avisynth_profiles: avisynth/profiles

  avisynth_dlls: C:/Program Files/AviSynth+/plugins64+
  avisynth_imports: C:/Program Files/AviSynth+/scripts

  virtualdub_codecs: virtualdub/codecs
  deshaker: virtualdub/deshaker.yaml
```

Beispiel Linux/Wine:

```yaml
paths:
  avisynth_dlls: /opt/avisynth/plugins64+
  avisynth_imports: /opt/avisynth/scripts
```

Da die erzeugten `.avs`-Dateien von VirtualDub/AviSynth unter Windows bzw. Wine interpretiert werden, müssen zusätzlich Windows-Pfade über den vorhandenen `PathConverter` erzeugt werden.

---

## 2.2 AviSynth Compiler erweitern

`medialang/engine/avisynth/compiler.go`

`Compiler` erweitern:

```go
type Compiler struct {
    ProfilesDir string

    DLLDir      string
    ImportDir   string

    Pather      WindowsPather
}
```

Beim Kompilieren folgende Werte bereitstellen:

```text
InputPath
InputPathWindows

DLLPath
DLLPathWindows

ImportPath
ImportPathWindows

Width
Height
FPS
FieldOrder
PixelFormat
ColorSpace
ColorFamily

IsYV12
EnsureYV12

+ alle Werte aus profile.Config.Values
```

---

## 2.3 Template-Engine

Die aktuelle Implementierung arbeitet mit `strings.ReplaceAll`.

Diese soll auf `text/template` umgestellt werden.

Grund:

- boolesche Werte wie `IsYV12`
- conditionals
- spätere Erweiterbarkeit
- zuverlässige Erkennung unbekannter Platzhalter

Empfohlen:

```go
template.New(...).
    Option("missingkey=error").
    Parse(...)
```

Damit führen unbekannte bzw. nicht vorhandene Template-Werte zu einem Fehler.

---

## 2.4 YV12-Erkennung

`media.MediaSpec` erweitern:

```go
type MediaSpec struct {
    ...
    IsYV12 bool
}
```

Die Information wird bereits beim `ffprobe` Mapping gesetzt.

Für die erste Implementierung gilt:

```text
yuv420p   → IsYV12 = true
yuvj420p  → IsYV12 = true

alle anderen pix_fmt → IsYV12 = false
```

Insbesondere:

```text
nv12
yuv422p
yuv444p
rgb24
gbrp
yuv420p10le
```

sind **nicht** als YV12 zu behandeln.

Wichtig: `ColorFamily == "yuv"` reicht ausdrücklich nicht aus.

---

## 2.5 Automatische YV12-Konvertierung

Der Compiler liefert zusätzlich den Template-Wert:

```text
EnsureYV12
```

Semantik:

```text
IsYV12 == true
    → ""

IsYV12 == false
    → "ConvertToYV12()"
```

Alle AviSynth-Templates, die Filter verwenden, welche YV12 voraussetzen, müssen direkt nach dem Laden des Quellclips enthalten:

```avs
{{.EnsureYV12}}
```

Beispiel:

```avs
LoadPlugin("{{.DLLPathWindows}}\ffms2.dll")
Import("{{.ImportPathWindows}}\QTGMC.avsi")

clip = FFVideoSource("{{.InputPathWindows}}")
clip = clip.{{.EnsureYV12}}
```

Da bei leerem `EnsureYV12` obige Punkt-Syntax ungeeignet wäre, soll der Platzhalter vorzugsweise als vollständige Script-Zeile erzeugt werden.

Empfohlene konkrete Semantik:

```text
IsYV12 == true
    → "# source already YV12"

IsYV12 == false
    → "ConvertToYV12()"
```

und im Template:

```avs
{{.EnsureYV12}}
```

Der Platzhalter steht dabei in der normalen AviSynth-Filterkette.

Alternativ darf ein Template selbst mit

```gotemplate
{{if not .IsYV12}}
ConvertToYV12()
{{end}}
```

arbeiten.

Beide Werte (`IsYV12` und `EnsureYV12`) sollen angeboten werden.

---

# 3. Neue FFmpeg Engine

## 3.1 Engine Type

`medialang/engine/type.go`

ergänzen:

```go
const (
    AviSynth   Type = "avisynth"
    VirtualDub Type = "virtualdub"
    FFmpeg     Type = "ffmpeg"
)
```

---

## 3.2 FFmpeg Tool-Konfiguration

`medialang/config/config.go`

ergänzen:

```go
type Tools struct {
    FFprobe    FFprobeTool    `yaml:"ffprobe"`
    FFmpeg     FFmpegTool     `yaml:"ffmpeg"`
    VirtualDub VirtualDubTool `yaml:"virtualdub"`
}

type FFmpegTool struct {
    Path string `yaml:"path"`
}
```

Beispiel:

```yaml
tools:
  ffprobe:
    path: C:/Program Files/ffmpeg/bin/ffprobe.exe

  ffmpeg:
    path: C:/Program Files/ffmpeg/bin/ffmpeg.exe
```

Unter Linux:

```yaml
tools:
  ffmpeg:
    path: ffmpeg
```

---

## 3.3 Neuer Filter

Neuer Filtertyp:

```text
ffmpeg.zscale
```

In `filter.DefaultRegistry()` registrieren.

Empfohlene Config:

```go
type ZScaleConfig struct {
    Width       int      `yaml:"width"`
    Height      int      `yaml:"height"`

    Filter      string   `yaml:"filter"`
    PixelFormat string   `yaml:"pixel_format"`

    VideoCodec  string   `yaml:"video_codec"`
    CRF         *int     `yaml:"crf,omitempty"`
    Preset      string   `yaml:"preset,omitempty"`

    AudioCodec  string   `yaml:"audio_codec"`
    AudioBitrate string  `yaml:"audio_bitrate,omitempty"`

    Extension   string   `yaml:"extension"`

    ExtraArgs   []string `yaml:"extra_args,omitempty"`
}
```

Minimalbeispiel:

```yaml
- id: final-scale
  filter: ffmpeg.zscale
  config:
    width: 1920
    height: 1080
    filter: spline36
    pixel_format: yuv420p

    video_codec: libx264
    crf: 18
    preset: slow

    audio_codec: aac
    audio_bitrate: 192k

    extension: .mp4
```

Defaults:

```text
filter       = spline36
pixel_format = yuv420p
```

`width`, `height`, `video_codec`, `audio_codec` und `extension` sind Muss-Felder.

---

## 3.4 FFmpeg Package

Neu:

```text
medialang/engine/ffmpeg/
    runner.go
```

Schnittstelle ungefähr:

```go
type Runner struct {
    Executable string
}

func (r Runner) Run(
    ctx context.Context,
    input string,
    output string,
    config filter.ZScaleConfig,
) error
```

FFmpeg-Aufruf sinngemäß:

```bash
ffmpeg -y \
  -i INPUT \
  -vf "zscale=w=WIDTH:h=HEIGHT:filter=spline36,format=yuv420p" \
  -c:v libx264 \
  -crf 18 \
  -preset slow \
  -c:a aac \
  -b:a 192k \
  OUTPUT
```

Argumente **nicht** über Shell-String zusammenbauen.

Immer:

```go
exec.CommandContext(ctx, executable, args...)
```

verwenden.

---

# 4. Workflow-Reihenfolge

Der Planner/Runner muss künftig drei Engine-Bereiche unterstützen:

```text
AviSynth
    ↓
VirtualDub
    ↓
FFmpeg
```

Zulässige Reihenfolge:

```text
0..1 AviSynth block
0..1 VirtualDub block
0..1 FFmpeg block
```

Nach einem FFmpeg-Filter darf kein AviSynth- oder VirtualDub-Filter mehr folgen.

Für die erste Version gilt:

```text
ffmpeg.zscale muss der letzte aktive Filter eines Workflows sein.
```

Ungültig:

```text
ffmpeg.zscale
virtualdub.encode
```

Ungültig:

```text
virtualdub.deshaker
ffmpeg.zscale
avisynth.profile
```

---

# 5. VirtualDub wird Intermediate-Stage

Aktuell schreibt `virtualdub.encode` direkt nach `file.output`.

Das muss geändert werden, wenn danach `ffmpeg.zscale` folgt.

## 5.1 Ohne FFmpeg

Bestehendes Verhalten bleibt möglich:

```text
VirtualDub → final output
```

## 5.2 Mit FFmpeg

Wenn ein Workflow einen nachfolgenden `ffmpeg.zscale` Filter enthält:

```text
VirtualDub → temporary intermediate
FFmpeg     → final output
```

VirtualDub-Intermediate:

```text
<work_dir>/<run-id>/virtualdub/output/
    0001-<basename>.avi
    0002-<basename>.avi
    ...
```

Für das Intermediate ist ein verlustarmer Codec zu verwenden.

Vorhandener Preset:

```text
huffyuv
```

kann dafür verwendet werden.

Die Wahl soll jedoch weiterhin über die Workflow-Konfiguration erfolgen und nicht im Go-Code fest verdrahtet werden.

---

# 6. Empfohlenes Workflow-Beispiel

Beispiel HDV:

```yaml
version: 1
id: hdv-25i
name: HDV 1080i25
priority: 100

match:
  interlaced: true
  width: 1440
  height: 1080
  fps: 25
  fps_tolerance: 0.1
  codec: [mpeg2video]

workflow:
  - id: prepare
    filter: avisynth.profile
    config:
      profile: hdv-25i

  - id: stabilize
    filter: virtualdub.deshaker
    config:
      reuse_analysis: true
      force_analysis: false

  - id: intermediate
    filter: virtualdub.encode
    config:
      preset: huffyuv

  - id: final-scale
    filter: ffmpeg.zscale
    config:
      width: 1920
      height: 1080
      filter: spline36
      pixel_format: yuv420p
      video_codec: libx264
      crf: 18
      preset: slow
      audio_codec: aac
      audio_bitrate: 192k
      extension: .mp4
```

Wichtig:

```text
zscale ist der letzte Verarbeitungsschritt.
```

---

# 7. VirtualDub2: fortlaufende Dateinummer

`medialang/engine/virtualdub/jobs.go`

Die vorhandene Struktur besitzt bereits:

```go
FileIndex int
```

Dieser Wert wird derzeit aber nicht in `values` für die Templates aufgenommen.

Zusätzlich wird eine dichte, fortlaufende Dateinummer benötigt.

## 7.1 Definition

Zwei unterschiedliche Werte bereitstellen:

```text
FileIndex
    0-basierter Index in Runner.Files
    bleibt stabil
    kann Lücken enthalten, wenn Dateien übersprungen werden

FileNumber
    1-basierte fortlaufende Nummer innerhalb des tatsächlich erzeugten
    VirtualDub-Batches
    immer 1..N ohne Lücken
```

Beispiel:

```text
Runner.Files:
0 fileA
1 fileB -> skipped
2 fileC

VirtualDub jobs:

fileA:
  FileIndex  = 0
  FileNumber = 1

fileC:
  FileIndex  = 2
  FileNumber = 2
```

---

## 7.2 VirtualDub Template-Werte

`JobsBuilder.Write()` ergänzen:

```go
for batchIndex, job := range jobs {
    values := map[string]string{
        "InputPath":   ...,
        "OutputPath":  ...,
        "LogPath":     ...,

        "FileIndex":   strconv.Itoa(job.FileIndex),
        "FileNumber":  strconv.Itoa(batchIndex + 1),
    }
}
```

Damit stehen in:

```text
deshaker.yaml
codec templates
```

folgende Platzhalter zur Verfügung:

```text
{{.InputPath}}
{{.OutputPath}}
{{.LogPath}}
{{.FileIndex}}
{{.FileNumber}}
```

---

## 7.3 JobNumber nicht mit FileNumber verwechseln

VirtualDub `$job` Nummern sind nicht identisch mit der Dateinummer.

Eine Datei kann wegen Deshaker zwei Jobs erzeugen:

```text
FileNumber 1
    VirtualDub Job 1 = analysis
    VirtualDub Job 2 = render

FileNumber 2
    VirtualDub Job 3 = analysis
    VirtualDub Job 4 = render
```

Optional zusätzlich als Template-Wert:

```text
{{.JobNumber}}
```

wenn dies später benötigt wird.

Die neue Muss-Anforderung ist jedoch:

```text
{{.FileNumber}}
```

---

# 8. Temporäre Dateien löschen

Im aktuellen Code existiert bereits:

```yaml
processing:
  keep_temp_files: true
```

die Option wird aktuell jedoch nicht vollständig für Cleanup umgesetzt.

## 8.1 Neue Semantik

Default:

```yaml
processing:
  keep_temp_files: false
```

Nach dem letzten Media-Step eines Runs:

```text
keep_temp_files == false
    → komplettes run-spezifisches work directory löschen

keep_temp_files == true
    → work directory erhalten
```

Run directory:

```text
<processing.work_dir>/<run-id>/
```

Zu löschende Dateien sind damit automatisch enthalten:

```text
generated .avs
VirtualDub .jobs
VirtualDub intermediate videos
FFmpeg temporary files
sonstige run-lokale temporäre Dateien
```

Nicht löschen:

```text
final output files
Deshaker-Logs, wenn diese bewusst im output_dir für reuse_analysis liegen
Quellmaterial
Konfigurationsdateien
```

---

## 8.2 Cleanup-Verhalten bei Fehlern

Die Option soll deterministisch gelten:

```text
keep_temp_files = false
    → auch bei failed/cancelled Run aufräumen

keep_temp_files = true
    → auch bei Fehlern behalten
```

Damit entscheidet ausschließlich die Konfiguration.

Cleanup soll über `defer` direkt nach erfolgreichem Erzeugen des run-spezifischen Workdirs registriert werden.

Beispiel:

```go
if !r.Config.Processing.KeepFiles {
    defer func() {
        if err := os.RemoveAll(workDir); err != nil {
            r.Logger.Warn("cleanup work directory failed", ...)
        }
    }()
}
```

Cleanup-Fehler dürfen einen bereits erfolgreich verarbeiteten Batch nicht nachträglich auf `failed` setzen, müssen aber geloggt werden.

---

# 9. Runner-Anpassung

`medialang/runner.go`

Der aktuelle Ablauf:

```text
plan
AviSynth
VirtualDub
verify final output
```

wird zu:

```text
plan all files
AviSynth preparation
create VirtualDub jobs
run VirtualDub once
map VirtualDub outputs back to current artifact
run FFmpeg stage per applicable file
verify final outputs
cleanup
```

`plannedFile` ergänzen:

```go
type plannedFile struct {
    ...

    ffmpegScale *filter.ZScale

    virtualDubOutput string
    finalOutput      string

    current media.Artifact
}
```

Semantik:

```text
file.current
```

ist nach jedem physischen Stage der tatsächliche Input des folgenden Stages.

Nach AviSynth:

```text
current = .avs
```

Nach VirtualDub:

```text
current = intermediate video
```

Nach FFmpeg:

```text
current = final video
```

---

# 10. Output Naming

Wenn `ffmpeg.zscale` vorhanden ist:

```text
final extension = ffmpeg.zscale.config.extension
```

Final:

```text
<output_dir>/<original-basename>_processed<extension>
```

VirtualDub schreibt dann nicht nach `finalOutput`, sondern nach:

```text
<work_dir>/<run-id>/virtualdub/output/<nnnn>-<basename><vdub-extension>
```

Wenn kein FFmpeg-Filter vorhanden ist, bleibt die bisherige VirtualDub-Ausgabe final.

---

# 11. AviSynth README

Neu:

```text
config/avisynth/README.md
```

Die README muss mindestens enthalten:

## Verzeichnisstruktur

```text
config/avisynth/
    README.md
    profiles/
        *.avs.tpl
```

## Verfügbare Standard-Platzhalter

```text
{{.InputPath}}
{{.InputPathWindows}}

{{.DLLPath}}
{{.DLLPathWindows}}

{{.ImportPath}}
{{.ImportPathWindows}}

{{.Width}}
{{.Height}}
{{.FPS}}
{{.FieldOrder}}

{{.PixelFormat}}
{{.ColorSpace}}
{{.ColorFamily}}

{{.IsYV12}}
{{.EnsureYV12}}
```

Zusätzlich:

```text
config.values
```

aus dem Workflow werden als weitere Template-Variablen angeboten.

Beispiel:

```yaml
config:
  profile: hdv-25i
  values:
    QTGMC_PRESET: Slower
```

Template:

```avs
QTGMC(Preset="{{.QTGMC_PRESET}}")
```

README muss klar festhalten:

```text
Unbekannte Template-Variablen sind Fehler.
```

und:

```text
AviSynth-Templates dürfen keine Skalierung enthalten,
wenn die Skalierung nach Deshaker über ffmpeg.zscale erfolgen soll.
```

---

# 12. VirtualDub README

Neu:

```text
config/virtualdub/README.md
```

Dokumentieren:

```text
config/virtualdub/
    README.md
    deshaker.yaml
    codecs/
        *.yaml
```

## Deshaker-Platzhalter

```text
{{.InputPath}}
{{.OutputPath}}
{{.LogPath}}
{{.FileIndex}}
{{.FileNumber}}
```

## Codec-Platzhalter

```text
{{.InputPath}}
{{.OutputPath}}
{{.LogPath}}
{{.FileIndex}}
{{.FileNumber}}
```

## Semantik

Dokumentieren:

```text
FileIndex  = ursprünglicher 0-basierter Dateindex
FileNumber = dichte 1-basierte Nummer im VirtualDub-Batch
```

sowie:

```text
VirtualDub2 wird pro Runner.Run() maximal einmal gestartet.
```

und:

```text
Wenn danach ein FFmpeg-Filter folgt, ist OutputPath ein temporäres Intermediate.
```

---

# 13. Workflow README

Im Repository heißt das Verzeichnis aktuell:

```text
config/workflows/
```

Daher dort anlegen:

```text
config/workflows/README.md
```

README beschreibt:

## Muss-Felder Workflow

```yaml
version:
id:
name:
priority:
match:
workflow:
```

## Filter

```text
avisynth.profile
virtualdub.deshaker
virtualdub.encode
ffmpeg.zscale
```

## Engine Ordering

```text
AviSynth → VirtualDub → FFmpeg
```

## Spezialregel

```text
ffmpeg.zscale muss letzter aktiver Filter sein.
```

## Beispiel vollständiger Workflow

Ein vollständiges Beispiel wie in Kapitel 6 aufnehmen.

## Match Attribute

Aktuell:

```text
interlaced
width
height
fps
fps_tolerance
pixel_format
color_space
codec
```

README soll Muss-/Optional-Charakter dokumentieren.

`match` selbst ist Pflicht.

Einzelne Match-Attribute sind optional.

---

# 14. Haupt-README für `config/`

Neu:

```text
config/README.md
```

Diese Datei dokumentiert `app.yaml`, `app.windows.yaml` bzw. zukünftige `app.<platform>.yaml`.

---

## 14.1 Struktur

```yaml
tools:
paths:
processing:
```

---

## 14.2 `tools.ffprobe`

```yaml
tools:
  ffprobe:
    path: ...
```

### `path`

**Muss**

Executable oder über PATH auflösbarer Name.

---

## 14.3 `tools.ffmpeg`

```yaml
tools:
  ffmpeg:
    path: ...
```

### `path`

**Bedingt Muss**

Pflicht, sobald mindestens ein Workflow einen `ffmpeg.*` Filter enthält.

---

## 14.4 `tools.virtualdub`

```yaml
tools:
  virtualdub:
    executable: ...
    arguments:
      - ...
    path_mode: windows
    wine_drive: "Z:"
    path_mappings: []
    env: {}
```

### executable

**Bedingt Muss**

Pflicht, sobald ein Workflow einen `virtualdub.*` Filter enthält.

### arguments

**Optional**, Default:

```text
[]
```

Für die tatsächliche Batch-Ausführung muss die Konfiguration jedoch den Jobs-File-Platzhalter unterstützen:

```text
{{.JobsFileWindows}}
```

### path_mode

**Optional**

Erlaubte Werte:

```text
windows
wine
```

Default:

```text
windows
```

### wine_drive

**Optional**

Nur bei:

```text
path_mode: wine
```

Default:

```text
Z:
```

### path_mappings

**Optional**

Mapping:

```yaml
- source: /mnt/media
  target: M:/media
```

### env

**Optional**

Zusätzliche Umgebungsvariablen für VirtualDub.

---

# 15. `paths` in app.xx

## workflows

```yaml
paths:
  workflows: workflows
```

**Muss**

---

## avisynth_profiles

**Bedingt Muss**

Pflicht, wenn `avisynth.profile` verwendet wird.

---

## avisynth_dlls

**Bedingt Muss**

Pflicht, wenn AviSynth-Templates DLL-Pfade benötigen.

Dieser Wert wird als:

```text
DLLPath
DLLPathWindows
```

an Templates übergeben.

---

## avisynth_imports

**Bedingt Muss**

Pflicht, wenn AviSynth-Templates Import-Scripte verwenden.

Dieser Wert wird als:

```text
ImportPath
ImportPathWindows
```

an Templates übergeben.

---

## virtualdub_codecs

**Bedingt Muss**

Pflicht bei `virtualdub.encode`.

---

## deshaker

**Bedingt Muss**

Pflicht bei `virtualdub.deshaker`.

---

# 16. `processing` in app.xx

## output_dir

**Muss**

Finale Benutzeroutputs.

---

## work_dir

**Muss**

Run-spezifische temporäre Dateien.

---

## keep_temp_files

**Optional**

Default:

```yaml
false
```

Semantik:

```text
false → Run-Verzeichnis nach Abschluss/Fehler löschen
true  → Run-Verzeichnis behalten
```

---

# 17. Config Validation

`config.Load()` soll nicht mehr nur YAML parsen und Pfade auflösen.

Nach dem Laden muss eine Basisvalidierung erfolgen.

Mindestens:

```text
tools.ffprobe.path != ""

paths.workflows != ""
processing.output_dir != ""
processing.work_dir != ""

virtualdub.path_mode ∈ {"", "windows", "wine"}
```

Workflow-abhängige Validation erfolgt beim Laden/Planen:

```text
avisynth.profile
    → avisynth_profiles vorhanden

virtualdub.deshaker
    → virtualdub executable + deshaker config vorhanden

virtualdub.encode
    → virtualdub executable + codec directory vorhanden

ffmpeg.zscale
    → tools.ffmpeg.path vorhanden
```

Fehlermeldungen müssen den konkreten Config-Key enthalten.

Beispiel:

```text
workflow "hdv-25i" uses ffmpeg.zscale but tools.ffmpeg.path is empty
```

---

# 18. Bestehendes `scale.avs.tpl`

`config/avisynth/profiles/scale.avs.tpl`

soll nicht mehr als aktiver Skalierungsweg verwendet werden.

Optionen:

1. Datei entfernen, sofern sie nirgends mehr benötigt wird.
2. Oder behalten und in `config/avisynth/README.md` als deprecated kennzeichnen.

Empfehlung:

```text
entfernen
```

sobald alle Workflows auf `ffmpeg.zscale` migriert sind.


---

# 19. Betroffene Dateien

Bestehend ändern:

```text
medialang/config/config.go
medialang/engine/type.go
medialang/engine/avisynth/compiler.go
medialang/engine/virtualdub/jobs.go
medialang/filter/filter.go
medialang/filter/registry.go
medialang/media/media.go
medialang/probe/ffprobe.go
medialang/runner.go

config/app.yaml
config/app.windows.yaml

config/workflows/*.yaml
config/avisynth/profiles/*.avs.tpl
```

Neu:

```text
medialang/engine/ffmpeg/runner.go

config/README.md
config/avisynth/README.md
config/virtualdub/README.md
config/workflows/README.md
```


---

# 21. Akzeptanzkriterien / Definition of Done

Die Änderung ist fertig, wenn:

- [ ] AviSynth-Templates Zugriff auf DLL- und Import-Pfade haben.
- [ ] Sowohl Host- als auch Windows/Wine-Pfade verfügbar sind.
- [ ] `MediaSpec` explizit angibt, ob das Material YV12-kompatibel ist.
- [ ] Nicht-YV12-Material erhält vor YV12-abhängigen AviSynth-Filtern eine Konvertierung ion AviSynth.
- [ ] AviSynth-Templates werden mit `text/template` und `missingkey=error` ausgewertet.
- [ ] `engine.FFmpeg` existiert.
- [ ] `ffmpeg.zscale` als Filter registriert ist.
- [ ] Skalierung erst **nach** VirtualDub/Deshaker stattfindet.
- [ ] `ffmpeg.zscale` der letzte aktive Filter eines Workflows ist.
- [ ] VirtualDub bei nachfolgendem FFmpeg nur ein temporäres Intermediate erzeugt.
- [ ] VirtualDub weiterhin nur einmal pro `Runner.Run()` gestartet wird.
- [ ] FFmpeg pro zu skalierender Datei auf dem vorherigen VirtualDub-Output arbeitet.
- [ ] VirtualDub-Templates `{{.FileNumber}}` erhalten.
- [ ] `FileNumber` 1-basiert und lückenlos über die tatsächlich verarbeiteten VDub-Dateien läuft.
- [ ] `FileIndex` weiterhin den ursprünglichen Input-Index enthält.
- [ ] Bei `keep_temp_files: false` das Run-Workdir nach dem letzten Schritt entfernt wird.
- [ ] Finale Outputs und wiederverwendbare Deshaker-Logs nicht gelöscht werden.
- [ ] `config/README.md` alle Muss-, bedingt notwendigen und optionalen `app.xx` Attribute dokumentiert.
- [ ] `config/avisynth/README.md` alle Template-Platzhalter dokumentiert.
- [ ] `config/virtualdub/README.md` alle Template-Platzhalter inklusive `FileNumber` dokumentiert.
- [ ] `config/workflows/README.md` Workflow-Schema und Engine-Reihenfolge dokumentiert.

---

# 22. Wichtigste neue Invarianten

Codex soll folgende Regeln als harte Architekturregeln behandeln:

```text
1. No media processing before Runner.Run().

2. The logical output of one filter is the logical input of the next filter.

3. All input files are identified and planned before execution begins.

4. AviSynth runs before VirtualDub.

5. Scaling with zscale runs after VirtualDub/Deshaker.

6. FFmpeg is the last engine in a workflow.

7. VirtualDub2 is started at most once per Runner.Run().

8. A VirtualDub batch receives a dense 1-based FileNumber for every
   actually processed file.

9. Temporary artifacts belong below:
      <work_dir>/<run-id>/

10. Final outputs never belong below the temporary run directory.

11. If keep_temp_files == false, the run directory is removed after the
    run has finished, failed or been cancelled.

12. YUV color family does not imply YV12.
    Only explicitly recognized 8-bit planar 4:2:0 formats are YV12-compatible.
```

---

# 23. Empfohlene Implementierungsreihenfolge für Codex

```text
1. Config-Strukturen erweitern:
   ffmpeg + avisynth_dlls + avisynth_imports

2. config/README.md erstellen.

3. MediaSpec um IsYV12 ergänzen und ffprobe mapping implementieren.

4. AviSynth Compiler auf text/template umstellen.

5. AviSynth TemplateContext mit DLL/import/YV12 erweitern.

6. config/avisynth/README.md erstellen und Templates anpassen.

7. engine.FFmpeg ergänzen.

8. ffmpeg.zscale Filter + Registry ergänzen.

9. FFmpeg Runner implementieren.

10. Planner/Runner auf:
      AviSynth → VirtualDub → FFmpeg
    erweitern.

11. VirtualDub final-vs-intermediate Output trennen.

12. FileNumber in VirtualDub JobsBuilder ergänzen.

13. config/virtualdub/README.md ergänzen.

14. Workflow-YAMLs auf neue Reihenfolge migrieren.

15. config/workflows/README.md erstellen.

16. Cleanup des Run-Verzeichnisses implementieren.

```

---

# 24. Nicht Teil dieses Deltas

Nicht unnötig umbauen:

```text
GUI
ffprobe-Grundarchitektur
Workflow-Matching-Grundprinzip
VirtualDub PathConverter
VirtualDub ONE-BATCH-Prinzip
Deshaker reuse_analysis Mechanismus
bestehende Result-/Status-Typen, soweit keine Erweiterung nötig ist
```

Der Schwerpunkt dieser Änderung liegt ausschließlich auf:

```text
AviSynth environment + YV12
FFmpeg zscale als letzte Engine
VirtualDub FileNumber
Temp Cleanup
vollständiger Konfigurations-/Template-Dokumentation
```
