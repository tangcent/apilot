from flask import Flask, Blueprint

app = Flask(__name__)


@app.route('/users')
def list_users():
    """listUsers returns all users."""
    pass


@app.route('/users', methods=['POST'])
def create_user():
    """createUser creates a new user."""
    pass


@app.route('/users/<id>', methods=['GET'])
def get_user(id):
    """getUser returns a single user by ID."""
    pass


@app.route('/users/<id>', methods=['PUT'])
def update_user(id):
    """updateUser updates an existing user."""
    pass


@app.route('/users/<id>', methods=['DELETE'])
def delete_user(id):
    """deleteUser removes a user by ID."""
    pass


@app.route('/users/<id>', methods=['PATCH'])
def patch_user(id):
    """patchUser partially updates a user."""
    pass


@app.route('/health', methods=['HEAD'])
def health_check():
    """healthCheck returns service health status."""
    pass


@app.route('/users', methods=['OPTIONS'])
def user_options():
    """userOptions returns allowed methods for /users."""
    pass


@app.route('/items/<item_id>/posts/<post_id>', methods=['GET'])
def get_post(item_id, post_id):
    """getPost returns a specific post."""
    pass


bp = Blueprint('api', __name__)


@bp.route('/products')
def list_products():
    """listProducts returns all products."""
    pass


@bp.route('/products/<id>', methods=['GET', 'POST'])
def product_detail(id):
    """productDetail handles GET and POST for a product."""
    pass
