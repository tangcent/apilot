from fastapi import FastAPI, APIRouter

app = FastAPI()
router = APIRouter()


@router.get("/items")
def list_items():
    """listItems returns all items."""
    pass


@router.delete("/items/{id}")
def delete_item(id: int):
    """deleteItem removes an item by ID."""
    pass


@app.get("/health")
def health_check():
    """healthCheck returns service health status."""
    pass


@router.post("/items")
def create_item():
    """createItem creates a new item."""
    pass


@router.get("/items/{id}")
def get_item(id: int):
    """getItem returns a single item by ID."""
    pass

app.include_router(router)
