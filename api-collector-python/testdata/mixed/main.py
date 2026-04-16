from fastapi import FastAPI

app = FastAPI()


@app.get("/fastapi/hello")
def fastapi_hello():
    """fastapiHello returns a greeting from FastAPI."""
    pass


@app.post("/fastapi/users")
def fastapi_create_user():
    """fastapiCreateUser creates a new user via FastAPI."""
    pass
