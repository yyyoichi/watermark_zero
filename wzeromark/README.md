# How to Generate Marks

This document describes the procedure for generating a byte sequence (**mark**, []byte) from a user-input string (**source**) using the Watermark Zero API (hereafter, `W-ZeroAPI`). The algorithm is publicly available and widely used.

## Overview

W-ZeroAPI generates a **mark** from a source via an **irreversible transformation**. The source is not embedded directly into the image. Instead, the source is stored in an external resource for each organization, identified by a timestamp and nonce. The mark has a fixed length of **664 bits (83 bytes)**, structured as follows:

| Name         | Length | Purpose                        | Details |
| :--          | --:    | :--                            | :--     |
| Version      | 1B     | Mark structure management      | Little-endian |
| Org Code     | 2.5B   | Organization identification    | 20-bit, unique value per organization, usually 5-digit hex (~1.04 million) |
| Nonce        | 1.5B   | Source identification          | 12-bit, random value (4,096 variations) |
| Timestamp    | 6B     | Mark generation date/time      | Unix milliseconds |
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
Stored as 1 byte (8 bits) in little-endian format, allowing 256 possible versions.

### Organization Code

A 20-bit (2.5-byte) value, represented as 5-digit hexadecimal (e.g., `0x0a1b2`).
Supports approximately 1.04 million organizations (1,048,576 variations).
Randomly assigned per organization.

### Nonce

A 12-bit (1.5-byte) random value, supporting 4,096 variations.
The combination of timestamp and nonce is used to query the source externally.

### Timestamp

Unix milliseconds represented as a 6-byte integer (approximately 8,900 years representable).

W-ZeroAPI processes more than 10 images per second, pessimistically estimating about 0.1 images per millisecond per organization (100/sec).
Considering traffic spikes and parallel processing environments, collisions in milliseconds per mark cannot be completely avoided.

Such collisions mean the source cannot be uniquely identified in external resources, so a nonce is used to ensure uniqueness.

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
