# How to Generate Marks

This document describes the procedure for generating a byte sequence (**mark**, []byte) from a user-input string (**source**) using the Watermark Zero API (hereafter, `W-ZeroAPI`). The algorithm is publicly available and widely used.

## Overview

W-ZeroAPI generates a **mark** from a source via an **irreversible transformation**. The source is not embedded directly into the image. Instead, the source is stored in an external resource for each organization, identified by a timestamp and nonce. The mark has a fixed length of **664 bits (83 bytes)**, structured as follows:

| Name         | Length | Purpose                        | Details |
| :--          | --:    | :--                            | :--     |
| Version      | 1B     | Mark structure management      |         |
| Timestamp    | 6B     | Mark generation date/time      | Unix milliseconds |
| Nonce        | 2B     | Source identification          | Random value |
| Org Code     | 2B     | Organization identification    | Unique value per organization, usually 4-digit hex |
| Hash         | 8B     | Source storage                 | First 8 bytes of HMAC-SHA256 hash deterministically derived from date/time and source |
| Signature    | 64B    | Tamper prevention              | Ed25519 signature of previous fields |

### Secrets

Secrets required for mark creation:

| Name        | Length | Unit   | Scope      | Details |
| :--         | --:    | :--    | :--        | :--     |
| Master Key  | 32B    | Org    | Hash, Signature | Used as input key for HKDF to generate HMAC pepper and Ed25519 seed |
| System Salt | 32B    | System | HKDF       | Salt for HKDF key generation |

## Structure Details

### Version

The current version is `0x01`.

### Timestamp

Unix milliseconds represented as a 6-byte integer.

W-ZeroAPI processes more than 10 images per second, pessimistically estimating about 0.1 images per millisecond per organization. Considering traffic spikes and parallel processing, collisions in marks per millisecond cannot be completely avoided.

Such collisions mean the source cannot be uniquely identified in external resources, so a nonce is used to ensure uniqueness.

### Nonce

A 2-byte random value. The combination of timestamp and nonce is used to query the source externally.

### Organization Code

A 2-byte hexadecimal value, e.g., `0x0a1b`. Randomly assigned per organization.

### Hash

The hash is the first 8 bytes of an HMAC-SHA256 value derived from the source. The HMAC pepper (secret key) is generated hourly for each organization using HKDF-SHA256 with the master key as input.

#### HMAC Pepper Generation via HKDF

Key structure:

| Parameter | Value | Details |
| --        | --    | --      |
| IKM       | Master Key | Organization-specific input key |
| Salt      | System Salt | Secret information |
| Info      | Generation time + 'W-ZeroAPI-HMAC-Key-V1' | Concatenation of mark generation time (YYYYMMddHH) and HMAC domain |

### Signature

The signature is a 64-byte Ed25519 signature of the preceding 19 bytes. The seed (private key) is generated hourly for each organization using HKDF-SHA256 with the master key as input.

#### Ed25519 Seed Generation via HKDF

Key structure:

| Parameter | Value | Details |
| --        | --    | --      |
| IKM       | Master Key | Organization-specific input key |
| Salt      | System Salt | Secret information |
| Info      | Generation time + 'W-ZeroAPI-Ed25519-Seed-V1' | Concatenation of mark generation time (YYYYMMddHH) and Ed25519 domain |
