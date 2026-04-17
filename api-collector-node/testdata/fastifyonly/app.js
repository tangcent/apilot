const fastify = require('fastify')({ logger: true });

/**
 * listItems returns all items.
 */
fastify.route({
    method: 'GET',
    url: '/items',
    handler: listItems
});

/**
 * createItem creates a new item.
 */
fastify.route({
    method: 'POST',
    url: '/items',
    handler: createItem
});

/**
 * getItem returns a single item by ID.
 */
fastify.route({
    method: 'GET',
    url: '/items/:id',
    handler: getItem
});

fastify.listen({ port: 3000 });
