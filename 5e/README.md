# 5e

Search D&D 5e data in a local checkout of the [5etools](https://github.com/5etools-mirror-3/5etools-src) data.

```bash
git clone https://github.com/5etools-mirror-3/5etools-src ~/github.com/5etools-mirror-3/5etools-src
go install ./5e
5e completion fish > ~/.config/fish/completions/5e.fish
```

The data directory defaults to `~/github.com/5etools-mirror-3/5etools-src/data`; point elsewhere with `-data` or `FIVETOOLS_DATA`.

## Usage

```bash
5e spells -e 24 -c paladin -l 1-3             # paladin spells, levels 1 to 3
5e spells -e 14 -c paladin -sub vengeance     # include Oath of Vengeance spells
5e spells -e 24 -c cleric -t bonus -no-conc   # bonus actions without concentration
5e spells -e 14 -ritual -school div           # divination rituals, any class
5e spells -e 24 fireball -full                # full text
```

| Flag | Meaning |
|------|---------|
| `-e 14\|24` | edition, required |
| `-c` | class |
| `-sub` | subclass short name (`vengeance`, `devotion`, `lore`); adds its spells |
| `-l` | level or range, `0` = cantrips |
| `-t` | casting time, comma list: `action`, `bonus`, `reaction`, `minute`, `hour` |
| `-conc` / `-no-conc` | only with / without concentration |
| `-ritual` | only rituals |
| `-school` | school, a prefix is enough |
| `-src` | only spells from one book |
| `-no-m` / `-no-costly` | no material component / none with a gp cost |
| `-full` | full spell text instead of the table |

Remaining words are a name search (`5e spells -e 24 smite`).

## Completion

`5e completion fish` prints a fish completion script. It completes flags, editions, casting times and schools, and reads the data for classes, subclasses (of the class already typed), books and spell names, limited to the edition given with `-e`.

## Columns

`C` concentration, `R` ritual, `M$` a material component with a gp cost. `Src` is the book the spell is printed in. `Via` is empty for the class's own list; otherwise it names the book that adds the spell to the list (XGE, TCE, …) or the subclass.

## Editions

The class lists come from `data/generated/gendata-spell-source-lookup.json`. 2014 uses the `PHB` classes, 2024 the `XPHB` ones (the class `edition` field decides). A spell printed in both editions shows once: 2024 takes the reprint, 2014 the original, and 2014 never shows spells from the 2024 Player's Handbook.
