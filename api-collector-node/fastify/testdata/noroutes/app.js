const fastify = require('fastify')({ logger: true });

function someHelper() {
    return 'hello';
}

class UserService {
    get_users() {
        return [];
    }
}

fastify.listen({ port: 3000 });
