from django.http import JsonResponse
from django.views.decorators.csrf import csrf_exempt
import json

@csrf_exempt
def list_users(request):
    if request.method == 'GET':
        name = request.GET.get('name')
        role = request.GET.get('role', 'user')
        return JsonResponse({"users": []})

@csrf_exempt
def create_user(request):
    if request.method == 'POST':
        data = json.loads(request.body)
        return JsonResponse(data, status=201)

@csrf_exempt
def user_detail(request, user_id):
    if request.method == 'GET':
        return JsonResponse({"id": user_id})
    elif request.method == 'PUT':
        data = json.loads(request.body)
        return JsonResponse(data)
    elif request.method == 'DELETE':
        return JsonResponse({}, status=204)
    elif request.method == 'PATCH':
        name = request.GET.get('name', 'unknown')
        return JsonResponse({"id": user_id})
