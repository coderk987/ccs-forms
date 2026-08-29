The repository is for a form creator app prototype hence using minimal code and logic and being basic API routes.
The tech stack being: Go, Gin, Pg pool
The folder structure is main => calling routers => routers => using middlewares and calling controllers => controllers handling logic

Create the following routes and controllers for forms by reading the ROUTES.md and setup.sql file for the database schema and route definitons

/admin routes that
Create other users to admins by their gmail
Add view permission to a gmail(must be an admin) => only valid to form author

NOTE: DO not introduce any business logic, just setup the routes and schema

Follow the existing project structure and conventions.

Do NOT introduce: ORM libraries, service/repository layers, change folder structures, additional architectural layers, unnecessary abstractions
new dependencies UNLESS EXTREMELY NEEDED in which case you must prompt me.
