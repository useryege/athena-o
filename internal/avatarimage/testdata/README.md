# WebP test fixtures

These files are unchanged copies from `golang.org/x/image` tag `v0.45.0`
(`3ebddc7c54bd879f8d84d11db82892726f5192fd`), under its BSD license in
`LICENSE.ximage`. Source archive:
`https://proxy.golang.org/golang.org/x/image/@v/v0.45.0.zip`.

| File | SHA-256 | Purpose |
| --- | --- | --- |
| `blue-purple-pink.lossy.webp` | `af8e87f21fa9fcb8e74c12d31d61a56b1d8e06819038efd10119b0f0726bfab4` | Ordinary lossy WebP upload |
| `yellow_rose.lossy-with-alpha.webp` | `b6ddb7ed2f0397f132a975e9c2cdbc3e01abb39096fb1a584cf8c30a0d612378` | Extended WebP upload and mutated animation/canvas tests |
| `gopher-doc.skip-hgroup.lossless.webp` | `576c62996ae5570f5d1a287fe50ae8525ffdd8f5075d37589f10c0acefd2a72c` | Valid VP8L with unreferenced Huffman groups |
| `large-huffman-index.lossless.webp.bz2` | `f9d9435c17626f50c8c9de51b8e58e55919c47d81cbe3e0503535199fd4bb64e` | Malformed VP8L that made the old decoder allocate about 170 MiB |

The decompressed malicious WebP is 163,879 bytes, with SHA-256
`3b95a3b75177f68c66bc4b22791e595216be02b6e1a6bfcd2558fe324033856c`.
Its test is deliberately opt-in and must run with a hard memory and time limit.
