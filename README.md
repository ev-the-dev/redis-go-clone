# REDIS Go Clone

## 0.0 Running & Testing

I've been testing the server functionality by using Redis' client to connect to it. To run this server and test with one or more clients do the following:

Ensure the Redis CLI is installed:
```sh
mise use --global redis
```

Clone this repository and `cd` to the root directory:
```sh
git clone git@github.com:ev-the-dev/redis-go-clone.git && cd redis-go-clone
```

Run the server from the root directory:
```sh
go run .
```
OR if tesing with an existing local rdb file:
```sh
go run . --dir ./ --dbfilename dump.rdb
```

Open a new shell to connect to the server from the previous step (this step can be repeated to simulate multiple connected clients):
```sh
redis-cli -p 6379
```

## 1.0 Understanding the RDB file

[File Reference](https://rdb.fnordig.de/file_format.html)

Each RDB file is broken up into a few sections:
1. Header
2. Metadata / Auxiliary Fields
3. Database Selection
4. Footer


### 1.1 Section Parsing & Length Encoding

To fully understand how section parsing and key:value pairs in RDB files work, this section is necessary. 

Some caveats:
- Timestamps and Checksums seem to be in little-endian, so when evaluating them be cognizant of reversing the bytes.

#### 1.1.1 Sections

There are some reserved hex codes for specific sections and sub-sections:
- `0xFA`: Beginning of a new Auxiliary field. Followed by length-encoded key:value pair.
- `0xFE`: Beginning of a new Database. Followed by length-encoded value describing DB number.
- `0xFB`: Proceeds `0xFE` and describes hash table sizes for main keyspace and expires.
    - NOTE: It's my understanding that for newer versions of REDIS, the next two bytes will always be `02 01`. This is because REDIS now uses *lazy resizing* and does not need precise initial sizing.
- `0xFD` & `0xFC`: Mutually exclusive (I think) codes representing the proceeding DB field's expire time (FD for seconds & FC for milliseconds).
    - `0xFD`: Following ***4 bytes*** represent uint Unix timestamp in seconds.
    - `0xFC`: Following ***8 bytes*** represent unsigned long Unix timestamp in milliseconds.
- `0xFF`: Signifies the end of the RDB file.

#### 1.1.2 Length Encoding

When parsing a length-encoded descriptor, you need to think about the underlying ***bits*** of the hexadecimal value. The first two bits in the byte (significant bits) determine how to parse the rest of the length-encoded descriptor *as well* as the field it describes and how many bytes comprise that field. Here are the 4 types of significant bit pairs:
- `00`: Next 6 bits represent length.
- `01`: Next 6 bits *plus* the next byte represent length (14 bits total).
- `10`: Discard remaining 6 bits. Next 4 bytes represent length.
- `11`: Special format. Next 6 bits describe format. Can be used to store numbers or strings using [*String Encoding*](#1a2-string-encoding).

> [!WARNING]
> Length Encoding has a significant nuance to it. The first 3 cases (00, 01, and 10) simply return how many bytes to read for the default value type -- this type being determined by external context, such as strings in metadata, or the [*Value Type*](#1a1-value-type) flag in the database sections.
> The 4th case (11) provides some *extra, internal* context. It, paired with that same external context, explains whether the proceding value is of an integer or LZF type (See: [*String Encoding*](#1a2-string-encoding)).

Example of a length-encoded descriptor and value:
`00 05 68 65 6c 6c 6f`
- `00`: Value type: `0` in binary represents a string (TODO: link incoming).
- `05`: Convert from HEX->Binary: `05`->`00000101`.
    - First 2 bits are `00`, thus remaining 6 bits determine the length in bytes.
    - Remaining 6 bits are `000101` or `5`, so read the next 5 bytes as ASCII (due to value type).
- `68 65 6c 6c 6f`: Convert from HEX->String: `68 65 6c 6c 6f`->`H E L L O`.

### 1.2 Header

The header is extremely simple, it contains two key pieces of information:
1. A "magic" string that spells out `REDIS`.
    1. This is represented as HEX->ASCII: `52 45 44 49 53`->`R E D I S`.
2. An ASCII RDB version number, i.e. `0012`.
    1. This is represanted as HEX->ASCII: `30 30 31 32`->`0 0 1 2`.
    2. NOTE: This is the ***RDB*** version, NOT the *REDIS* version.


### 1.3 Metadata / Aux Fields

This section *should* have a fixed amount of entries. I say *should* because there could be unknown fields, but these should be ignored by the parser.

> [!NOTE]
> Each key:value pair is preceded by the `0xFA` op code and are of the ***string*** value type.
> Because they'll always be of a string value type, there's no need for the extra preceeding byte to tell us what the value type is, like we see in the database section.

These are the supported fields:
- `redis-ver`: REDIS version that wrote the RDB file.
- `redis-bits`: Bit architecture of OS that wrote the RDB (32 or 64).
- `ctime`: Creation time of the RDB file.
- `used-mem`: Used memory of instance that wrote the RDB file.

Here's an example of how one of these fields look like in the RDB file with the op code prefix:
`fa 09 72 65 64 69 73 2d 76 65 72 05 37 2e 34 2e 32`
- `fa`: Indicates new aux field.
- `09`: [length encoded](#11-section-parsing--length-encoding) descriptor for the key of the key:value pair.
    - Converting HEX->Binary: `09`->`00001001`.
        - First 2 bits are `00`, thus remaining 6 bits describe length, in bytes, of proceeding key.
    - `001001` = `9`, thus next 9 bytes are the key.
- `72 65 64 69 73 2d 76 65 72`: The 9 bytes converted to ASCII read as -> `redis-ver`.
- `05`: [length encoded](#11-section-parsing--length-encoding) descriptor for the value of the key:value pair.
    - Converting HEX->Binary: `05`->`00000101`.
    - `000101` = `5`, thus 5 bytes are the value.
- `37 2e 34 2e 32`: the 5 bytes converted to ASCII->`7.4.2`.

> [!TIP]
> Putting everything in this example together we see that there is an aux field named `redis-ver` with a value of `7.4.2`.


### 1.4 Database Selection

There can be ***n*** number of DB selectors. Each section starts with `0xFE` op code followed by a byte signifying the DB number -- i.e. `fe 00` = DB number 00.

Each DB section will contain series of records with a specific order of data:
1. `0xFC` or `0xFD`: [Expire times](#111-sections). Represented in little-endian I believe (reverse order of bytes to read the number value properly). Optional?
2. [*Value Type*](#1a1-value-type): 1 byte flag indicating type (string, list, hash).
3. Key: [string-encoded](#1a2-string-encoding).
4. Value: Parsed according to previously read Value Type (see #2 above).

Here's an example of a DB record:
`fc 7d ab e7 4f 96 01 00 00 00 05 68 65 6c 6c 6f 05 77 6f 72 6c 64`

Let's break that up a bit so it's easier to read/parse:
`fc 7d ab e7 4f 96 01 00 00` `00` `05 68 65 6c 6c 6f` `05 77 6f 72 6c 64`

1. Expire time: `fc 7d ab e7 4f 96 01 00 00`.
    - The `fc` indicates an expire time in milliseconds, hence the following 8 bytes allocated for that timestamp.
    - Since these timestamps are in a little-endian format, to convert them to the timestamp they need to be reversed, ergo when computing it would look like this: `00 00 01 96 4f e7 ab 7d`.
2. Value Type: `00`.
    - String value type.
    - Might indicate [*String Encoded*](#1a2-string-encoding) type instead -- in which case the value ultimately is determined by the length encoded significant bits (all cases would be a string type, unless significant bits are `11`).
3. Key: `05 68 65 6c 6c 6f`.
    - `05` HEX->Binary: `05`->`00000101`.
        - First 2 bytes are `00`, thus only the next 6 bits describe size.
        - Remaining bits equal `000101`, or `5`, thus the key comprises the next 5 bytes.
    - Converting HEX->ASCII: `68 65 6c 6c 6f`->`h e l l o`
4. Value: `05 77 6f 72 6c 64`.
    - `05` HEX->Binary: `05`->`00000101`.
        - First 2 bytes are `00`, thus only the next 6 bits describe size.
        - Remaining bits equal `000101`, or `5`, thus the key comprises the next 5 bytes.
    - Converting HEX->ASCII: `77 6f 72 6c 64`->`w o r l d`

> [!TIP]
> Putting everything in this example together we see that there is a DB field named with an expiry time in milliseconds equalling `1745097304957`, the value type is a string represented as `00`, the key is 5 bytes long and spells `hello`, the value is 5 bytes long and is a string stating `world`.


### 1.5 Footer

The footer is pretty basic, it just contains two things:
1. `0xFF`: EOF indicator op code.
2. Checksum: little-endian(?) 8 bytes of CRC64 checksum of the entire file.

### 1.A Appendix

#### 1.A.1 Value Type

Value types are generally 1-byte flags. I say generally, because I still use the term "value type" for implicitly defined types, like those for string encoded keys in the metadata section.

The value of the 1-byte flag correlates to a specific type:
- `0`: [*String Encoding*](#1a2-string-encoding)
- `1`: [*List Encoding*](#1a3-list-encoding)
- `2`: [*Set Encoding*](#1a4-set-encoding)
- `3`: [*Sorted Set Encoding*](#1a5-sorted-set-encoding)
- `4`: [*Hash Encoding*](#1a6-hash-encoding)
- `9`: [*Zipmap Encoding*](#1a7-zipmap-encoding)
- `10`: [*Ziplist Encoding*](#1a8-ziplist-encoding)
- `11`: [*Intset Encoding*](#1a9-intset-encoding)
- `12`: [*Ziplist Sorted Set Encoding*](#1a10-ziplist-sorted-set-encoding)
- `13`: [*Ziplist Hashmap Encoding*](#1a11-ziplist-hashmap-encoding)
- `14`: [*Quicklist List Encoding*](#1a12-quicklist-list-encoding)

#### 1.A.2 String Encoding

There are three types of Strings in a Redis RDB file:
- Length prefixed strings
- 8, 16, or 32 bit integer
- LZF compressed string

*Length Prefixed String:*
- Length of the string, in bytes, is encoded using [*Length Encoding*](#112-length-encoding). Then the following raw bytes of the string are stored.
- This is the case when the [length-encoded](#112-length-encoding) significant bits are either `00`, `01`, or `10` *AND* the [*Value Type*](#1a1-value-type) is `00` (string encoded) -- whether implied, as in the metadata fields; or explicit, as in the database sections.

*Integers as Strings:*
- After a [length-encoded](#112-length-encoding) value produced `11` as the significant bits, the remaining 6 bits are read to determine whether the integer is 8, 16, or 32 bits long.
    - `0`: indicates an 8 bit integer follows. Read the next byte for the int value.
    - `1`: indicates a 16 bit integer follows. Read the next 2 bytes for the int value.
    - `2`: indicates a 32 bit integer follows. Read the next 4 bytes for the int value.

*Compressed Strings:*
- Like "Integers as Strings" above, after the length-encoded value produces `11`, the remaining 6 bits are read:
    - `3`: indicates an LZF string follows.
- The compressed string is read as follows:
    - Compressed length `clen` is read using [length-encoding](#112-length-encoding).
    - Uncompressed length is read using [length-encoding](#112-length-encoding).
    - Next `clen` bytes are read.
    - Finally, the read `clen` bytes are decompressed using the `LZF` algorithm.


