const express = require('express');
const app = express();

app.use(express.json());

function someHelper() {
    return 'hello';
}

class UserService {
    get_users() {
        return [];
    }
}

app.listen(3000);
