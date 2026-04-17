const express = require('express');
const fastify = require('fastify')({ logger: true });
const app = express();

/**
 * expressHello returns a greeting from Express.
 */
app.get('/express/hello', expressHello);

/**
 * expressCreateUser creates a new user via Express.
 */
app.post('/express/users', expressCreateUser);

/**
 * fastifyHello returns a greeting from Fastify.
 */
fastify.route({
    method: 'GET',
    url: '/fastify/hello',
    handler: fastifyHello
});

/**
 * fastifyDeleteUser deletes a user via Fastify.
 */
fastify.route({
    method: 'DELETE',
    url: '/fastify/users/:id',
    handler: fastifyDeleteUser
});

app.listen(3000);
