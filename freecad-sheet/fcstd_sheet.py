#!/usr/bin/env python3
"""
CLI for reading and writing FreeCAD spreadsheet variables in .FCStd files.

Expects the spreadsheet to follow the convention:
  Column A = human-readable label (string)
  Column B = value (numeric, with alias set)
  Column C = unit (string)

Usage:
  fcstd_sheet.py read <file>
  fcstd_sheet.py set <file> <alias> <value>
  fcstd_sheet.py add <file> <label> <value> <unit> <alias>

Environment:
  FREECAD_LIB  Path to FreeCAD lib directory
               Default: ~/squashfs-root/usr/lib/
"""

import argparse
import os
import sys

FREECAD_LIB = os.environ.get(
    'FREECAD_LIB',
    os.path.expanduser('~/squashfs-root/usr/lib/')
)
sys.path.insert(0, FREECAD_LIB)
os.environ.setdefault('QT_QPA_PLATFORM', 'offscreen')

import FreeCAD  # noqa: E402


def open_doc(path):
    return FreeCAD.openDocument(os.path.abspath(path))


def _row_of(address):
    return int(''.join(c for c in address if c.isdigit()))


def _col_of(address):
    return ''.join(c for c in address if c.isalpha())


def read_rows(sheet):
    cells = sheet.getNonEmptyCells()
    rows = sorted(set(_row_of(a) for a in cells))
    result = []
    for row in rows:
        label = ''
        value = ''
        unit = ''
        alias = ''
        try:
            label = sheet.get(f'A{row}')
        except Exception:
            pass
        try:
            value = sheet.get(f'B{row}')
            alias = sheet.getAlias(f'B{row}') or ''
        except Exception:
            pass
        try:
            unit = sheet.get(f'C{row}')
        except Exception:
            pass
        result.append((row, label, alias, value, unit))
    return result


def next_empty_row(sheet):
    cells = sheet.getNonEmptyCells()
    if not cells:
        return 1
    return max(_row_of(a) for a in cells) + 1


def cmd_read(args):
    doc = open_doc(args.file)
    sheet = doc.Spreadsheet
    rows = read_rows(sheet)
    if not rows:
        print('No cells found.')
    else:
        print(f'{"alias":<24} {"value":<16} {"unit":<6} label')
        print('-' * 60)
        for _, label, alias, value, unit in rows:
            print(f'{alias or "(no alias)":<24} {str(value):<16} {str(unit):<6} {label}')
    FreeCAD.closeDocument(doc.Name)


def cmd_set(args):
    doc = open_doc(args.file)
    sheet = doc.Spreadsheet
    addr = sheet.getCellFromAlias(args.alias)
    if not addr:
        print(f'Error: alias "{args.alias}" not found', file=sys.stderr)
        FreeCAD.closeDocument(doc.Name)
        sys.exit(1)
    sheet.set(addr, str(args.value))
    doc.recompute()
    doc.save()
    FreeCAD.closeDocument(doc.Name)
    print(f'Set {args.alias} ({addr}) = {args.value}')


def cmd_alias(args):
    doc = open_doc(args.file)
    sheet = doc.Spreadsheet
    try:
        sheet.get(args.address)
    except Exception:
        print(f'Error: cell "{args.address}" is empty or does not exist', file=sys.stderr)
        FreeCAD.closeDocument(doc.Name)
        sys.exit(1)
    sheet.setAlias(args.address, args.alias)
    doc.recompute()
    doc.save()
    FreeCAD.closeDocument(doc.Name)
    print(f'Set alias {args.address} -> {args.alias}')


def cmd_add(args):
    doc = open_doc(args.file)
    sheet = doc.Spreadsheet
    row = next_empty_row(sheet)
    sheet.set(f'A{row}', f"'{args.label}")
    sheet.set(f'B{row}', str(args.value))
    sheet.set(f'C{row}', f"'{args.unit}")
    sheet.setAlias(f'B{row}', args.alias)
    doc.recompute()
    doc.save()
    FreeCAD.closeDocument(doc.Name)
    print(f'Added row {row}: {args.alias} = {args.value} {args.unit} ({args.label})')


def main():
    parser = argparse.ArgumentParser(
        description='Read and write FreeCAD spreadsheet variables in .FCStd files.'
    )
    sub = parser.add_subparsers(dest='command', required=True)

    p = sub.add_parser('read', help='List all variables in the spreadsheet')
    p.add_argument('file', help='Path to .FCStd file')
    p.set_defaults(func=cmd_read)

    p = sub.add_parser('alias', help='Set an alias on an existing cell by address')
    p.add_argument('file', help='Path to .FCStd file')
    p.add_argument('address', help='Cell address (e.g. B1)')
    p.add_argument('alias', help='Alias to assign')
    p.set_defaults(func=cmd_alias)

    p = sub.add_parser('set', help='Update a variable by alias')
    p.add_argument('file', help='Path to .FCStd file')
    p.add_argument('alias', help='Cell alias to update')
    p.add_argument('value', help='New value')
    p.set_defaults(func=cmd_set)

    p = sub.add_parser('add', help='Add a new variable row')
    p.add_argument('file', help='Path to .FCStd file')
    p.add_argument('label', help='Human-readable label (column A)')
    p.add_argument('value', help='Numeric value (column B)')
    p.add_argument('unit', help='Unit string (column C)')
    p.add_argument('alias', help='Alias for the value cell')
    p.set_defaults(func=cmd_add)

    args = parser.parse_args()
    args.func(args)


if __name__ == '__main__':
    main()
