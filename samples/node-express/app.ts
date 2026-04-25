import { Request, Response } from 'express';
const app = require('express')();

app.use(require('express').json());

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

interface UpdateUserRequest {
    name?: string;
    email?: string;
}

/**
 * listUsers returns all users.
 */
app.get('/users', (req: Request, res: Response<ListUsersResponse>) => {
    const { name, role = 'user' } = req.query;
    res.json({ users: [], total: 0 });
});

/**
 * createUser creates a new user.
 */
app.post('/users', (req: Request<{}, {}, CreateUserRequest>, res: Response<UserResponse>) => {
    const { name, email } = req.body;
    res.status(201).json({ id: 1, name, email });
});

/**
 * getUser returns a single user by ID.
 */
app.get('/users/:id', (req: Request<{ id: string }>, res: Response<UserResponse>) => {
    res.json({ id: 1, name: 'test', email: 'test@example.com' });
});

/**
 * updateUser updates an existing user.
 */
app.put('/users/:id', (req: Request<{ id: string }, {}, UpdateUserRequest>, res: Response<UserResponse>) => {
    const { name, email } = req.body;
    res.json({ id: 1, name: name || 'unknown', email: email || '' });
});

/**
 * deleteUser removes a user by ID.
 */
app.delete('/users/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});

app.listen(3000, () => {
    console.log('Server running on port 3000');
});
