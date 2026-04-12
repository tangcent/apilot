from fastapi import FastAPI, Query
from pydantic import BaseModel
from typing import Optional

app = FastAPI()

class CreateUserReq(BaseModel):
    name: str
    email: str

class UpdateUserReq(BaseModel):
    name: str
    email: str

@app.get("/users")
def list_users(name: Optional[str] = None, role: str = "user"):
    return {"users": []}

@app.post("/users")
def create_user(req: CreateUserReq):
    return req

@app.get("/users/{user_id}")
def get_user(user_id: str):
    return {"id": user_id}

@app.put("/users/{user_id}")
def update_user(user_id: str, req: UpdateUserReq):
    return req

@app.delete("/users/{user_id}")
def delete_user(user_id: str):
    return ""

@app.patch("/users/{user_id}")
def patch_user(user_id: str, name: str = "unknown"):
    return {"id": user_id}
