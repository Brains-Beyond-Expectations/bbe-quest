# `bbe version`

**Alias:** `bbe v`

Prints the currently installed BBE-Quest CLI version.

---

## Usage

```bash
bbe version
# or
bbe v
```

---

## Output

```
bbe version 0.6.1
```

The version is injected at build time via Go `ldflags` (`-X constants.Version=<version>`). Development builds will show an empty string or a dev placeholder.
