# Browser auth bridge retirement

Wave 9 removes the temporary Laravel browser-auth introspection bridge from the
Go runtime. Browser authentication must be handled by Go-native identity,
customer, session, and authorization code after the compatibility gates pass.

## Retired configuration

The API fails closed during startup when any retired bridge setting is present:

- `IDELIUM_AUTH_BRIDGE_URL`
- `IDELIUM_AUTH_BRIDGE_TOKEN`
- `IDELIUM_AUTH_BRIDGE_SECRET`
- `IDELIUM_BROWSER_AUTH_BRIDGE_URL`
- `IDELIUM_LARAVEL_AUTH_INTROSPECTION_URL`

The diagnostic reports only the variable name and retirement reason. It never
prints URLs, tokens, shared secrets, session identifiers, cookies, authorization
headers, or payload data.

## Deployment impact

Before enabling this version in an environment:

1. remove any bridge endpoint or bridge credential variables from the Go API
   deployment;
2. rotate and revoke the retired internal bridge credential in the Laravel
   environment;
3. confirm that browser-auth traffic is routed only to the Go-native auth
   implementation or deliberately kept on Laravel through the gateway route
   owner matrix;
4. run the browser-auth compatibility and tenant-isolation smoke checks.

## Rollback

Rollback is configuration-first:

1. revert the deployment to the last dual-runtime release if the Go-native auth
   path fails;
2. restore the gateway route owner for browser-auth routes to Laravel;
3. provision a freshly rotated bridge credential only for the older fallback
   release;
4. keep bridge traffic on the private Docker network and remove it again before
   retrying cutover.

No schema change is introduced by this slice and no HTTP contract changes are
exposed directly.
