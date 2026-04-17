const express = require('express');
const app = express();

app.use(express.json());

/**
 * listUsers returns all users.
 */
app.get('/users', listUsers);

/**
 * createUser creates a new user.
 */
app.post('/users', createUser);

/**
 * getUser returns a single user by ID.
 */
app.get('/users/:id', getUser);

/**
 * updateUser updates an existing user.
 */
app.put('/users/:id', updateUser);

/**
 * deleteUser removes a user by ID.
 */
app.delete('/users/:id', deleteUser);

/**
 * patchUser partially updates a user.
 */
app.patch('/users/:id', patchUser);

/**
 * healthCheck returns service health status.
 */
app.head('/health', healthCheck);

/**
 * userOptions returns allowed methods for /users.
 */
app.options('/users', userOptions);

app.listen(3000, () => {
    console.log('Server running on port 3000');
});
