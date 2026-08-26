The repository is for a form creator app prototype hence using minimal code and logic and being basic API routes.
The tech stack being: Go, Gin, Pg pool
The folder structure is main => calling routers => routers => using middlewares and calling controllers => controllers handling logic

Create the following routes and controllers for forms by reading the ROUTES.md and setup.sql file for the database schema and route definitons
GET /draft_forms/ => forms created by current user
POST /draft_forms/ => create a form draft by current user
GET /draft_forms/:id => get a draft forms information
PATCH /draft_forms/:id => edit metadata of form(info, deadline, viewers)
DELETE /draft_forms/:id => delete the current form

Follow the existing project structure and conventions.

Do NOT introduce: ORM libraries, service/repository layers, change folder structures, additional architectural layers, unnecessary abstractions
new dependencies UNLESS EXTREMELY NEEDED in which case you must prompt me.