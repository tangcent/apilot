from rest_framework.decorators import api_view
from rest_framework.response import Response

@api_view(['GET'])
def get_user(request):
    """Get user information."""
    return Response({"user": "John"})

@api_view(['GET', 'POST'])
def user_list(request):
    """List all users or create a new user."""
    if request.method == 'GET':
        return Response([])
    elif request.method == 'POST':
        return Response({"created": True})

@api_view(['PUT', 'DELETE'])
def user_detail(request, pk):
    """Update or delete a user."""
    return Response({"id": pk})
