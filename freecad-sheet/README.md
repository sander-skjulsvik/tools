# freecad-sheet

CLI for reading and writing variables in a FreeCAD spreadsheet (`.FCStd` file) without opening the GUI.

Expects the spreadsheet to follow this column convention:

| Column | Content |
|--------|---------|
| A | Human-readable label |
| B | Numeric value — alias set here |
| C | Unit string |

## Requirements

FreeCAD installed via AppImage, extracted to `~/squashfs-root/`:

```bash
wget https://github.com/FreeCAD/FreeCAD/releases/download/1.1.1/FreeCAD_1.1.1-Linux-x86_64-py311.AppImage
chmod +x FreeCAD_1.1.1-Linux-x86_64-py311.AppImage
./FreeCAD_1.1.1-Linux-x86_64-py311.AppImage --appimage-extract
```

Override the lib path with `FREECAD_LIB` if installed elsewhere.

## Usage

```bash
# List all variables
QT_QPA_PLATFORM=offscreen ~/squashfs-root/usr/bin/python fcstd_sheet.py read file.FCStd

# Update a variable by alias
QT_QPA_PLATFORM=offscreen ~/squashfs-root/usr/bin/python fcstd_sheet.py set file.FCStd room_x 4100

# Add a new variable
QT_QPA_PLATFORM=offscreen ~/squashfs-root/usr/bin/python fcstd_sheet.py add file.FCStd "Bench depth" 650 mm bench_depth
```

## Example output

```
alias                    value            unit   label
------------------------------------------------------------
room_x                   4100             mm     Room x
room_y                   4000             mm     Room y
bench_depth              650              mm     Bench depth
```
