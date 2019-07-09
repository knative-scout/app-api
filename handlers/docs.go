/*
Most endpoints to not require authentication.  

Those which do will be marked. Provide authentication as a bearer token in the
`Authorization` header.  

Endpoints which specify a response of `None` will return the 
JSON: `{"ok": true}`.
*/
package handlers
