# magnetico
*Autonomous (self-hosted) BitTorrent DHT search engine suite.*

magnetico is the first autonomous (self-hosted) BitTorrent DHT search engine suite that is *designed
for end-users*. The suite consists of two packages:

- **magneticod:** Autonomous BitTorrent DHT crawler and metadata fetcher.
- **magneticow:** Lightweight web interface for magnetico.

Both programs, combined together, allows anyone with a decent Internet connection to access the vast
amount of torrents waiting to be discovered within the BitTorrent DHT space, *without relying on any
central entity*.

**magnetico** liberates BitTorrent from the yoke of centralised trackers & web-sites and makes it
*truly decentralised*. Finally!

This is a continuation of the original ![magnetico](https://github.com/boramalper/magnetico) and has the following improvements
- Support for postgres in addition to sqlite (magneticod only for now)
- enhanced configuration via enviroment variables
- fixed a bug that prevent single torrents with just 1 file from being added
- magneticow is working again (go variant) and has enhanced statistics

## Features
- Easy installation & minimal requirements:
  - Static binaries, no dependencies required
  - Root access is *not* required to install.
- Near-zero configuration:
  - magneticod works out of the box, and magneticow requires minimal configuration to work with the
    web server you choose.
  - Detailed, step-by-step manual to guide you through the installation.
- No reliance on any centralised entity:
  - **magneticod** crawls the BitTorrent DHT by "going" from one node to another, and fetches the
    metadata using the nodes without using trackers.
- Resilience:
  - Unlike client-server model that web applications use, P2P networks are *chaotic* and
    **magneticod** is designed to handle all the operational errors accordingly.
- High performance implementation:
  - **magneticod** utilizes every bit of your bandwidth to discover as many infohashes & metadata as
    possible.
- Built-in lightweight web interface:
  - **magneticow** features a lightweight web interface to help you access the database without
    getting on your way.

### Screenshots
*Click on the images to view full-screen.*

<!-- Use https://www.tablesgenerator.com/markdown_tables -->
| ![The Homepage](https://camo.githubusercontent.com/488606a87a3e1d7238c0539c6b9cf8429e2c8f16/68747470733a2f2f696d6775722e636f6d2f3634794433714e2e706e67) | ![Searching for torrents](https://camo.githubusercontent.com/0b6def355a17b944de163a11f77c17c1c622280c/68747470733a2f2f696d6775722e636f6d2f34786a733335382e706e67) | ![ss](https://camo.githubusercontent.com/0bd679ad8bbf038b50c082d80a8e0e37516c813e/68747470733a2f2f696d6775722e636f6d2f6c3354685065692e706e67) |
|:-------------------------------------------------------------------------------------------------------------------------------------------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------:|:---------------------------------------------------------------------------------------------------------------------------------------------:|
|                                                                     __The Homepage__                                                                    |                                                                     __Searching for torrents__                                                                    |                                                     __Viewing the metadata of a torrent__                                                     |

## Why?
BitTorrent, being a distributed P2P file sharing protocol, has long suffered because of the
centralised entities that people depended on for searching torrents (websites) and for discovering
other peers (trackers). Introduction of DHT (distributed hash table) eliminated the need for
trackers, allowing peers to discover each other through other peers and to fetch metadata from the
leechers & seeders in the network. **magnetico** is the finishing move that allows users to search
for torrents in the network, hence removing the need for centralised torrent websites.

## Installation Instructions
> **WARNING:**
>
> **magnetico** is still under active construction, and is considered *pre-alpha* software. Please
> use **magnetico** suite with care and follow the installation instructions carefully to install
> it & secure the installation. Feel perfectly free to send bug reports, suggestions, or whatever
> comes to your mind to send to us through GitHub or personal e-mail.


> **WARNING:**
>
> **magnetico** currently does NOT have any filtering system NOR it allows individual torrents to be
> removed from the database, and BitTorrent DHT network is full of the materials that are considered
> illegal in many countries (violence, pornography, copyright infringing content, and even
> child-pornography). If you are afraid of the legal consequences, or simply morally against
> (indirectly) assisting those content to spread around, follow the **magneticow** installation
> instructions carefully to password-protect the web-interface from others.

0. Install ![dep](https://github.com/golang/dep) and ![go-bindata](https://github.com/jteeuwen/go-bindata)
1. Install via ```go get -tags fts5 github.com/izolight/magnetico/...```
2. The **magneticod** binary should be now under $GOPATH/bin (default is ~/go/bin). If not we will build it together with magneticow later
3. For **magneticow** additional steps are required
4. Navigate to $GOPATH/src/github.com/izolight/magnetico
5. Run ```make all``` to install both or ```make magneticow``` for just magneticow (or follow the commands in the Makefile)

## How to use

Both work out of the box without configuration, but you can set options for more control.

### magneticod

You can set the database-url via the -d flag or $DATABASE environment variable.

The Address that we should accept connections (allow this udp port on your firewall if necessary) on can be configured via the -b flag or $BIND_ADDR environment variable (accepts multiple values).

The interval can be configured via the -i flag or $INTERVAL environment variable and is in milliseconds (default should be fine for most use cases).

For increased verbosity you can add -v or for even more -vv.

### magneticow

You can set the database-url and address similar to magneticod.

## License

All the code is licensed under AGPLv3, unless otherwise stated in the source specific source. See
`COPYING` file for the full license text.
