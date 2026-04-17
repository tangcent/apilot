const express = require('express');
const app = express();

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

app.listen(3000);
