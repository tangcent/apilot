from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/users', methods=['GET'])
def list_users():
    name = request.args.get('name')
    role = request.args.get('role', 'user')
    return jsonify({"users": []})

@app.route('/users', methods=['POST'])
def create_user():
    data = request.get_json()
    return jsonify(data), 201

@app.route('/users/<user_id>', methods=['GET'])
def get_user(user_id):
    return jsonify({"id": user_id})

@app.route('/users/<user_id>', methods=['PUT'])
def update_user(user_id):
    data = request.get_json()
    return jsonify(data)

@app.route('/users/<user_id>', methods=['DELETE'])
def delete_user(user_id):
    return '', 204

@app.route('/users/<user_id>', methods=['PATCH'])
def patch_user(user_id):
    name = request.args.get('name', 'unknown')
    return jsonify({"id": user_id})

if __name__ == '__main__':
    app.run(port=5000)
