# go-download-osm

This is an experimental Go port of the Python tool [download-osm](https://github.com/openmaptiles/openmaptiles-tools/blob/master/bin/download-osm), a "fast multi-mirror parallel planet and extracts downloader". The original tool is part of the [openmaptiles/openmaptiles-tools](https://github.com/openmaptiles/openmaptiles-tools) package. This is a partial implementation of that tool, mostly focused on an OSM planet file download for now.

This tool will generate a list of URL's from a set of mirrors hosting the recent OSM planet file. The URL's can then be used by [aria2](https://aria2.github.io/) to download the file.

I came across ```download-osm``` while watching [a recent talk](https://youtu.be/uDT7C7Mg83s?t=679) from PostGIS Day 2021 given by Yuri Astrakhan ([nyurik](https://github.com/nyurik)).
