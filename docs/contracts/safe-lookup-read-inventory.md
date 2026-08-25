# Safe Lookup Read Inventory

Wave 3 reviewed the read-only routes that looked small enough to migrate before
stateful CLI, execution, and administration domains. The result is deliberately
conservative: the standalone safe lookup surface is already covered by the
platform catalog read endpoints moved to Go ownership.

## Migrated safe lookup reads

These `GET|HEAD` routes are Go-owned and covered by Laravel-Go golden fixture
comparison:

- `/api/admin/platforms/types`
- `/api/admin/platforms/status`
- `/api/admin/platforms/locations`
- `/api/admin/platforms/brands`
- `/api/admin/platforms/models/{idBrand}`
- `/api/admin/platforms/os/{idType}`
- `/api/admin/platforms/osversion/{idOs}`
- `/api/admin/platforms/browsers/{idOs}`
- `/api/admin/platforms/browserversions/{idBrowser}`
- `/api/admin/platforms/manageplatforms/{type}`

The legacy `browserversions` spelling is intentionally preserved because it is
part of the public Laravel route contract consumed by Idelium Web.

## Wave 6 launcher configuration read

`GET|HEAD /api/admin/platforms/manageplatforms/{type}` is now Go-owned as part
of the Wave 6 launcher configuration read slice. The paired `POST`, `PUT`, and
`DELETE` platform management mutations remain Laravel-owned until their
dedicated mutation tickets move validation and rollback behavior together.

## Other read groups

- CLI configuration reads stay in Wave 4 because they use the API-key trust path
  and must preserve CLI compatibility contracts.
- Execution result reads stay in Wave 7 because they depend on artifact storage,
  redaction, and tenant-scoped retrieval policies.
- Identity, customer, project, account, profile, and menu reads stay in Wave 9
  because they are browser-session administration routes coupled to Laravel
  identity and tenant context.

No traffic is moved by this inventory update. Rollback is a Git revert; the
route rollout overrides remain unchanged.
