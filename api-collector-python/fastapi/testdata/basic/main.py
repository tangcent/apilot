from fastapi import FastAPI, Path, Query, Header, Cookie, Body, File, UploadFile, Form
from typing import Optional

app = FastAPI()


@app.get("/users")
def list_users(name: str = Query(...), role: Optional[str] = Query(None)):
    """listUsers returns all users."""
    pass


@app.post("/users")
def create_user():
    """createUser creates a new user."""
    pass


@app.get("/users/{id}")
def get_user(id: int = Path(...)):
    """getUser returns a single user by ID."""
    pass


@app.put("/users/{id}")
def update_user(id: int = Path(...)):
    """updateUser updates an existing user."""
    pass


@app.delete("/users/{id}")
def delete_user(id: int = Path(...)):
    """deleteUser removes a user by ID."""
    pass


@app.patch("/users/{id}")
def patch_user(id: int = Path(...)):
    """patchUser partially updates a user."""
    pass


@app.head("/health")
def health_check():
    """healthCheck returns service health status."""
    pass


@app.options("/users")
def user_options():
    """userOptions returns allowed methods for /users."""
    pass


@app.post("/upload")
def upload_file(file: UploadFile = File(...), description: str = Form(...)):
    """uploadFile handles file uploads."""
    pass
