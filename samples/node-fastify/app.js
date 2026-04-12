const fastify = require('fastify')({ logger: true });

fastify.get('/users', async (request, reply) => {
    const { name, role = 'user' } = request.query;
    return { users: [] };
});

fastify.post('/users', async (request, reply) => {
    const { name, email } = request.body;
    reply.code(201);
    return { name, email };
});

fastify.get('/users/:id', async (request, reply) => {
    return { id: request.params.id };
});

fastify.put('/users/:id', async (request, reply) => {
    const { name, email } = request.body;
    return { name, email };
});

fastify.delete('/users/:id', async (request, reply) => {
    reply.code(204);
    return '';
});

fastify.patch('/users/:id', async (request, reply) => {
    const { name = 'unknown' } = request.query;
    return { id: request.params.id };
});

const start = async () => {
    try {
        await fastify.listen({ port: 3000 });
    } catch (err) {
        fastify.log.error(err);
        process.exit(1);
    }
};

start();
