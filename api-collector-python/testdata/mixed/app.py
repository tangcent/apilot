from flask import Flask

app = Flask(__name__)


@app.route('/flask/hello')
def flask_hello():
    """flaskHello returns a greeting from Flask."""
    pass


@app.route('/flask/users/<id>', methods=['PUT'])
def flask_update_user(id):
    """flaskUpdateUser updates a user via Flask."""
    pass
