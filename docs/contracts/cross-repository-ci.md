# Cross-repository Go API CI contract

The Docker, Web, and CLI repositories consume this API through dedicated CI
jobs. Each job checks out immutable Go revision `68d22b2`, validates the
generated smoke contracts, and keeps Laravel fallback workflows unchanged.

| Repository | Workflow commit | Required check |
| --- | --- | --- |
| `idelium-docker` | `0ab36f7` | pinned Go formatting, app/handler tests, sanitized fixtures |
| `idelium-web` | `5dc3ccf` | Web smoke-target regeneration and fixture validation |
| `idelium-cli` | `2a1d37c` | CLI smoke-target regeneration and fixture validation |

The jobs use no runtime credentials and introduce no mutable image reference or
production route switch. Docker remains the only runtime for Go checks when the
host toolchain is unavailable.
