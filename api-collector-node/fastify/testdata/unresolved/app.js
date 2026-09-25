const fastify = require('fastify')({ logger: true });

/**
 * echo echoes the request payload.
 */
fastify.post('/echo', {
    schema: {
        body: echoRequest,
        response: {
            200: echoReply
        }
    }
}, echoHandler);

fastify.listen({ port: 3000 });
