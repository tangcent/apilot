import { FastifyRequest, FastifyReply } from 'fastify';

interface CreateUserRequest {
    name: string;
    email: string;
    age?: number;
}

interface UserResponse {
    id: number;
    name: string;
    email: string;
}

interface ListUsersResponse {
    users: UserResponse[];
    total: number;
}

/**
 * listUsers returns all users.
 */
fastify.get('/users', async (request: FastifyRequest, reply: FastifyReply) => {
    return { users: [], total: 0 };
});

/**
 * createUser creates a new user.
 */
fastify.post('/users', async (request: FastifyRequest<{ Body: CreateUserRequest }>, reply: FastifyReply) => {
    const { name, email } = request.body;
    return { id: 1, name, email };
});

/**
 * getUser returns a single user by ID.
 */
fastify.get('/users/:id', async (request: FastifyRequest<{ Params: { id: string } }>, reply: FastifyReply) => {
    return { id: request.params.id };
});

/**
 * updateUser updates an existing user.
 */
fastify.put('/users/:id', async (request: FastifyRequest<{ Params: { id: string }; Body: CreateUserRequest }>, reply: FastifyReply) => {
    const { name, email } = request.body;
    return { id: 1, name, email };
});
