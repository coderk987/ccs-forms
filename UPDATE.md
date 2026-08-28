The repository is for a form creator app prototype hence using minimal code and logic and being basic API routes.
The tech stack being: Go, Gin, Pg pool
The folder structure is main => calling routers => routers => using middlewares and calling controllers => controllers handling logic

Create the following routes and controllers for forms by reading the ROUTES.md and setup.sql file for the database schema and route definitons
I dont want you to write the underlying business logic, just write the route handler, the controller, and the validation in gin by binding to a schema.

GET /published_forms/ => all live forms of current user => No schema needed
PATCH /published_form/:id => edit metadata of form => Schema should have either email OR description OR deadline, it can have one or two empty/non existant fields but atleast one of them should exist
DELETE /published_form/:id => convert published back to a draft => No schema needed

GET /published_forms/:id/responses => get all the responses of a live form => No schema needed
GET /form/:id => form from a responders pov => no schema needed
DELETE /response/:id => takeback the response => no schema needed
GET /responses/:id => get the information of a response => no schema needed

NOTE: DO not introduce any business logic, just setup the routes and schema

Follow the existing project structure and conventions.

Do NOT introduce: ORM libraries, service/repository layers, change folder structures, additional architectural layers, unnecessary abstractions
new dependencies UNLESS EXTREMELY NEEDED in which case you must prompt me.