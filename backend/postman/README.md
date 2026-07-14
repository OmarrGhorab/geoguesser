# GeoGuess Postman

## Import
1. Postman → **Import**
2. Select:
   - `GeoGuess-API.postman_collection.json`
   - `GeoGuess-Local.postman_environment.json`
3. Choose environment **GeoGuess Local**

## Quick start
1. Run **01 Auth → Get current session (bootstrap cookies)**
2. Set `csrfToken` if the test script did not (copy `csrf_token` cookie)
3. Run **Register** or **Login**
4. Explore folders for Health, Maps, Games, Rooms, Friends, etc.

## Auth notes
- Cookie sessions (`access_token`, `refresh_token`, optional `guest_session`)
- Mutations need header `X-CSRF-Token: {{csrfToken}}` matching the cookie
- Keep Postman's cookie jar enabled

## Examples
- Example responses are embedded on each request (Examples dropdown in Postman).

## Sources
- `backend/openapi/openapi.yaml`
- `backend/internal/app/routes.go`
