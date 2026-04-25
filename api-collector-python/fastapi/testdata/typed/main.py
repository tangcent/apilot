from fastapi import FastAPI, Body, Query, Path
from pydantic import BaseModel
from typing import Optional, List, Dict

app = FastAPI()


class Address(BaseModel):
    street: str
    city: str
    zip_code: str


class User(BaseModel):
    name: str
    email: str
    age: int
    address: Address


class CreateUserReq(BaseModel):
    name: str
    email: str
    age: int


class UpdateUserReq(BaseModel):
    name: str
    email: str


class Item(BaseModel):
    id: int
    name: str
    price: float


class Order(BaseModel):
    order_id: str
    items: List[Item]
    total: float


class Result(BaseModel):
    code: int
    message: str
    data: str


class UserResult(BaseModel):
    code: int
    message: str
    data: User


class PaginatedUsers(BaseModel):
    total: int
    page: int
    items: List[User]


@app.get("/users", response_model=PaginatedUsers)
def list_users(name: Optional[str] = None):
    """listUsers returns all users."""
    pass


@app.post("/users", response_model=UserResult)
def create_user(req: CreateUserReq):
    """createUser creates a new user."""
    pass


@app.get("/users/{user_id}", response_model=UserResult)
def get_user(user_id: int = Path(...)):
    """getUser returns a single user by ID."""
    pass


@app.put("/users/{user_id}", response_model=UserResult)
def update_user(user_id: int = Path(...), req: UpdateUserReq = Body(...)):
    """updateUser updates an existing user."""
    pass


@app.delete("/users/{user_id}", response_model=Result)
def delete_user(user_id: int = Path(...)):
    """deleteUser removes a user by ID."""
    pass


@app.post("/orders", response_model=Order)
def create_order(req: Order):
    """createOrder creates a new order."""
    pass


@app.get("/items")
def list_items(category: str = Query(...)) -> List[Item]:
    """listItems returns all items in a category."""
    pass


@app.get("/config")
def get_config() -> Dict[str, str]:
    """getConfig returns application configuration."""
    pass
