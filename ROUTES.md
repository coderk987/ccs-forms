AUTH
POST /auth/signup
POST /auth/login 🚧 [protoype for now then OAUTH]

DRAFTS
GET /draft_forms/ => forms created by current user ✅
POST /draft_forms/ => create a form draft by current user ✅
GET /draft_forms/:id => get a draft forms information ✅🚧
PUT /draft_forms/:id => edit questions/sections added to draft !! ✅🚧
PATCH /draft_forms/:id => edit metadata of form(info, deadline, viewers) ✅
DELETE /draft_forms/:id => delete the current form ✅
POST /draft_forms/:id/publish => publish the forms ! ✅

Note for frontend => Get :id will give you a json of a forms information, it is the frontends job to maintain it on the client side and send the complete changed json again start to finish in the PUT request when saving/when the client is closing the window
Client shouldnt change the schema and NEVER touch the id's only the information/deletions/insertions

PUBLISH
GET /published_forms/ => all live forms of current user ✅
PATCH /published_form/:id => edit metadata of form ✅
DELETE /published_form/:id => convert published back to a draft ✅

GET /published_forms/:id/responses => get all the responses of a live form ✅
QUERY /published_forms/:id/analytics => [work in progress]

RESPONDER
GET /form/:id => form from a responders pov ✅
POST /form/:id/response => add a response to the given form id ✅

RESPONSES
DELETE /response/:id => takeback the response ✅
GET /responses/:id => get the information of a response ✅

ADMIN
PATCH /users/admin => makes a user an admin
POST /forms/:id/view-permissions => grants view permission to a user ✅