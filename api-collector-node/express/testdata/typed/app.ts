import { Request, Response } from 'express';

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
app.get('/users', (req: Request, res: Response<ListUsersResponse>) => {
    res.json({ users: [], total: 0 });
});

/**
 * createUser creates a new user.
 */
app.post('/users', (req: Request<{}, {}, CreateUserRequest>, res: Response<UserResponse>) => {
    const { name, email } = req.body;
    res.json({ id: 1, name, email });
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
app.put('/users/:id', (req: Request<{ id: string }, {}, CreateUserRequest>, res: Response<UserResponse>) => {
    const { name, email } = req.body;
    res.json({ id: 1, name, email });
});

/**
 * deleteUser removes a user by ID.
 */
app.delete('/users/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});
