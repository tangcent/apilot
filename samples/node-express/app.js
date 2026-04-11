const express = require('express');
const app = express();

app.use(express.json());

app.get('/users', (req, res) => {
    const { name, role = 'user' } = req.query;
    res.json({ users: [] });
});

app.post('/users', (req, res) => {
    const { name, email } = req.body;
    res.status(201).json({ name, email });
});

app.get('/users/:id', (req, res) => {
    res.json({ id: req.params.id });
});

app.put('/users/:id', (req, res) => {
    const { name, email } = req.body;
    res.json({ name, email });
});

app.delete('/users/:id', (req, res) => {
    res.status(204).send();
});

app.patch('/users/:id', (req, res) => {
    const { name = 'unknown' } = req.query;
    res.json({ id: req.params.id });
});

app.listen(3000, () => {
    console.log('Server running on port 3000');
});
