const fastify = require('fastify')({ logger: true });

/**
 * listUsers returns all users.
 */
fastify.get('/users', listUsers);

/**
 * createUser creates a new user.
 */
fastify.post('/users', createUser);

/**
 * getUser returns a single user by ID.
 */
fastify.get('/users/:id', getUser);

/**
 * updateUser updates an existing user.
 */
fastify.put('/users/:id', updateUser);

/**
 * deleteUser removes a user by ID.
 */
fastify.delete('/users/:id', deleteUser);

/**
 * patchUser partially updates a user.
 */
fastify.patch('/users/:id', patchUser);

fastify.listen({ port: 3000 }, function (err) {
    if (err) {
        fastify.log.error(err);
        process.exit(1);
    }
});
