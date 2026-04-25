from fastapi import FastAPI
from pydantic import BaseModel
from typing import Optional

app = FastAPI()


class BaseEntity(BaseModel):
    id: int
    created_at: str
    updated_at: str


class User(BaseEntity):
    name: str
    email: str


class Product(BaseEntity):
    name: str
    price: float


@app.post("/users", response_model=User)
def create_user(user: User):
    """createUser creates a new user."""
    pass


@app.post("/products", response_model=Product)
def create_product(product: Product):
    """createProduct creates a new product."""
    pass
