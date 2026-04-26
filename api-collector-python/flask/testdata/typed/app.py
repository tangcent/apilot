from flask import Flask, Blueprint, request
from pydantic import BaseModel
from marshmallow import Schema, fields

app = Flask(__name__)


class UserCreate(BaseModel):
    name: str
    email: str
    age: int


class UserResponse(BaseModel):
    id: int
    name: str
    email: str


class ItemCreate(BaseModel):
    title: str
    price: float
    description: str = ""


class ItemResponse(BaseModel):
    id: int
    title: str
    price: float


class UserSchema(Schema):
    id = fields.Int()
    name = fields.Str(required=True)
    email = fields.Email(required=True)


class ItemSchema(Schema):
    id = fields.Int()
    title = fields.Str(required=True)
    price = fields.Float(required=True)


class NestedItemSchema(Schema):
    id = fields.Int()
    title = fields.Str(required=True)
    tags = fields.List(fields.Str())


@app.route('/users', methods=['POST'])
def create_user(body: UserCreate) -> UserResponse:
    """createUser creates a new user."""
    pass


@app.route('/users/<id>', methods=['GET'])
def get_user(id: int) -> UserResponse:
    """getUser returns a single user by ID."""
    pass


@app.route('/users/<id>', methods=['PUT'])
def update_user(id: int, body: UserCreate) -> UserResponse:
    """updateUser updates an existing user."""
    pass


@app.route('/items', methods=['POST'])
def create_item(body: ItemCreate) -> ItemResponse:
    """createItem creates a new item."""
    pass


@app.route('/items/<id>', methods=['GET'])
def get_item(id: int) -> ItemResponse:
    """getItem returns a single item by ID."""
    pass


@app.route('/marshmallow/users', methods=['POST'])
def create_user_marshmallow(body: UserSchema) -> UserSchema:
    """createUserMarshmallow creates a user using Marshmallow schema."""
    pass


@app.route('/marshmallow/items', methods=['GET'])
def list_items_marshmallow() -> ItemSchema:
    """listItemsMarshmallow lists items using Marshmallow schema."""
    pass


@app.route('/marshmallow/nested', methods=['GET'])
def get_nested_item() -> NestedItemSchema:
    """getNestedItem returns a nested item using Marshmallow schema."""
    pass


@app.route('/users/<id>/items', methods=['GET'])
def list_user_items(id: int) -> list[ItemResponse]:
    """listUserItems returns all items for a user."""
    pass


@app.route('/users/batch', methods=['POST'])
def batch_create_users(body: list[UserCreate]) -> list[UserResponse]:
    """batchCreateUsers creates multiple users."""
    pass


@app.route('/health', methods=['GET'])
def health_check() -> dict:
    """healthCheck returns service health status."""
    pass


@app.route('/no-type', methods=['POST'])
def no_type_endpoint():
    """noTypeEndpoint has no type annotations."""
    pass


@app.route('/optional-response', methods=['GET'])
def optional_response() -> UserResponse | None:
    """optionalResponse may return a user or None."""
    pass


bp = Blueprint('api', __name__, url_prefix='/api/v2')


@bp.route('/products', methods=['POST'])
def create_product(body: ItemCreate) -> ItemResponse:
    """createProduct creates a new product."""
    pass


@bp.route('/products/<id>', methods=['GET'])
def get_product(id: int) -> ItemResponse:
    """getProduct returns a product by ID."""
    pass
