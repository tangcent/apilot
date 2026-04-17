from rest_framework.decorators import api_view
from rest_framework.response import Response


@api_view(['GET'])
def django_hello(request):
    """djangoHello returns a greeting from Django."""
    return Response({"message": "hello"})


@api_view(['DELETE'])
def django_delete_user(request, pk):
    """djangoDeleteUser deletes a user via Django."""
    return Response({"id": pk})
