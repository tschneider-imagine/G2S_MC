# G2S XSD schemas saved for project use — pass 8

This folder is a project-ready staging area for the old public G2S schema set found in `anthony-folen-igt/TestScriptRunner`.

## Actual XSD files saved here

- `g2sMessage.xsd` — actual file pulled earlier into the archive. It declares namespace `http://www.gamingstandards.com/g2s/schemas/v1.0.3` and version `2007-10-04 v1.0.3-igt-1`.

## Additional XSD files found

See `schema_inventory_found.csv`. I found 57 XSD filenames in the same public repo path, including cabinet, communications, commConfig, meters, eventHandler, printer, player, gameplay, and device/hardware schemas.

## Pull helper

Run `pull_all_found_xsd.ps1` from this folder on a networked Windows machine to download the full discovered XSD set into this project folder.

## Limitation

The execution container used for this pass could not resolve/connect to raw GitHub, so I could not bulk-download all discovered XSDs directly into the archive from inside the sandbox. I still staged the actual available `g2sMessage.xsd`, the complete inventory, and the bulk pull script in the project folder.
