const fastify = require('fastify')({ logger: true });

/**
 * listUsers returns all users.
 */
fastify.get('/users', {
    schema: {
        querystring: {
            type: 'object',
            properties: {
                limit: { type: 'integer' },
                offset: { type: 'integer' }
            }
        },
        response: {
            200: {
                type: 'object',
                properties: {
                    users: { type: 'array' },
                    total: { type: 'integer' }
                }
            }
        }
    }
}, listUsers);

/**
 * createUser creates a new user.
 */
fastify.post('/users', {
    schema: {
        body: {
            type: 'object',
            required: ['name', 'email'],
            properties: {
                name: { type: 'string' },
                email: { type: 'string' },
                age: { type: 'integer' }
            }
        },
        response: {
            200: {
                type: 'object',
                properties: {
                    id: { type: 'integer' },
                    name: { type: 'string' },
                    email: { type: 'string' }
                }
            }
        }
    }
}, createUser);

/**
 * getUser returns a single user by ID.
 */
fastify.get('/users/:id', {
    schema: {
        params: {
            type: 'object',
            properties: {
                id: { type: 'string' }
            }
        },
        response: {
            200: {
                type: 'object',
                properties: {
                    id: { type: 'integer' },
                    name: { type: 'string' }
                }
            }
        }
    }
}, getUser);

fastify.listen({ port: 3000 });
