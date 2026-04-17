const express = require('express');
const app = express();
const router = express.Router();

/**
 * healthCheck returns service health status.
 */
app.get('/health', healthCheck);

/**
 * listItems returns all items.
 */
router.get('/items', listItems);

/**
 * createItem creates a new item.
 */
router.post('/items', createItem);

/**
 * getItem returns a single item by ID.
 */
router.get('/items/:id', getItem);

/**
 * deleteItem removes an item by ID.
 */
router.delete('/items/:id', deleteItem);

app.use('/api', router);
app.listen(3000);
