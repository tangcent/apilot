const fastify = require('fastify')({ logger: true });

/**
 * createUser creates a new user.
 */
fastify.post('/users', {
    schema: {
        body: {
            type: 'object',
            required: ['name'],
            properties: {
                name: {
                    type: 'string',
                    description: 'display name',
                    example: 'John'
                },
                role: {
                    type: 'string',
                    description: 'user role',
                    default: 'viewer',
                    enum: ['admin', 'editor', 'viewer']
                }
            }
        }
    }
}, createUser);

fastify.listen({ port: 0 });
