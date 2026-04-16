from fastapi import FastAPI

app = FastAPI()


@app.get("/ping")
def ping():
    """ping returns pong."""
    pass
